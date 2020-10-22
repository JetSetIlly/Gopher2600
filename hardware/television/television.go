// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package television

import "strings"

// Television defines the operations that can be performed on the conceptual
// television. Note that the television implementation itself does not present
// any information, either visually or sonically. Instead, PixelRenderers and
// AudioMixers are added to perform those tasks.
type Television interface {
	String() string

	// Make a copy of the television state
	Snapshot() TelevisionState

	// Copy back television state
	RestoreSnapshot(TelevisionState)

	// AddPixelRenderer registers an (additional) implementation of PixelRenderer
	AddPixelRenderer(PixelRenderer)

	// AddFrameTrigger registers an (additional) implementation of FrameTrigger
	AddFrameTrigger(FrameTrigger)

	// AddAudioMixer registers an (additional) implementation of AudioMixer
	AddAudioMixer(AudioMixer)

	// Reset the television to an initial state
	Reset() error

	// some televisions may need to conclude and/or dispose of resources
	// gently. implementations of End() should call EndRendering() and
	// EndMixing() on each PixelRenderer and AudioMixer that has been added.
	//
	// for simplicity, the Television should be considered unusable
	// after EndRendering() has been called
	End() error

	Signal(SignalAttributes) error

	// \/\/ reflection/information \/\/

	// IsStable returns true if the television thinks the image being sent by
	// the VCS is stable
	IsStable() bool

	// Returns a copy of SignalAttributes for reference
	GetLastSignal() SignalAttributes

	// Returns state information
	GetState(StateReq) (int, error)

	// \/\/ specfication \/\/
	//
	// Set the television's specification
	SetSpec(spec string) error

	// GetReqSpecID returns the specification that was requested on creation
	GetReqSpecID() string

	// Returns the television's current specification. Renderers should use
	// GetSpec() rather than keeping a private pointer to the specification.
	GetSpec() Spec

	// \/\/ FPS \/\/

	// Set whether the emulation should wait for FPS limiter
	SetFPSCap(set bool)

	// Request the number frames per second. This overrides the frame rate of
	// the specification. A negative  value restores the spec's frame rate.
	SetFPS(fps float32)

	// The requested number of frames per second. Compare with GetActualFPS()
	// to check for accuracy
	GetReqFPS() float32

	// The current number of frames per second
	GetActualFPS() float32
}

// TelevisionTIA exposes only the functions required by the TIA.
type TelevisionTIA interface {
	Signal(SignalAttributes) error
	GetState(StateReq) (int, error)
}

// TelevisionSprite exposes only the functions required by the video sprites.
type TelevisionSprite interface {
	GetState(StateReq) (int, error)
}

// TelevisionState is a deliberately opaque type returned by Snapshot() and
// used by RestoreSnapshot() in the Television interface. The state itself can
// consist of anything necessary to the Television implementation.
type TelevisionState interface{}

// PixelRenderer implementations displays, or otherwise works with, visual
// information from a television. For example digest.Video.
//
// PixelRenderer implementations often find it convenient to maintain a reference to
// the parent Television implementation and maybe even embed the Television
// interface. ie.
//
//	type ExampleTV struct {
//		television.Television
//		...
//	}
type PixelRenderer interface {
	// Resize is called when the television implementation detects that extra
	// scanlines are required in the display.
	//
	// It may be called when television specification has changed. As a point
	// of convenience a reference to the currently selected specification is
	// provided. However, renderers should call GetSpec() rather than keeping a
	// private pointer to the specification, if knowledge of the spec is
	// required after the Resize() event.
	//
	// Renderers should use the values sent by the Resize() function, rather
	// than the equivalent values in the specification. Unless of course, the
	// renderer is intended to be strict about specification accuracy.
	//
	// Renderers should make sure that any data structures that depend on the
	// specification being used are still adequate.
	Resize(spec Spec, topScanline, visibleScanlines int) error

	// NewFrame and NewScanline are called at the start of the frame/scanline
	NewFrame(frameNum int, isStable bool) error
	NewScanline(scanline int) error

	// setPixel() is called every cycle regardless of the state of VBLANK and
	// HBLANK.
	//
	// things to consider:
	//
	// o the x argument is measured from zero so renderers should decide how to
	//	handle pixels of during the HBLANK (x < ClocksPerHBLANK)
	//
	// o the y argument is also measured from zero but because VBLANK can be
	//	turned on at any time there's no easy test. the VBLANK flag is sent to
	//	help renderers decide what to do.
	//
	// o for renderers that are producing an accurate visual image, the pixel
	//	should always be set to video black if VBLANK is on.
	//
	//	some renderers however, may find it useful to set the pixel to the RGB
	//	value regardless of VBLANK. for example, DigestTV does this.
	//
	//	a vey important note is that some ROMs use VBLANK to control pixel
	//	color within the visible display area. ROMs affected:
	//
	//	* Custer's Revenge
	//	* Ladybug
	//	* ET (turns VBLANK off late on scanline 40)
	//
	// Set refreshing flag to true when called between Refresh(true) and
	// Refresh(false)
	SetPixel(x, y int, red, green, blue byte, vblank bool, refreshing bool) error

	// some renderers may need to conclude and/or dispose of resources gently.
	// for simplicity, the PixelRenderer should be considered unusable after
	// EndRendering() has been called
	EndRendering() error

	// Mark the start and end of a refresh event from the television. These
	// events occur when the television wants to dump many pixels at once.
	// Use SetPixel() with a refreshing flag of true between calls to
	// Refresh(true) and Refresh(false)
	Refresh(refreshing bool)
}

// FrameTrigger implementations listen for NewFrame events. FrameTrigger is a
// subset of PixelRenderer.
type FrameTrigger interface {
	NewFrame(frameNum int, isStable bool) error
}

// AudioMixer implementations work with sound; most probably playing it. An
// example of an AudioMixer that does not play sound but otherwise works with
// it is the digest.Audio type.
type AudioMixer interface {
	SetAudio(audioData uint8) error

	// some mixers may need to conclude and/or dispose of resources gently.
	// for simplicity, the AudioMixer should be considered unusable after
	// EndMixing() has been called
	EndMixing() error
}

// ColorSignal represents the signal that is sent from the VCS to the.
type ColorSignal int

// VideoBlack is the PixelSignal value that indicates no VCS pixel is to be shown.
const VideoBlack ColorSignal = -1

// SignalAttributes represents the data sent to the television.
type SignalAttributes struct {
	VSync     bool
	VBlank    bool
	CBurst    bool
	HSync     bool
	Pixel     ColorSignal
	AudioData uint8

	// which equates to 30Khz
	AudioUpdate bool
}

func (a SignalAttributes) String() string {
	s := strings.Builder{}
	if a.VSync {
		s.WriteString("VSYNC ")
	}
	if a.VBlank {
		s.WriteString("VBLANK ")
	}
	if a.CBurst {
		s.WriteString("CBURST ")
	}
	if a.HSync {
		s.WriteString("HSYNC ")
	}
	return s.String()
}

// StateReq is used to identify which television attribute is being asked
// with the GetState() function.
type StateReq int

// List of valid state requests.
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
)
