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
	"github.com/inkyblackness/imgui-go/v2"
)

const pixelDepth = 3

const pixelWidth = 2

const defScaling = 2.0

const tvscreenTitle = "TV Screen"

type tvScreen struct {
	img *SdlImgui

	// playmode controls how the screen is displayed. currently, when
	// playmode is true:
	//   o tv screen imgui window will be created without decorations
	//   o host sdl window will be set to the same size as the tv screen
	playmode bool

	// is tvscreen currently pointed at
	isHovered bool

	// the tv screen has captured mouse input
	isCaptured bool

	// create texture on the next call of render
	createTexture bool

	// the tv screen texture and backing pixels
	texture uint32
	pixels  *image.RGBA

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value
	scaling float32

	// aspect bias is taken from the television specification
	aspectBias float32

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int
	horizPixels int
}

func newTvScreen(img *SdlImgui) (*tvScreen, error) {
	scr := &tvScreen{
		img:     img,
		scaling: defScaling,

		// horizPixels is always the same regardless of tv spec
		horizPixels: television.HorizClksVisible,
	}

	// generate texture, creation of texture will be done on first call to
	// render()
	gl.GenTextures(1, &scr.texture)
	gl.BindTexture(gl.TEXTURE_2D, scr.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// start off by showing entirity of NTSC screen
	scr.resizeFromMain(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	return scr, nil
}

func (scr *tvScreen) destroy() {
}

// Resize implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *tvScreen) Resize(topScanline int, visibleScanlines int) error {
	test.AssertNonMainThread()
	return scr.resize(topScanline, visibleScanlines, scr.setWindowFromThread)
}

// resizeFromMain is a thread version of Resize()
//
// MUST ONLY be called from the #mainthread
func (scr *tvScreen) resizeFromMain(topScanline int, visibleScanlines int) error {
	test.AssertMainThread()
	return scr.resize(topScanline, visibleScanlines, scr.setWindow)
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *tvScreen) resize(topScanline int, visibleScanlines int, setWindow func(float32) error) error {
	scr.topScanline = topScanline
	scr.scanlines = visibleScanlines
	scr.pixels = image.NewRGBA(image.Rect(0, 0, scr.horizPixels, scr.scanlines))

	scr.img.lmtr.SetLimit(scr.img.tv.GetSpec().FramesPerSecond)
	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	setWindow(reapplyScale)

	// defer recreation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	scr.createTexture = true

	return nil
}

const reapplyScale = -1.0

// MUST ONLY be called from the #mainthread
func (scr *tvScreen) setWindow(scale float32) error {
	test.AssertMainThread()

	if scale != reapplyScale {
		scr.scaling = scale
	}

	return nil
}

// MUST NOT be called from the #mainthread
// see setWindow() for non-main alternative
func (scr *tvScreen) setWindowFromThread(scale float32) error {
	test.AssertNonMainThread()

	scr.img.service <- func() {
		scr.setWindow(scale)
		scr.img.serviceErr <- nil
	}
	return <-scr.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *tvScreen) NewFrame(frameNum int) error {
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (scr *tvScreen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (scr *tvScreen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	scr.pixels.Set(x-television.HorizClksHBlank, y-scr.topScanline,
		color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)})
	return nil
}

// SetAltPixel implements the television.PixelRenderer interface
func (scr *tvScreen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *tvScreen) EndRendering() error {
	return nil
}

func (scr *tvScreen) scaledWidth() float32 {
	return float32(scr.pixels.Bounds().Size().X*pixelWidth) * scr.aspectBias * scr.scaling
}

func (scr *tvScreen) scaledHeight() float32 {
	return float32(scr.pixels.Bounds().Size().Y) * scr.scaling
}

// render is called by service loop
func (scr *tvScreen) render() {
	gl.BindTexture(gl.TEXTURE_2D, scr.texture)

	if scr.createTexture {
		scr.createTexture = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(scr.pixels.Bounds().Size().X), int32(scr.pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(scr.pixels.Pix))
	} else {
		gl.BindTexture(gl.TEXTURE_2D, scr.texture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(scr.pixels.Bounds().Size().X), int32(scr.pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(scr.pixels.Pix))
	}
}

// draw is called by service loop
func (scr *tvScreen) draw() {
	imgui.SetNextWindowPosV(imgui.Vec2{35, 19}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(tvscreenTitle, nil, imgui.WindowFlagsAlwaysAutoResize)

	imgui.Image(imgui.TextureID(scr.texture),
		imgui.Vec2{
			scr.scaledWidth(),
			scr.scaledHeight(),
		})

	scr.isHovered = imgui.IsItemHovered()

	imgui.End()
}
