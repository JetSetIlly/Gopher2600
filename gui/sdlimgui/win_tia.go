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
	"fmt"

	"github.com/inkyblackness/imgui-go/v2"
)

const winTIATitle = "TIA"

type winTIA struct {
	windowManagement
	img *SdlImgui
}

func newWinTIA(img *SdlImgui) (managedWindow, error) {
	win := &winTIA{
		img: img,
	}

	return win, nil
}

func (win *winTIA) init() {
}

func (win *winTIA) destroy() {
}

func (win *winTIA) id() string {
	return winTIATitle
}

// draw is called by service loop
func (win *winTIA) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{12, 500}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{718, 156}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winTIATitle, &win.open, 0)

	imgui.BeginTabBar("")
	if imgui.BeginTabItem("HSYNC/Playfield") {
		win.drawPlayfield()
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 0") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 1") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 0") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 1") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Ball") {
		imgui.EndTabItem()
	}
	imgui.EndTabBar()

	imgui.End()
}

func (win *winTIA) drawPlayfield() {
	imgui.Spacing()

	imgui.BeginGroup()
	imguiLabel("Foreground")
	if win.img.imguiColorCirc(win.img.vcs.TIA.Video.Playfield.ForegroundColor) {
	}

	imguiLabel("Background")
	if win.img.imguiColorCirc(win.img.vcs.TIA.Video.Playfield.BackgroundColor) {
		fmt.Println("bg")
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("Reflected")
	imgui.Checkbox("##reflected", &win.img.vcs.TIA.Video.Playfield.Reflected)

	imgui.SameLine()
	imguiLabel("Priority")
	imgui.Checkbox("##priority", &win.img.vcs.TIA.Video.Playfield.Priority)

	imgui.SameLine()
	imguiLabel("Scoremode")
	imgui.Checkbox("##scoremode", &win.img.vcs.TIA.Video.Playfield.Scoremode)

	imgui.Spacing()
	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("PF0")
	imgui.SameLine()
	imgui.BeginGroup()
	for i := 0; i < 4; i++ {
		var col uint8
		if (win.img.vcs.TIA.Video.Playfield.PF0<<i)&0x80 == 0x80 {
			col = win.img.vcs.TIA.Video.Playfield.ForegroundColor
		} else {
			col = win.img.vcs.TIA.Video.Playfield.BackgroundColor
		}
		if win.img.imguiColorRect(col) {
			v := win.img.vcs.TIA.Video.Playfield.PF0
			v ^= 0x80 >> i
			win.img.vcs.TIA.Video.Playfield.SetPF0(v)
		}
	}
	imgui.EndGroup()

	imgui.SameLine()
	imguiLabel("PF1")
	imgui.SameLine()
	imgui.BeginGroup()
	for i := 0; i < 8; i++ {
		var col uint8
		if (win.img.vcs.TIA.Video.Playfield.PF1<<i)&0x80 == 0x80 {
			col = win.img.vcs.TIA.Video.Playfield.ForegroundColor
		} else {
			col = win.img.vcs.TIA.Video.Playfield.BackgroundColor
		}
		if win.img.imguiColorRect(col) {
			v := win.img.vcs.TIA.Video.Playfield.PF1
			v ^= 0x80 >> i
			win.img.vcs.TIA.Video.Playfield.SetPF1(v)
		}
	}
	imgui.EndGroup()

	imgui.SameLine()
	imguiLabel("PF2")
	imgui.SameLine()
	imgui.BeginGroup()
	for i := 0; i < 8; i++ {
		var col uint8
		if (win.img.vcs.TIA.Video.Playfield.PF2<<i)&0x80 == 0x80 {
			col = win.img.vcs.TIA.Video.Playfield.ForegroundColor
		} else {
			col = win.img.vcs.TIA.Video.Playfield.BackgroundColor
		}
		if win.img.imguiColorRect(col) {
			v := win.img.vcs.TIA.Video.Playfield.PF2
			v ^= 0x80 >> i
			win.img.vcs.TIA.Video.Playfield.SetPF2(v)
		}
	}
	imgui.EndGroup()
}
