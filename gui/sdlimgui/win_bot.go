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
	"strings"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winBotID = "Bot"

type winBot struct {
	img  *SdlImgui
	open bool

	obsTexture  uint32
	diagnostics []bots.Diagnostic
	dirty       bool

	// render channels are given to use by the main emulation through a GUI request
	feedback *bots.Feedback
}

func newWinBot(img *SdlImgui) (window, error) {
	win := &winBot{
		img:         img,
		diagnostics: make([]bots.Diagnostic, 0, 1024),
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

// start bot session will effectively end a bot session if feedback channels are nil
func (win *winBot) startBotSession(feedback *bots.Feedback) {
	win.feedback = feedback

	gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, 1, 1, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr([]uint8{0}))

	win.diagnostics = win.diagnostics[:]
}

// do not open if no bot is defined
func (win *winBot) setOpen(open bool) {
	if win.feedback == nil {
		win.open = false
		return
	}
	win.open = open
}

func (win *winBot) draw() {
	// no bot feedback instance
	if win.feedback == nil {
		return
	}

	// receive new thumbnail data and copy to texture
	select {
	case img := <-win.feedback.Images:
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

	done := false
	for !done {
		select {
		case d := <-win.feedback.Diagnostic:
			if len(win.diagnostics) == cap(win.diagnostics) {
				win.diagnostics = win.diagnostics[1:]
			}
			win.diagnostics = append(win.diagnostics, d)
			win.dirty = true
		default:
			done = true
		}
	}

	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{75, 75}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{500, 525}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{500, 500}, imgui.Vec2{500, 800})

	if imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNone) {
		imgui.Image(imgui.TextureID(win.obsTexture), imgui.Vec2{specification.ClksVisible * 3, specification.AbsoluteMaxScanlines})

		if imgui.BeginChildV("##log", imgui.Vec2{}, true, imgui.WindowFlagsAlwaysAutoResize) {
			var clipper imgui.ListClipper
			clipper.Begin(len(win.diagnostics))
			for clipper.Step() {
				for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
					imgui.Text(win.diagnostics[i].Group)
					for _, s := range strings.Split(win.diagnostics[i].Diagnostic, "\n") {
						imgui.SameLine()
						imgui.Text(s)
					}
				}
			}

			if win.dirty {
				imgui.SetScrollHereY(0.0)
				win.dirty = false
			}
			imgui.EndChild()
		}
	}
	imgui.End()
}
