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
	"fmt"
	"strconv"

	"github.com/jetsetilly/gopher2600/hardware/riot/timer"

	"github.com/inkyblackness/imgui-go/v4"
)

const winTimerID = "Timer"

type winTimer struct {
	img  *SdlImgui
	open bool

	// required dimensions for interval dropdown
	intervalComboDim imgui.Vec2
}

func newWinTimer(img *SdlImgui) (window, error) {
	win := &winTimer{
		img: img,
	}

	return win, nil
}

func (win *winTimer) init() {
	win.intervalComboDim = imguiGetFrameDim("", timer.IntervalList...)
}

func (win *winTimer) id() string {
	return winTimerID
}

func (win *winTimer) isOpen() bool {
	return win.open
}

func (win *winTimer) setOpen(open bool) {
	win.open = open
}

func (win *winTimer) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{667, 656}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushItemWidth(win.intervalComboDim.X)
	if imgui.BeginComboV("##timerinterval", win.img.lz.Timer.Divider, imgui.ComboFlagsNoArrowButton) {
		for _, s := range timer.IntervalList {
			if imgui.Selectable(s) {
				t := s // being careful about scope
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.RIOT.Timer.SetInterval(t)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	value := fmt.Sprintf("%02x", win.img.lz.Timer.INTIMvalue)
	imguiLabel("Value")
	if imguiHexInput("##value", 2, &value) {
		if v, err := strconv.ParseUint(value, 16, 8); err == nil {
			win.img.dbg.PushRawEvent(func() { win.img.vcs.RIOT.Timer.SetValue(uint8(v)) })
		}
	}

	imgui.SameLine()
	remaining := fmt.Sprintf("%04x", win.img.lz.Timer.TicksRemaining)
	imguiLabel("Ticks")
	if imguiHexInput("##remaining", 4, &remaining) {
		if v, err := strconv.ParseUint(value, 16, 16); err == nil {
			win.img.dbg.PushRawEvent(func() { win.img.vcs.RIOT.Timer.TicksRemaining = int(v) })
		}
	}

	imgui.End()
}
