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

const winScreenTitle = "TV Screen"

const (
	pixelDepth = 3
	pixelWidth = 2
	defScaling = 2.0
)

type winScreen struct {
	windowManagement
	img *SdlImgui

	// playmode controls how the screen is displayed. currently, when
	// playmode is true:
	//   o tv screen imgui window will be created without decorations
	//   o host sdl window will be set to the same size as the tv screen
	playmode bool

	// is screen currently pointed at
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

func newWinScreen(img *SdlImgui) (managedWindow, error) {
	win := &winScreen{
		img:     img,
		scaling: defScaling,

		// horizPixels is always the same regardless of tv spec
		horizPixels: television.HorizClksVisible,
	}

	// generate texture, creation of texture will be done on first call to
	// render()
	gl.GenTextures(1, &win.texture)
	gl.BindTexture(gl.TEXTURE_2D, win.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// start off by showing entirity of NTSC screen
	win.resizeFromMain(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	return win, nil
}

func (win *winScreen) destroy() {
}

func (win *winScreen) id() string {
	return winScreenTitle
}

// draw is called by service loop
func (win *winScreen) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	// if isCaptured flag is set then change the title and border colors of the
	// TV Screen window.
	if win.isCaptured {
		imgui.PushStyleColor(imgui.StyleColorTitleBgActive, win.img.cols.CapturedScreenTitle)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.CapturedScreenBorder)
	}

	imgui.BeginV(winScreenTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// once the window has been drawn then remove any additional styling
	if win.isCaptured {
		imgui.PopStyleColorV(2)
	}

	imgui.Image(imgui.TextureID(win.texture),
		imgui.Vec2{
			win.scaledWidth(),
			win.scaledHeight(),
		})

	win.isHovered = imgui.IsItemHovered()

	if win.img.vcs != nil {
		imgui.Text(win.img.vcs.TV.String())
	}

	imgui.End()
}

// render is called by service loop
func (win *winScreen) render() {
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

// Resize implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (win *winScreen) Resize(topScanline int, visibleScanlines int) error {
	test.AssertNonMainThread()
	return win.resize(topScanline, visibleScanlines, win.setWindowFromThread)
}

// resizeFromMain is a thread version of Resize()
//
// MUST ONLY be called from the #mainthread
func (win *winScreen) resizeFromMain(topScanline int, visibleScanlines int) error {
	test.AssertMainThread()
	return win.resize(topScanline, visibleScanlines, win.setWindow)
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (win *winScreen) resize(topScanline int, visibleScanlines int, setWindow func(float32) error) error {
	win.topScanline = topScanline
	win.scanlines = visibleScanlines
	win.pixels = image.NewRGBA(image.Rect(0, 0, win.horizPixels, win.scanlines))

	win.img.lmtr.SetLimit(win.img.tv.GetSpec().FramesPerSecond)
	win.aspectBias = win.img.tv.GetSpec().AspectBias

	setWindow(reapplyScale)

	// defer recreation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	win.createTexture = true

	return nil
}

const reapplyScale = -1.0

// MUST ONLY be called from the #mainthread
func (win *winScreen) setWindow(scale float32) error {
	test.AssertMainThread()

	if scale != reapplyScale {
		win.scaling = scale
	}

	return nil
}

// MUST NOT be called from the #mainthread
// see setWindow() for non-main alternative
func (win *winScreen) setWindowFromThread(scale float32) error {
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
func (win *winScreen) NewFrame(frameNum int) error {
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (win *winScreen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (win *winScreen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {

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
func (win *winScreen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (win *winScreen) EndRendering() error {
	return nil
}

func (win *winScreen) scaledWidth() float32 {
	return float32(win.pixels.Bounds().Size().X*pixelWidth) * win.aspectBias * win.scaling
}

func (win *winScreen) scaledHeight() float32 {
	return float32(win.pixels.Bounds().Size().Y) * win.scaling
}
