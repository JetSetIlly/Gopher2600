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
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// PixelRenderer implementations displays, or otherwise works with, visual
// information from a television. For example digest.Video
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
// GetFrameInfo() function
type PixelRenderer interface {
	// NewFrame is called at the start of a new frame
	//
	// Renderers should be prepared to resize the rendering display to either a
	// smaller or larger frame size. Renderers should also expect NewFrame to be
	// called multiple times but not necesssarily with NewScanline() or
	// SetPixels() being called
	NewFrame(frameinfo.Current) error

	// NewScanline is called at the start of a new scanline
	NewScanline(scanline int) error

	// SetPixels is used to Render a series of signals. The number of signals
	// will always be television.MaxSignalHistory
	//
	// Producing a 2d image from the signals sent by SetPixels() can easily be
	// done by first allocating a bitmap of width specification.ClksScanline
	// and height specification.AbsoluateMaxScanlines. This bitmap will have
	// television.MaxSignalHistory entries
	//
	// Every signal from SetPixels() therefore corresponds to a pixel in the
	// bitmap - the first entry always referes to the top-left pixel
	//
	// If the entry contains signal.NoSignal then that screen pixel has not been
	// written to recently. However, the bitmap may still need to be updated
	// with "nil" information if the size of the screen has reduced
	//
	// For renderers that are producing an accurate visual image, the pixel
	// should always be set to video black if VBLANK is on. Some renderers
	// however may find it useful to set the pixel to the RGB value regardless
	// of VBLANK
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
	SetPixels(sig []signal.SignalAttributes, last int) error

	// Reset all pixels. Called when TV is reset.
	//
	// Note that a Reset event does not imply a Resize() event. Implementations
	// should not call the Resize() function as a byproduct of a Reset(). The
	// television will send an explicit Resize() request if it is appropriate
	Reset()

	// Some renderers may need to conclude and/or dispose of resources gently
	EndRendering() error
}

// PixelRendererDisplay is a special case of the the PixelRenderer interface. it should be
// implemented by a renderer that works with a hardware display.
type PixelRendererDisplay interface {
	PixelRenderer

	// returns refresh rate of display and whether the limiter should quantise the emulated frame
	// rate to the monitor speed. the boolean indicates whether the current purpose of the display
	// is suitable for quantisation by a frame limiter
	DisplayRefreshRate() (float32, bool)
}

// PixelRendererRotation is an extension to the PixelRenderer interface. Pixel
// renderes that implement this interface can show the television image in a
// rotated aspect. Not all pixel renderers need to worry about rotation.
type PixelRendererRotation interface {
	SetRotation(specification.Rotation)
}

// PixelRendererFPSLimiter is an extension to the PixelRenderer interface. Pixel
// renderers that implement this interface will be notified when the
// television's frame capping policy is changed. Not all pixel renderers need to
// worry about frame rate.
type PixelRendererFPSLimiter interface {
	SetFPSLimit(limit bool)
}

// FrameTrigger implementations listen for Pause events
type PauseTrigger interface {
	Pause(pause bool) error
}

// FrameTrigger implementations listen for NewFrame events. FrameTrigger is a
// subset of PixelRenderer
type FrameTrigger interface {
	NewFrame(frameinfo.Current) error
}

// ScanlineTrigger implementations listen for NewScanline events. It is a
// subset of PixelRenderer
type ScanlineTrigger interface {
	NewScanline(frameinfo.Current) error
}

// AudioMixer implementations work with sound; most probably playing it. An
// example of an AudioMixer that does not play sound but otherwise works with
// it is the digest.Audio type
type AudioMixer interface {
	// for efficiency reasons, SetAudio() implementations can be sent
	// SignalAttributes values that do not have valid AudioData (ie.
	// AudioUpdate bit is zero). implemenations should therefore take care when
	// processing the sig slice
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
	SetAudio(sig []signal.AudioSignalAttributes) error

	// some mixers may need to conclude and/or dispose of resources gently.
	// for simplicity, the AudioMixer should be considered unusable after
	// EndMixing() has been called
	EndMixing() error

	// Reset buffered audio and anything else that might need doing on, for
	// example, a cartridge change
	Reset()
}

// RealtimeAudioMixer should be implemented by audio mixers that are sensitive
// to the refresh rate of the console/television
type RealtimeAudioMixer interface {
	AudioMixer

	// Notifies the mixer of the basic refresh rate of the television. If the
	// refresh rate is significantly outside the nominal rate for the
	// specification then the audio queue will be suboptimal
	SetSpec(specification.Spec)
}

// VCS is used to send information from the TV back to the parent console
type VCS interface {
	SetClockSpeed(specification.Spec)
}

// Interface to a developer helper that can cause the emulation to halt on
// various television related conditions
type Debugger interface {
	HaltFromTelevision(reason string)
}
