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

package sdlimgui

import (
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v2"
)

const winPlayScrTitle = "Atari VCS"

// note that values from the lazy package will not be updated in the service
// loop when the emulator is in playmode. nothing in winPlayScr() therefore
// should rely on any lazy value

type winPlayScr struct {
	windowManagement

	img *SdlImgui
	scr *screen

	// textures
	screenTexture uint32

	// (re)create textures on next render()
	createTextures bool

	// the tv screen has captured mouse input
	isCaptured bool

	// additional padding for the image so that it is centred in its content space
	imagePadding imgui.Vec2

	// size of window and content area in which to centre the image
	winDim     imgui.Vec2
	contentDim imgui.Vec2

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value.
	//
	// use getScaling() and setScaling to access this value
	scaling float32
}

func newWinPlayScr(img *SdlImgui) managedWindow {
	win := &winPlayScr{
		img:     img,
		scr:     img.screen,
		scaling: 2.0,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win
}

func (win *winPlayScr) init() {
}

func (win *winPlayScr) destroy() {
}

func (win *winPlayScr) id() string {
	return winPlayScrTitle
}

func (win *winPlayScr) draw() {
	if !win.open {
		return
	}

	// actual display
	w := win.getScaledWidth()
	h := win.getScaledHeight()

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.PlayWindowBg)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.PlayWindowBorder)

	imgui.SetNextWindowPosV(imgui.Vec2{0, 0}, 0, imgui.Vec2{0, 0})
	dimen := win.img.plt.displaySize()
	win.winDim = imgui.Vec2{dimen[0], dimen[1]}
	imgui.SetNextWindowSizeV(win.winDim, 0)

	// we don't want to ever show scrollbars
	imgui.BeginV(winPlayScrTitle, &win.open,
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
			imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoFocusOnAppearing)

	// note size of window
	win.contentDim = imgui.ContentRegionAvail()

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))
	imgui.Image(imgui.TextureID(win.screenTexture), imgui.Vec2{w, h})
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	imgui.PopStyleColorV(2)

	imgui.End()
}

func (win *winPlayScr) resize() {
	win.createTextures = true
}

// render is called by service loop.
func (win *winPlayScr) render() {
	if !win.open {
		return
	}

	// critical section
	win.scr.crit.section.Lock()

	// set screen image scaling (and image padding) based on the current window size
	win.setScaleFromWindow(win.contentDim)

	// get pixels
	pixels := win.scr.crit.cropPixels

	// make a note of fram stability for later on outside of the critical section
	isStable := win.scr.crit.isStable

	win.scr.crit.section.Unlock()
	// end of critical section

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	gl.ActiveTexture(gl.TEXTURE0)

	// only draw image if television frame is stable
	if isStable {
		if win.createTextures {
			gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
			gl.TexImage2D(gl.TEXTURE_2D, 0,
				gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(pixels.Pix))

			win.createTextures = false
		} else {
			gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
			gl.TexSubImage2D(gl.TEXTURE_2D, 0,
				0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(pixels.Pix))
		}
	}
}

func (win *winPlayScr) getScaledWidth() float32 {
	// must be called from with a critical section
	return float32(win.scr.crit.cropPixels.Bounds().Size().X) * win.getScaling(true)
}

func (win *winPlayScr) getScaledHeight() float32 {
	// must be called from with a critical section
	return float32(win.scr.crit.cropPixels.Bounds().Size().Y) * win.getScaling(false)
}

func (win *winPlayScr) setScaleFromWindow(sz imgui.Vec2) {
	// must be called from with a critical section

	winAspectRatio := sz.X / sz.Y

	imageW := float32(win.scr.crit.cropPixels.Bounds().Size().X)
	imageH := float32(win.scr.crit.cropPixels.Bounds().Size().Y)
	imageW *= pixelWidth * win.scr.aspectBias
	aspectRatio := imageW / imageH

	if aspectRatio < winAspectRatio {
		win.scaling = sz.Y / imageH
		win.imagePadding = imgui.Vec2{X: float32(int((sz.X - (imageW * win.scaling)) / 2))}
	} else {
		win.scaling = sz.X / imageW
		win.imagePadding = imgui.Vec2{Y: float32(int((sz.Y - (imageH * win.scaling)) / 2))}
	}
}

func (win *winPlayScr) getScaling(horiz bool) float32 {
	if horiz {
		return pixelWidth * win.scr.aspectBias * win.scaling
	}
	return win.scaling
}

func (win *winPlayScr) setScaling(scaling float32) {
	win.winDim = win.winDim.Times(scaling / win.scaling)
	win.img.plt.window.SetSize(int32(win.winDim.X), int32(win.winDim.Y))
}
