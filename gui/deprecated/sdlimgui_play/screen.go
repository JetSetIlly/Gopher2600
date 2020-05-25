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

package sdlimgui_play

import (
	"image"
	"image/color"

	"github.com/jetsetilly/gopher2600/television"
	"github.com/jetsetilly/gopher2600/test"

	"github.com/go-gl/gl/v3.2-core/gl"
)

const (
	pixelWidth = 2
	defScaling = 2.0
)

// screen implements television.PixelRenderer
type screen struct {
	img *SdlImguiPlay

	crit screenCrit

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value
	scaling float32

	// aspect bias is taken from the television specification
	aspectBias float32

	// create texture on the next call of render
	createTextures bool

	// the tv screen texture
	screenTexture uint32

	// show pixel perfect image or with crt effect
	pixelPerfect bool
}

// for clarity, variables accessed in the critical section are encapsulated in
// their own subtype
type screenCrit struct {
	// WARNING: running without critical section protection for now

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int

	// pixel data
	pixels *image.RGBA

	// number of scanlines at the moment the television first became "stable".
	// we use this value, rather than scanlines above, to report the
	// scaledHeight(). this is so that the display window itself does not
	// change size
	stableScanlines int
}

func newScreen(img *SdlImguiPlay) *screen {
	scr := &screen{
		img:          img,
		scaling:      defScaling,
		pixelPerfect: false,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &scr.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, scr.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// start off by showing entirity of NTSC screen
	scr.resize(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	return scr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *screen) resize(topScanline int, visibleScanlines int) error {
	scr.crit.topScanline = topScanline
	scr.crit.scanlines = visibleScanlines

	// maybe counter-intuitively, update stableScanline if television is not
	// yet stable
	if !scr.img.tv.IsStable() {
		scr.crit.stableScanlines = visibleScanlines
	}

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksVisible, scr.crit.scanlines))
	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	// defer re-creation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	scr.createTextures = true

	// call fitDisplaySize() unless the screen field of the SdlImguiPlay is
	// nil. this happens when resize() is called as part of the newScreen()
	// function
	if scr.img.screen != nil {
		scr.img.plt.fitDisplaySize()
	}

	return nil
}

func (scr *screen) scaledWidth() float32 {
	return float32(scr.crit.pixels.Bounds().Size().X*pixelWidth) * scr.aspectBias * scr.scaling
}

func (scr *screen) scaledHeight() float32 {
	return float32(scr.crit.stableScanlines) * scr.scaling
}

// render is called by service loop
func (scr *screen) render() {
	var pixels *image.RGBA
	pixels = scr.crit.pixels

	// if frame rate is below a given threshold then fake a pause image. we
	// don't want to do this with too high of a threshold though because it
	// would just look like weird
	if scr.createTextures {
		scr.createTextures = false
		gl.ActiveTexture(gl.TEXTURE0)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

	} else {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}
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
	test.AssertNonMainThread()

	// handle VBLANK by setting pixels to black
	if vblank {
		red = 0
		green = 0
		blue = 0
	}

	rgb := color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)}
	cx := x - television.HorizClksHBlank
	cy := y - scr.crit.topScanline
	scr.crit.pixels.SetRGBA(cx, cy, rgb)

	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}
