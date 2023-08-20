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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

const winTimerID = "Timer"

type winTimer struct {
	debuggerWin
	img *SdlImgui
}

func newWinTimer(img *SdlImgui) (window, error) {
	win := &winTimer{
		img: img,
	}

	return win, nil
}

var dividerList = []string{"TIM1T", "TIM8T", "TIM64T", "T1024T"}

func (win *winTimer) init() {
}

func (win *winTimer) id() string {
	return winTimerID
}

func (win *winTimer) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{825, 617}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTimer) draw() {
	Divider := win.img.cache.VCS.RIOT.Timer.PeekField("divider").(timer.Divider)
	INTIM := win.img.cache.VCS.RIOT.Timer.PeekField("intim").(uint8)
	TicksRemaining := win.img.cache.VCS.RIOT.Timer.PeekField("ticksRemaining").(int)
	TIMINT := win.img.cache.VCS.RIOT.Timer.PeekField("timint").(uint8)

	if imgui.BeginComboV("##divider", Divider.String(), imgui.ComboFlagsNone) {
		for _, s := range dividerList {
			if imgui.Selectable(s) {
				var div timer.Divider
				switch s {
				case "TIM1T":
					div = timer.TIM1T
				case "TIM8T":
					div = timer.TIM8T
				case "TIM64T":
					div = timer.TIM64T
				case "T1024T":
					div = timer.T1024T
				default:
					panic("unknown timer divider")
				}
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Timer.PokeField("divider", div)
				})
			}
		}

		imgui.EndCombo()
	}

	intim := fmt.Sprintf("%02x", INTIM)
	imguiLabel("INTIM")
	if imguiHexInput("##intim", 2, &intim) {
		if v, err := strconv.ParseUint(intim, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().RIOT.Timer.PokeField("intim", uint8(v)) })
		}
	}

	imgui.SameLine()
	remaining := fmt.Sprintf("%04x", TicksRemaining)
	imguiLabel("Ticks")
	if imguiHexInput("##remaining", 4, &remaining) {
		if v, err := strconv.ParseUint(remaining, 16, 16); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().RIOT.Timer.PokeField("ticksRemaining", int(v)) })
		}
	}

	imguiLabel("TIMINT")
	drawRegister("##TIMINT", TIMINT, timer.MaskTIMINT, win.img.cols.timerBit,
		func(v uint8) {
			win.img.dbg.PushFunction(func() {
				win.img.dbg.VCS().RIOT.Timer.PokeField("timint", v)
			})
		})
}
