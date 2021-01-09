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
	"github.com/inkyblackness/imgui-go/v3"
)

const winPlayScrTitle = "Atari VCS"

// note that values from the lazy package will not be updated in the service
// loop when the emulator is in playmode. nothing in winPlayScr() therefore
// should rely on any lazy value

type winPlayScr struct {
	img  *SdlImgui
	open bool

	// fps sub-window
	fps *winPlayScrFPS

	// reference to screen data
	scr *screen

	// textures
	screenTexture uint32
	phosphor      uint32

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

func newWinPlayScr(img *SdlImgui) window {
	win := &winPlayScr{
		img:     img,
		scr:     img.screen,
		scaling: 2.0,
	}

	win.fps = newWinPlayScrFPS(win.img).(*winPlayScrFPS)

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.GenTextures(1, &win.phosphor)
	gl.BindTexture(gl.TEXTURE_2D, win.phosphor)
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

func (win *winPlayScr) isOpen() bool {
	return win.open
}

func (win *winPlayScr) setOpen(open bool) {
	win.open = open
}

func (win *winPlayScr) draw() {
	if !win.open {
		return
	}

	dimen := win.img.plt.displaySize()
	win.winDim = imgui.Vec2{dimen[0], dimen[1]}
	imgui.SetNextWindowSizeV(win.winDim, 0)

	imgui.SetNextWindowPosV(imgui.Vec2{0, 0}, 0, imgui.Vec2{0, 0})

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.PlayWindowBg)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.PlayWindowBorder)

	imgui.BeginV(winPlayScrTitle, &win.open,
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus)

	// note size of window
	win.contentDim = imgui.ContentRegionAvail()

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// actual display
	imgui.Image(imgui.TextureID(win.screenTexture), imgui.Vec2{win.getScaledWidth(), win.getScaledHeight()})

	// capture mouse on double click but only if image is being hovered over
	// and there is no modal window.
	if !win.img.hasModal && imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
		win.img.setCapture(true)
	}

	imgui.PopStyleColorV(2)
	imgui.End()

	win.fps.draw()
}

// resize() implements the textureRenderer interface.
func (win *winPlayScr) resize() {
	win.createTextures = true
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). must be inside
// screen critical section.
func (win *winPlayScr) render() {
	if !win.open {
		return
	}

	// set screen image scaling (and image padding) based on the current window size
	win.setScaleAndPadding(win.contentDim)

	// get pixels
	pixels := win.scr.crit.cropPixels
	phosphor := win.scr.crit.cropPhosphor

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if win.createTextures {
		win.createTextures = false

		// (re)create textures
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphor)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(phosphor.Bounds().Size().X), int32(phosphor.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))
	} else if win.scr.crit.isStable {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphor)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(phosphor.Bounds().Size().X), int32(phosphor.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))
	}
}

// must be called from with a critical section.
func (win *winPlayScr) getScaledWidth() float32 {
	return float32(win.scr.crit.cropPixels.Bounds().Size().X) * win.getScaling(true)
}

// must be called from with a critical section.
func (win *winPlayScr) getScaledHeight() float32 {
	return float32(win.scr.crit.cropPixels.Bounds().Size().Y) * win.getScaling(false)
}

// must be called from with a critical section.
func (win *winPlayScr) setScaleAndPadding(sz imgui.Vec2) {
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
