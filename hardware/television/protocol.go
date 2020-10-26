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

// TelevisionTIA exposes only the functions required by the TIA.
type TelevisionTIA interface {
	Signal(SignalAttributes) error
	GetState(StateReq) int
}

// TelevisionSprite exposes only the functions required by the video sprites.
type TelevisionSprite interface {
	GetState(StateReq) int
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
	SetPixel(x, y int, red, green, blue byte, vblank bool) error

	// some renderers may need to conclude and/or dispose of resources gently.
	// for simplicity, the PixelRenderer should be considered unusable after
	// EndRendering() has been called
	EndRendering() error
}

// PixelRefresher implementations are prepared to accept pixels outside of the
// normal PixelRenderer sequence.
type PixelRefresher interface {
	// Mark the start and end of a refresh event from the television. These
	// events occur when the television wants to dump many pixels at once.
	// Use SetPixel() with a refreshing flag of true between calls to
	// Refresh(true) and Refresh(false)
	Refresh(refreshing bool)

	// RefreshPixel should only be called between two call of Refresh() as
	// described above
	RefreshPixel(x, y int, red, green, blue byte, vblank bool, stale bool) error
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

	// the position on the screen this signal was applied to. added by the
	// television implementation.
	horizPos int
	scanline int
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
