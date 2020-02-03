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

package sdlwindows

import (
	"gopher2600/television"
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v2"
)

const pixelDepth = 3

const pixelWidth = 2
const pixelScale = 2.0

type tvScreen struct {
	wnd *SdlWindows

	createTexture bool
	texture       uint32
	img           *image.RGBA

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int
	horizPixels int

	scaling    float32
	aspectBias float32
}

func newTvScreen(wnd *SdlWindows) (*tvScreen, error) {
	scr := &tvScreen{
		wnd:     wnd,
		scaling: pixelScale,

		// horizPixels is always the same regardless of tv spec
		horizPixels: television.HorizClksVisible,
	}

	gl.GenTextures(1, &scr.texture)
	gl.BindTexture(gl.TEXTURE_2D, scr.texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// start off by showing entirity of NTSC screen
	scr.Resize(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	return scr, nil
}

func (scr *tvScreen) destroy() {
}

func (scr *tvScreen) render() {
	gl.BindTexture(gl.TEXTURE_2D, scr.texture)

	if scr.createTexture {
		scr.createTexture = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(scr.img.Bounds().Size().X), int32(scr.img.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(scr.img.Pix))
	} else {
		gl.BindTexture(gl.TEXTURE_2D, scr.texture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(scr.img.Bounds().Size().X), int32(scr.img.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(scr.img.Pix))
	}
}

// Resize implements the television.PixelRenderer interface
func (scr *tvScreen) Resize(topScanline int, visibleScanlines int) error {
	scr.topScanline = topScanline
	scr.scanlines = visibleScanlines
	scr.img = image.NewRGBA(image.Rect(0, 0, scr.horizPixels, scr.scanlines))

	scr.wnd.lmtr.SetLimit(scr.wnd.tv.GetSpec().FramesPerSecond)
	scr.aspectBias = scr.wnd.tv.GetSpec().AspectBias

	scr.setScale(reapplyScale)

	// defer recreation of texture to render(). we have to do it in the
	// #mainthread so we may as well defer it to there
	scr.createTexture = true

	return nil
}

const reapplyScale = -0.0

func (scr *tvScreen) setScale(scale float32) error {
	if scale > 0.0 {
		scr.scaling = scale
	}

	// we need to add some padding because I can't get a true borderless imgui
	// window. not sure what the reasoning is for the value but it works
	padding := float32(4.0)

	scr.wnd.platform.setDisplaySize(int(scr.width()+padding), int(scr.height()+padding))
	return nil
}

// NewFrame implements the television.PixelRenderer interface
func (scr *tvScreen) NewFrame(frameNum int) error {
	if scr.wnd.showOnNextStable && scr.wnd.tv.IsStable() {
		scr.wnd.platform.window.Show()
		scr.wnd.showOnNextStable = false
	}
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (scr *tvScreen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (scr *tvScreen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	scr.img.Set(x-television.HorizClksHBlank, y-scr.topScanline,
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

func (scr *tvScreen) width() float32 {
	return float32(scr.img.Bounds().Size().X*pixelWidth) * scr.aspectBias * scr.scaling
}

func (scr *tvScreen) height() float32 {
	return float32(scr.img.Bounds().Size().Y) * scr.scaling
}

func (scr *tvScreen) draw() {
	imgui.SetNextWindowPos(imgui.Vec2{0, 0})
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{0, 0})
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 0.0)

	open := false
	imgui.BeginV("TV Screen", &open,
		imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDecoration|
			imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoMove|
			imgui.WindowFlagsNoTitleBar,
	)
	imgui.Image(imgui.TextureID(scr.texture),
		imgui.Vec2{
			scr.width(),
			scr.height(),
		})
	imgui.End()

	imgui.PopStyleVar()
	imgui.PopStyleVar()
}
