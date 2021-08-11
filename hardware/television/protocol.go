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

import (
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

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
//
// The most useful source of information though is the FrameInfo type supplied
// to the PixelRenderer through the Resize() and NewFrame() functions. A
// current copy of this information is also available from the television type
// GetFrameInfo() function.
type PixelRenderer interface {
	// Resize is called when the television implementation detects that extra
	// scanlines are required in the display.
	//
	// Renderers must be prepared to resize to either a smaller of larger size.
	//
	// The VisibleTop and VisibleBottom fields in the FrameInfor argument,
	// describe the top and bottom scanline that is *visible* on a normal
	// screen. pixels outside this range that are sent by SetPixel() can be
	// handled according to the renderers needs but would not normally be shown
	// for game-playing purposes.
	Resize(FrameInfo) error

	// NewFrame is called at the start of a new scanline.
	//
	// PixelRenderer implementations should consider what to do when a
	// non-synced frame is submitted. Rolling the screen is a good response to
	// the non-synced frame, with the possiblity of a one or two tolerance (ie.
	// do not roll unless the non-sync frame are continuous)
	NewFrame(FrameInfo) error

	// NewScanline is called at the start of a new scanline
	NewScanline(scanline int) error

	// Mark the start and end of an update event from the television.
	// SetPixel() should only be called between calls of UpdatingPixels(true)
	// and UpdatingPixels(false)
	UpdatingPixels(updating bool)

	// SetPixel sends an instance of SignalAttributes to the Renderer. The
	// current flag states that this pixel should be considered to be the most
	// recent outputted by the television for this frame. In most instances,
	// this will always be true.
	//
	// things to consider:
	//
	// o the x argument is measured from zero so renderers should decide how to
	//	handle pixels of during the HBLANK (x < ClksHBlank)
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
	SetPixel(sig signal.SignalAttributes, current bool) error
	SetPixels(sig []signal.SignalAttributes, current bool) error

	// Reset all pixels. Called when TV is reset.
	//
	// Note that a Reset event does not imply a Resize() event. Implementations
	// should not call the Resize() function as a byproduct of a Reset().
	Reset()

	// Some renderers may need to conclude and/or dispose of resources gently.
	// for simplicity, the PixelRenderer should be considered unusable after
	// EndRendering() has been called.
	EndRendering() error
}

// FrameTrigger implementations listen for Pause events.
type PauseTrigger interface {
	Pause(pause bool) error
}

// FrameTrigger implementations listen for NewFrame events. FrameTrigger is a
// subset of PixelRenderer.
type FrameTrigger interface {
	// See NewFrame() comment for PixelRenderer interface.
	NewFrame(FrameInfo) error
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

	// Reset buffered audio and anything else that might need doing on, for
	// example, a cartridge change.
	Reset()
}

// MaxSignalHistory is the absolute maximum number of entries in a signal history for an entire frame.
const MaxSignalHistory = specification.ClksScanline * specification.AbsoluteMaxScanlines

// VCSReturnChannel is used to send information from the TV back to the parent
// console. Named because I think of it as being similar to the Audio Return
// Channel (ARC) present in modern TVs.
type VCSReturnChannel interface {
	SetClockSpeed(tvSpec string) error
}
