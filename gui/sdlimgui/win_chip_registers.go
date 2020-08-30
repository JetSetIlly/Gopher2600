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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

const winChipRegistersTitle = "Chip Registers"

type winChipRegisters struct {
	windowManagement
	widgetDimensions

	img *SdlImgui
}

func newWinChipRegisters(img *SdlImgui) (managedWindow, error) {
	win := &winChipRegisters{
		img: img,
	}

	return win, nil
}

func (win *winChipRegisters) init() {
	win.widgetDimensions.init()
}

func (win *winChipRegisters) destroy() {
}

func (win *winChipRegisters) id() string {
	return winChipRegistersTitle
}

func (win *winChipRegisters) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{653, 400}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winChipRegistersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	win.drawChipRegister("SWACHA", win.img.lz.ChipRegisters.SWACHA)
	imgui.SameLine()
	win.drawChipRegister("SWACNT", win.img.lz.ChipRegisters.SWACNT)

	win.drawChipRegister("SWACHB", win.img.lz.ChipRegisters.SWACHB)
	imgui.SameLine()
	win.drawChipRegister("SWBCNT", win.img.lz.ChipRegisters.SWBCNT)

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	win.drawChipRegister("INPT0", win.img.lz.ChipRegisters.INPT0)
	imgui.SameLine()
	win.drawChipRegister("INPT1", win.img.lz.ChipRegisters.INPT1)

	win.drawChipRegister("INPT2", win.img.lz.ChipRegisters.INPT2)
	imgui.SameLine()
	win.drawChipRegister("INPT3", win.img.lz.ChipRegisters.INPT3)

	win.drawChipRegister("INPT4", win.img.lz.ChipRegisters.INPT4)
	imgui.SameLine()
	win.drawChipRegister("INPT5", win.img.lz.ChipRegisters.INPT5)

	// poking chip registers may not have the effect the user
	// expects (compare to poking CPU registers for example)
	// !!TODO: warning/help text for chip registers window

	imgui.End()
}

func (win *winChipRegisters) drawChipRegister(label string, val uint8) {
	s := fmt.Sprintf("%02x", val)
	imguiText(label)
	imgui.PushItemWidth(win.twoDigitDim.X)
	if imguiHexInput(fmt.Sprintf("##%s", label), !win.img.paused, 2, &s) {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			panic(err)
		}
		win.img.lz.Dbg.PushRawEvent(func() {
			err := win.img.lz.Dbg.VCS.Mem.Poke(addresses.ReadAddress[label], uint8(v))
			if err != nil {
				panic(err)
			}
		})
	}
	imgui.PopItemWidth()
}
