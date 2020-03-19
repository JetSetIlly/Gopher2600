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

package sdlimgui

import (
	"gopher2600/television"
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.2-core/gl"
)

const (
	pixelDepth = 3
	pixelWidth = 2
	defScaling = 2.0
)

// screen implements television.PixelRenderer
type screen struct {
	img *SdlImgui
	tv  television.Television

	// pixels and altPixels should be constructed exactly the same. the only
	// difference is the colors
	pixels    *image.RGBA
	altPixels *image.RGBA

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value
	scaling float32

	// aspect bias is taken from the television specification
	aspectBias float32

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int
	horizPixels int

	// create texture on the next call of render
	createTexture bool

	// the tv screen texture and backing pixels
	texture uint32

	// the coordinates of the last SetPixel(). used to help set the alpha
	// channel when emulation is paused
	lastX int
	lastY int

	// whether to use the alternative pixel layer
	useAltPixels bool
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:     img,
		scaling: defScaling,

		// horizPixels is always the same regardless of tv spec
		horizPixels: television.HorizClksVisible,
	}

	// start off by showing entirity of NTSC screen
	scr.resize(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	return scr
}

// Resize implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) Resize(topScanline int, visibleScanlines int) error {
	scr.img.service <- func() {
		scr.img.serviceErr <- scr.resize(topScanline, visibleScanlines)
	}
	return <-scr.img.serviceErr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *screen) resize(topScanline int, visibleScanlines int) error {
	scr.topScanline = topScanline
	scr.scanlines = visibleScanlines
	scr.pixels = image.NewRGBA(image.Rect(0, 0, scr.horizPixels, scr.scanlines))
	scr.altPixels = image.NewRGBA(image.Rect(0, 0, scr.horizPixels, scr.scanlines))

	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	scr.setWindow(reapplyScale)

	// defer recreation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	scr.createTexture = true

	return nil
}

// the value to use
const reapplyScale = -1.0

// MUST ONLY be called from the #mainthread
func (scr *screen) setWindow(scale float32) error {
	if scale != reapplyScale {
		scr.scaling = scale
	}

	return nil
}

// MUST NOT be called from the #mainthread
// see setWindow() for non-main alternative
func (scr *screen) setWindowFromThread(scale float32) error {
	scr.img.service <- func() {
		scr.setWindow(scale)
		scr.img.serviceErr <- nil
	}
	return <-scr.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) NewFrame(frameNum int) error {
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (scr *screen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	// handle VBLANK by setting pixels to black
	if vblank {
		red = 0
		green = 0
		blue = 0
	}

	scr.lastX = x - television.HorizClksHBlank
	scr.lastY = y - scr.topScanline

	scr.pixels.Set(scr.lastX, scr.lastY,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})

	return nil
}

// SetAltPixel implements the television.PixelRenderer interface
func (scr *screen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	scr.altPixels.Set(x-television.HorizClksHBlank, y-scr.topScanline,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})

	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}

func (scr *screen) scaledWidth() float32 {
	return float32(scr.pixels.Bounds().Size().X*pixelWidth) * scr.aspectBias * scr.scaling
}

func (scr *screen) scaledHeight() float32 {
	return float32(scr.pixels.Bounds().Size().Y) * scr.scaling
}

// render is called by service loop
func (scr *screen) render() {
	var pixels *image.RGBA
	if scr.useAltPixels {
		pixels = scr.altPixels
	} else {
		pixels = scr.pixels
	}

	gl.BindTexture(gl.TEXTURE_2D, scr.texture)

	if scr.createTexture {
		scr.createTexture = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	} else {
		gl.BindTexture(gl.TEXTURE_2D, scr.texture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}
}

func (scr *screen) pause(set bool) {
	// when emulation is paused, process the current pixel data to
	// differentiate "old" pixels (from previous frame) and "new" pixels (drawn
	// this frame)
	if set {
		// pixel offset for last x/y coordinates. we're going to assume that
		// the scr.pixels and scr.altPixels array are constructed exactyle the
		// same (reasonable assumption)
		o := scr.pixels.PixOffset(scr.lastX, scr.lastY)
		if o < 0 || o >= len(scr.pixels.Pix) {
			return
		}

		// make sure all pixels from current frame have full alpha value
		for i := 0; i <= o; i += 4 {
			scr.pixels.Pix[i+3] = 255
			scr.altPixels.Pix[i+3] = 255
		}

		// make sure old pixels are faded
		for i := o + 4; i < len(scr.pixels.Pix); i += 4 {
			scr.pixels.Pix[i+3] = 100
			scr.altPixels.Pix[i+3] = 100
		}
	}
}
