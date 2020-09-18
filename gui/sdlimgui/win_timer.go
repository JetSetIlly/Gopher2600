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

	"github.com/inkyblackness/imgui-go/v2"
)

const winTimerTitle = "Timer"

type winTimer struct {
	windowManagement

	img *SdlImgui

	// widget dimensions
	intervalComboDim imgui.Vec2
}

func newWinTimer(img *SdlImgui) (managedWindow, error) {
	win := &winTimer{
		img: img,
	}

	return win, nil
}

func (win *winTimer) init() {
	win.intervalComboDim = imguiGetFrameDim("", timer.IntervalList...)
}

func (win *winTimer) destroy() {
}

func (win *winTimer) id() string {
	return winTimerTitle
}

func (win *winTimer) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 358}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winTimerTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushItemWidth(win.intervalComboDim.X)
	if imgui.BeginComboV("##timerinterval", win.img.lz.Timer.Divider, imgui.ComboFlagNoArrowButton) {
		for _, s := range timer.IntervalList {
			if imgui.Selectable(s) {
				t := s // being careful about scope
				win.img.lz.Dbg.PushRawEvent(func() {
					win.img.lz.Dbg.VCS.RIOT.Timer.SetInterval(t)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	value := fmt.Sprintf("%02x", win.img.lz.Timer.INTIMvalue)
	imguiText("Value")
	if imguiHexInput("##value", !win.img.paused, 2, &value) {
		if v, err := strconv.ParseUint(value, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.RIOT.Timer.SetValue(uint8(v)) })
		}
	}

	imgui.SameLine()
	remaining := fmt.Sprintf("%04x", win.img.lz.Timer.TicksRemaining)
	imguiText("Ticks")
	if imguiHexInput("##remaining", !win.img.paused, 4, &remaining) {
		if v, err := strconv.ParseUint(value, 16, 16); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.RIOT.Timer.TicksRemaining = int(v) })
		}
	}

	imgui.End()
}
