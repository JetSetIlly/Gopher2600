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

	imgui.BeginGroup()
	imgui.Text(win.img.vcs.TIA.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.String())
	imgui.Text(win.img.vcs.TIA.Video.Player0.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.Video.Player0.String())
	imgui.Text(win.img.vcs.TIA.Video.Player1.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.Video.Player1.String())
	imgui.Text(win.img.vcs.TIA.Video.Missile0.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.Video.Missile0.String())
	imgui.Text(win.img.vcs.TIA.Video.Missile1.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.Video.Missile1.String())
	imgui.Text(win.img.vcs.TIA.Video.Ball.Label())
	imgui.SameLine()
	imgui.Text(win.img.vcs.TIA.Video.Ball.String())
	imgui.EndGroup()

	imgui.End()
}
