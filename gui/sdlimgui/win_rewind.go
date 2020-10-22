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
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"

	"github.com/inkyblackness/imgui-go/v2"
)

const winRewindTitle = "Rewind"

type winRewind struct {
	windowManagement

	img *SdlImgui

	// widget dimensions
	intervalComboDim imgui.Vec2
}

func newWinRewind(img *SdlImgui) (managedWindow, error) {
	win := &winRewind{
		img: img,
	}

	return win, nil
}

func (win *winRewind) init() {
	win.intervalComboDim = imguiGetFrameDim("", timer.IntervalList...)
}

func (win *winRewind) destroy() {
}

func (win *winRewind) id() string {
	return winRewindTitle
}

func (win *winRewind) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 358}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winRewindTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	n := int32(win.img.lz.Rewind.NumStates) - 1
	if n < 0 {
		n = 0
	}
	pos := int32(win.img.lz.Rewind.CurrState)

	if imgui.SliderInt("Frame", &pos, 0, n) {
		win.img.lz.Dbg.PushRawEvent(func() {
			win.img.lz.Dbg.VCS.Rewind.SetPosition(int(pos))
		})
	}

	imgui.End()
}
