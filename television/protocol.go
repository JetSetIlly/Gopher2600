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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package television

import "strings"

// Television defines the operations that can be performed on the conceptual
// television. Note that the television implementation itself does not present
// any information, either visually or sonically. Instead, PixelRenderers and
// AudioMixers are added to perform those tasks.
type Television interface {
	String() string

	// Reset the television to an initial state
	Reset() error

	// AddPixelRenderer registers an (additional) implementation of PixelRenderer
	AddPixelRenderer(PixelRenderer)

	// AddAudioMixer registers an (additional) implementation of AudioMixer
	AddAudioMixer(AudioMixer)

	Signal(SignalAttributes) error

	// Returns the value of the requested state. eg. the current scanline.
	GetState(StateReq) (int, error)

	// Set the television's specification
	SetSpec(spec string) error

	// Returns the television's current specification. Renderers should use
	// GetSpec() rather than keeping a private pointer to the specification.
	GetSpec() *Specification

	// IsStable returns true if the television thinks the image being sent by
	// the VCS is stable
	IsStable() bool

	// some televisions may need to conclude and/or dispose of resources
	// gently. implementations of End() should call EndRendering() and
	// EndMixing() on each PixelRenderer and AudioMixer that has been added.
	//
	// for simplicity, the Television should be considered unusable
	// after EndRendering() has been called
	End() error

	// SpecIDOnCreation() returns the string that was to ID the television
	// type/spec on creation. because the actual spec can change, the ID field
	// of the Specification type can not be used for things like regression
	// test recreation etc.
	//
	// we use this to help recreate the television that was used to make a
	// playback recording. we may need to expand on this (and maybe replace
	// with a more generalised function) if we ever add another television
	// implementation.
	SpecIDOnCreation() string

	// Set whether the emulation should wait for FPS limiter
	SetFPSCap(set bool)

	// Request the number frames per second. This overrides the frame rate of
	// the specification. A negative FPS value restores the specifcications
	// frame rate.
	//
	// Note that this is only a request, the emulation may not be able to
	// achieve that rate.
	SetFPS(fps float32)

	// The requested number of frames per second. Compare with GetActualFPS()
	// to check for accuracy
	GetReqFPS() float32

	// The current number of frames per second
	GetActualFPS() float32

	// Returns a copy of SignalAttributes for reference
	GetLastSignal() SignalAttributes
}

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
	// It may be called when television specification has changed. Renderers
	// should use GetSpec() rather than keeping a private pointer to the
	// specification.
	//
	// Renderers should use the values sent by the Resize() function, rather
	// than the equivalent values in the specification. Unless of course, the
	// renderer is intended to be strict about specification accuracy.
	//
	// Renderers should also make sure that any data structures that depend on
	// the specification being used are still adequate.
	//
	// Renderers should consider how to handle increased scanlines. It's quite
	// common for ROMs to need a few extra scanlines after a couple of screens,
	// hardly worth resizing the display window for. A good strategy is to
	// resize the display only when the screen in unstable. Otherwise, it is
	// better that additional scanlines should be squeezed into the available
	// physical space (see ladybug and tapper for good exampls of ROMs)
	//
	// Of course, there's a point when the squeezing becomes too much. There
	// are no good examples of ROMs that are excessive in the use of scanlines
	// but it's something to consider.
	Resize(topScanline, visibleScanlines int) error

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
	SetPixel(x, y int, red, green, blue byte, vblank bool) error

	// some renderers may need to conclude and/or dispose of resources gently.
	// for simplicity, the PixelRenderer should be considered unusable after
	// EndRendering() has been called
	EndRendering() error
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

// ColorSignal represents the signal that is sent from the VCS to the
type ColorSignal int

// VideoBlack is the PixelSignal value that indicates no VCS pixel is to be shown
const VideoBlack ColorSignal = -1

// SignalAttributes represents the data sent to the television
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
// with the GetState() function
type StateReq int

// List of valid state requests
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
)
