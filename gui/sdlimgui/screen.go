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
	"gopher2600/test"
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

	pixels *image.RGBA

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
func (win *screen) Resize(topScanline int, visibleScanlines int) error {
	win.img.service <- func() {
		win.img.serviceErr <- win.resize(topScanline, visibleScanlines)
	}
	return <-win.img.serviceErr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (win *screen) resize(topScanline int, visibleScanlines int) error {
	win.topScanline = topScanline
	win.scanlines = visibleScanlines
	win.pixels = image.NewRGBA(image.Rect(0, 0, win.horizPixels, win.scanlines))

	win.aspectBias = win.img.tv.GetSpec().AspectBias

	win.setWindow(reapplyScale)

	// defer recreation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	win.createTexture = true

	return nil
}

const reapplyScale = -1.0

// MUST ONLY be called from the #mainthread
func (win *screen) setWindow(scale float32) error {
	test.AssertMainThread()

	if scale != reapplyScale {
		win.scaling = scale
	}

	return nil
}

// MUST NOT be called from the #mainthread
// see setWindow() for non-main alternative
func (win *screen) setWindowFromThread(scale float32) error {
	test.AssertNonMainThread()

	win.img.service <- func() {
		win.setWindow(scale)
		win.img.serviceErr <- nil
	}
	return <-win.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (win *screen) NewFrame(frameNum int) error {
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (win *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (win *screen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {

	// handle VBLANK by setting pixels to black
	if vblank {
		red = 0
		green = 0
		blue = 0
	}

	win.pixels.Set(x-television.HorizClksHBlank, y-win.topScanline,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})

	return nil
}

// SetAltPixel implements the television.PixelRenderer interface
func (win *screen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (win *screen) EndRendering() error {
	return nil
}

func (win *screen) scaledWidth() float32 {
	return float32(win.pixels.Bounds().Size().X*pixelWidth) * win.aspectBias * win.scaling
}

func (win *screen) scaledHeight() float32 {
	return float32(win.pixels.Bounds().Size().Y) * win.scaling
}

// render is called by service loop
func (win *screen) render() {
	gl.BindTexture(gl.TEXTURE_2D, win.texture)

	if win.createTexture {
		win.createTexture = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(win.pixels.Bounds().Size().X), int32(win.pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(win.pixels.Pix))
	} else {
		gl.BindTexture(gl.TEXTURE_2D, win.texture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(win.pixels.Bounds().Size().X), int32(win.pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(win.pixels.Pix))
	}
}
