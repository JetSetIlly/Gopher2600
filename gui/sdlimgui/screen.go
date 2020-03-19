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
	pixelWidth = 2
	defScaling = 2.0
)

// screen implements television.PixelRenderer
type screen struct {
	img *SdlImgui

	// pixels and altPixels should be constructed exactly the same. the only
	// difference is the colors
	pixels    *image.RGBA
	altPixels *image.RGBA

	// in addition to the unmasked pixel array we also maintain and draw to a
	// smaller pixel array that represents the masked screen. ideally, we would
	// only have the masked pixel array defined above, and to draw only a
	// selected group of pixels when drawing a masked screen. however, there's
	// no good way of doing this because gl.TexSubImage2d() expects the pixels
	// in the pixel array to be contiguous. this seems wasteful (and possibly
	// is) but it is easier and ultimately quicker to maintain two sets of
	// arrays (according to my current understanding that is)
	//
	// this is obviously slower than writing to one set of pixels but not
	// noticably so when SetRGBA() is used (rather than Set() which includes a
	// needless conversion to RGBA format)
	//
	// why not just write to one set or the other depending on whether masking
	// is activated or not? because we want to be able to flip between masked
	// and unmapsed displays even when paused.
	//
	// would it be better to have two textures one which is "full" size and one
	// which "zooms" on the pixels in the non-masked area of the screen? maybe,
	// but it seems messy to me by comparison.
	maskedPixels    *image.RGBA
	maskedAltPixels *image.RGBA

	// which set of pixels to use: masked or unmasked
	masked bool

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value
	scaling float32

	// aspect bias is taken from the television specification
	aspectBias float32

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int

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
		masked:  true,
	}

	// generate texture, creation of texture will be done on first call to
	// render()
	gl.GenTextures(1, &scr.texture)
	gl.BindTexture(gl.TEXTURE_2D, scr.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

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

	scr.pixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))
	scr.altPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))

	scr.maskedPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksVisible, scr.scanlines))
	scr.maskedAltPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksVisible, scr.scanlines))

	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	scr.setWindow(reapplyScale)

	// defer re-creation of texture to render(). we have to do it in the
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

	scr.lastX = x
	scr.lastY = y

	scr.pixels.SetRGBA(scr.lastX, scr.lastY,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})
	scr.maskedPixels.SetRGBA(scr.lastX-television.HorizClksHBlank, scr.lastY-scr.topScanline,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})

	return nil
}

// SetAltPixel implements the television.PixelRenderer interface
func (scr *screen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	scr.altPixels.SetRGBA(x, y,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})
	scr.maskedAltPixels.SetRGBA(x-television.HorizClksHBlank, y-scr.topScanline,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})

	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}

func (scr *screen) scaledWidth() float32 {
	return float32(scr.maskedPixels.Bounds().Size().X*pixelWidth) * scr.aspectBias * scr.scaling
}

func (scr *screen) scaledHeight() float32 {
	return float32(scr.maskedPixels.Bounds().Size().Y) * scr.scaling
}

// render is called by service loop
func (scr *screen) render() {
	var pixels *image.RGBA
	if scr.useAltPixels {
		if scr.masked {
			pixels = scr.maskedAltPixels
		} else {
			pixels = scr.altPixels
		}
	} else {
		if scr.masked {
			pixels = scr.maskedPixels
		} else {
			pixels = scr.pixels
		}
	}

	// if frame rate is below a given threshold then fake a pause image. we
	// don't want to do this with too high of a threshold though because it
	// would just look like wierd
	var pixelsCp []uint8
	if scr.img.lazy.TV.ReqFPS < 3.0 {
		copy(pixelsCp, pixels.Pix)
		scr.pause(true)
	}

	if scr.createTexture {
		scr.createTexture = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	} else {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}

	// undo fake pause image
	if scr.img.lazy.TV.ReqFPS < 3.0 {
		copy(pixels.Pix, pixelsCp)
	}
}

func (scr *screen) pause(set bool) {

	// when emulation is paused, process the current pixel data to
	// differentiate "old" pixels (from previous frame) and "new" pixels (drawn
	// this frame)
	if set {

		// do no fade image if we're still on the first scanline after a new
		// frame. this is to prevent the display being faded after a STEP
		// FRAME. the user wouldn't expect the image to be faded after asking
		// to step forward one frame
		if scr.lastY == 0 {
			return
		}

		// pixel offset for last x/y coordinates. we're going to assume that
		// the scr.pixels and scr.altPixels array are constructed exactyle the
		// same (reasonable assumption)
		o := scr.pixels.PixOffset(scr.lastX, scr.lastY)
		if o >= 0 && o < len(scr.pixels.Pix) {

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

		// similar process for masked pixels. some care is required when
		// finding the starting offset for array traversal

		x := scr.lastX - television.HorizClksHBlank
		if x < 0 {
			x = 0
		}

		y := scr.lastY - scr.topScanline
		if y < 0 {
			// the y pixel is outside (and above) the masked display so
			// logically the x pixel must be as well
			y = 0
			x = 0
		}

		o = scr.maskedPixels.PixOffset(x, y)
		if o >= 0 && o < len(scr.maskedPixels.Pix) {
			for i := 0; i <= o; i += 4 {
				scr.maskedPixels.Pix[i+3] = 255
				scr.maskedAltPixels.Pix[i+3] = 255
			}

			for i := o + 4; i < len(scr.maskedPixels.Pix); i += 4 {
				scr.maskedPixels.Pix[i+3] = 100
				scr.maskedAltPixels.Pix[i+3] = 100
			}
		}
	}
}

func (scr *screen) setMasking(set bool) {
	scr.masked = set
	scr.createTexture = true
}
