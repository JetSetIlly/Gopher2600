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
	// Renderers must be prepared to resize to either a smaller or larger size.
	//
	// Resize can also be called speculatively so implementations should take
	// care not to perform any resizing unless absolutely necessary.
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

	// SetPixels sends a slice of SignalAttributes to the Renderer.
	//
	// The television is guaranteed to only send signal.NoSignal as trailing
	// filler. For these signals the corresponding pixels should be set to
	// black (or if appropriate, ignored).
	//
	// For renderers that are producing an accurate visual image, the pixel
	// should always be set to video black if VBLANK is on. Some renderers
	// however may find it useful to set the pixel to the RGB value regardless
	// of VBLANK.
	//
	// A very important note is that some ROMs use VBLANK to control pixel
	// color within the visible display area. For example:
	//
	//	* Custer's Revenge
	//	* Ladybug
	//	* ET (turns VBLANK off late on scanline 40)
	//
	// In other words, the PixelRenderer should not simply assume VBLANK is
	// restricted to the "off-screen" areas as defined by the FrameInfo sent to
	// Resize()
	//
	// The current flag states that the signals should be considered to be
	// signals for the current frame. for most applications this will always be
	// true but in some circumstances, the television will send pixels from the
	// previous frame if they haven't been drawn yet for the current frame. An
	// implementation of PixelRenderer may choose to ignore non-current
	// signals.
	SetPixels(sig []signal.SignalAttributes, current bool) error

	// Reset all pixels. Called when TV is reset.
	//
	// Note that a Reset event does not imply a Resize() event. Implementations
	// should not call the Resize() function as a byproduct of a Reset(). The
	// television will send an explicit Resize() request if it is appropriate.
	Reset()

	// Some renderers may need to conclude and/or dispose of resources gently.
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
	// for efficiency reasons, SetAudio() implementations can be sent
	// SignalAttributes values that do not have valid AudioData (ie.
	// AudioUpdate bit is zero). implemenations should therefore take care when
	// processing the sig slice.
	//
	// the general algorithm for processing the sig slice is:
	//
	//	for _, s := range sig {
	//		if s&signal.AudioUpdate != signal.AudioUpdate {
	//			continue
	//		}
	//		d := uint8((s & signal.AudioData) >> signal.AudioDataShift)
	//
	//		...
	//	}
	SetAudio(sig []signal.SignalAttributes) error

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
