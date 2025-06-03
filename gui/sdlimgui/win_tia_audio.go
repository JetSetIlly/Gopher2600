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
)

const winTIAAudioID = "TIA Audio"
const winTIAAudioMenu = "TIA (Audio)"

type winTIAAudio struct {
	debuggerWin
	img *SdlImgui
}

func newWinTIAAudio(img *SdlImgui) (window, error) {
	win := &winTIAAudio{img: img}
	return win, nil
}

func (win *winTIAAudio) init() {
}

func (win *winTIAAudio) id() string {
	return winTIAAudioID
}

func (win *winTIAAudio) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 800, Y: 400}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTIAAudio) draw() {
	ch0 := win.img.cache.VCS.TIA.Audio.Channel0.Registers
	ch1 := win.img.cache.VCS.TIA.Audio.Channel1.Registers

	audc0 := fmt.Sprintf("%02x", ch0.Control)
	if imguiHexInput("AUDC0##audc0", 2, &audc0) {
		if v, err := strconv.ParseUint(audc0, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel0.Registers.Control = uint8(v) })
		}
	}
	imgui.SameLineV(0, 10)
	audc1 := fmt.Sprintf("%02x", ch1.Control)
	if imguiHexInput("AUDC1##audc1", 2, &audc1) {
		if v, err := strconv.ParseUint(audc1, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel1.Registers.Control = uint8(v) })
		}
	}

	audf0 := fmt.Sprintf("%02x", ch0.Freq)
	if imguiHexInput("AUDF0##audf0", 2, &audf0) {
		if v, err := strconv.ParseUint(audf0, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel0.Registers.Freq = uint8(v) })
		}
	}
	imgui.SameLineV(0, 10)
	audf1 := fmt.Sprintf("%02x", ch1.Freq)
	if imguiHexInput("AUDF1##audf1", 2, &audf1) {
		if v, err := strconv.ParseUint(audf1, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel1.Registers.Freq = uint8(v) })
		}
	}

	audv0 := fmt.Sprintf("%02x", ch0.Volume)
	if imguiHexInput("AUDV0##audv0", 2, &audv0) {
		if v, err := strconv.ParseUint(audv0, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel0.Registers.Volume = uint8(v) })
		}
	}
	imgui.SameLineV(0, 10)
	audv1 := fmt.Sprintf("%02x", ch1.Volume)
	if imguiHexInput("AUDV1##audv1", 2, &audv1) {
		if v, err := strconv.ParseUint(audv1, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TIA.Audio.Channel1.Registers.Volume = uint8(v) })
		}
	}
}
