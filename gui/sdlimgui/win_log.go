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
	"github.com/inkyblackness/imgui-go/v2"
)

const winLogTitle = "Log"

type winLog struct {
	windowManagement

	img *SdlImgui
}

func newWinLog(img *SdlImgui) (managedWindow, error) {
	win := &winLog{
		img: img,
	}

	return win, nil
}

func (win *winLog) init() {
}

func (win *winLog) destroy() {
}

func (win *winLog) id() string {
	return winLogTitle
}

func (win *winLog) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{500, 480}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{400, 400}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.LogBackground)
	imgui.BeginV(winLogTitle, &win.open, 0)
	imgui.PopStyleColor()

	var clipper imgui.ListClipper
	clipper.Begin(len(win.img.lz.Log.Log))
	for clipper.Step() {
		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			imgui.Text(win.img.lz.Log.Log[i].String())
		}
	}

	// scroll to end if log has been dirtied (ie. a new entry)
	if win.img.lz.Log.Dirty {
		imgui.SetScrollHereY(0.0)
		win.img.lz.Log.Dirty = false
	}

	imgui.End()
}
