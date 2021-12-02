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
	"image"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winBotID = "Bot"

type winBot struct {
	img  *SdlImgui
	open bool

	obsTexture uint32

	// render channels are given to use by the main emulation through a GUI request
	render chan *image.RGBA
}

func newWinBot(img *SdlImgui) (window, error) {
	win := &winBot{
		img: img,
	}

	gl.GenTextures(1, &win.obsTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

func (win *winBot) init() {
}

func (win winBot) id() string {
	return winBotID
}

func (win *winBot) isOpen() bool {
	return win.open
}

func (win *winBot) setOpen(open bool) {
	if win.render == nil {
		return
	}

	win.open = open

	if win.open {
		// clear texture
		gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, 1, 1, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr([]uint8{0}))
	}
}

func (win *winBot) draw() {
	// receive new thumbnail data and copy to texture
	select {
	case img := <-win.render:
		if img != nil {
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(img.Stride)/4)
			defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

			gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
			gl.TexImage2D(gl.TEXTURE_2D, 0,
				gl.RGBA, int32(img.Bounds().Size().X), int32(img.Bounds().Size().Y), 0,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(img.Pix))
		}
	default:
	}

	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{75, 75}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize) {
		imgui.Image(imgui.TextureID(win.obsTexture), imgui.Vec2{specification.ClksVisible * 3, specification.AbsoluteMaxScanlines})
	}
	imgui.End()
}
