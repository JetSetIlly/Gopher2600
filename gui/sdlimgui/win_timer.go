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
	"gopher2600/hardware/riot/timer"
	"strconv"

	"github.com/inkyblackness/imgui-go/v2"
)

const winTimerTitle = "Timer"

type winTimer struct {
	windowManagement
	img *SdlImgui

	// widget dimensions
	intervalComboDim imgui.Vec2
	valueDim         imgui.Vec2
	ticksDim         imgui.Vec2
}

func newWinTimer(img *SdlImgui) (managedWindow, error) {
	win := &winTimer{
		img: img,
	}

	return win, nil
}

func (win *winTimer) init() {
	win.intervalComboDim = imguiGetFrameDim("", timer.IntervalList...)
	win.ticksDim = imguiGetFrameDim("FFFF")
	win.valueDim = imguiGetFrameDim("FF")
}

func (win *winTimer) destroy() {
}

func (win *winTimer) id() string {
	return winTimerTitle
}

// draw is called by service loop
func (win *winTimer) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{359, 664}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{464, 48}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winTimerTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushItemWidth(win.intervalComboDim.X)
	if imgui.BeginComboV("", win.img.lazy.Timer.Requested, imgui.ComboFlagNoArrowButton) {
		for _, s := range timer.IntervalList {
			if imgui.Selectable(s) {
				t := s // being careful about scope
				win.img.lazy.Dbg.PushRawEvent(func() {
					win.img.lazy.VCS.RIOT.Timer.SetInterval(t)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	value := fmt.Sprintf("%02x", win.img.lazy.Timer.INTIMvalue)
	imgui.PushItemWidth(win.ticksDim.X)
	imgui.SameLine()
	imguiText("Value")
	if imguiHexInput("##value", !win.img.paused, 2, &value) {
		if v, err := strconv.ParseUint(value, 16, 8); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.VCS.RIOT.Timer.SetValue(uint8(v)) })
		}
	}
	imgui.PopItemWidth()

	remaining := fmt.Sprintf("%04x", win.img.lazy.Timer.TicksRemaining)
	imgui.PushItemWidth(win.ticksDim.X)
	imgui.SameLine()
	imguiText("Ticks")
	if imguiHexInput("##remaining", !win.img.paused, 4, &remaining) {
		if v, err := strconv.ParseUint(value, 16, 16); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.VCS.RIOT.Timer.TicksRemaining = uint16(v) })
		}
	}
	imgui.PopItemWidth()

	imgui.End()
}
