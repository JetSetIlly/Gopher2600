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
	Resize(spec specification.Spec, topScanline, visibleScanlines int) error

	// NewFrame and NewScanline are called at the start of the frame/scanline
	NewFrame(frameNum int, isStable bool) error
	NewScanline(scanline int) error

	// Mark the start and end of an update event from the television.
	// SetPixel() should only be called between calls of UpdatingPixels(true)
	// and UpdatingPixels(false)
	UpdatingPixels(updating bool)

	// SetPixel() is called every cycle regardless of the state of VBLANK and
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
	// current flag states that this pixel should be considered to be the most
	// recent outputted by the television for this frame. In most instances,
	// this will always be true.
	SetPixel(sig signal.SignalAttributes, current bool) error

	// some renderers may need to conclude and/or dispose of resources gently.
	// for simplicity, the PixelRenderer should be considered unusable after
	// EndRendering() has been called
	EndRendering() error
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

type ReflectionSynchronising interface {
	SetPendingReflectionPixel(idx int) error
	NewFrame()
}

// the maximum number of scanlines allowed by the television implementation.
const MaxScanlinesAbsolute = 400

// the number of entries in signal history.
const MaxSignalHistory = specification.HorizClksScanline * MaxScanlinesAbsolute
