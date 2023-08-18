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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
)

const winSuperchargerRegistersID = "AR Registers"

type winSuperchargerRegisters struct {
	debuggerWin

	img *SdlImgui

	width float32
}

func newWinSuperchargerRegisters(img *SdlImgui) (window, error) {
	win := &winSuperchargerRegisters{
		img: img,
	}

	return win, nil
}

func (win *winSuperchargerRegisters) init() {
}

func (win *winSuperchargerRegisters) id() string {
	return winSuperchargerRegistersID
}

func (win *winSuperchargerRegisters) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no cartridge registers bus available
	bus := win.img.cache.VCS.Mem.Cart.GetRegistersBus()
	if bus == nil {
		return false
	}
	regs, ok := bus.GetRegisters().(supercharger.Registers)
	if !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{203, 134}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw(regs)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSuperchargerRegisters) draw(regs supercharger.Registers) {
	val := fmt.Sprintf("%02x", regs.Value)
	imguiLabel("Value")
	if imguiHexInput("##value", 2, &val) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("value", val)
		})
	}

	imgui.SameLine()

	if regs.LastWriteAddress != 0x0000 {
		imgui.Text(fmt.Sprintf("last write %#02x to %#04x", regs.LastWriteValue, regs.LastWriteAddress))
	} else {
		imgui.Text("no writes yet")
	}

	imgui.Spacing()

	imgui.PushItemWidth(250.0)
	delay := int32(regs.Delay)
	if imgui.SliderInt("Delay##delay", &delay, 1, 6) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("delay", fmt.Sprintf("%d", delay))
		})
	}
	imgui.PopItemWidth()

	imguiSeparator()

	rw := regs.RAMwrite
	imguiLabel("RAM Write")
	if imgui.Checkbox("##ramwrite", &rw) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("ramwrite", fmt.Sprintf("%v", rw))
		})
	}

	imgui.SameLine()

	rp := regs.ROMpower
	imguiLabel("ROM Power")
	if imgui.Checkbox("##rompower", &rp) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("rompower", fmt.Sprintf("%v", rp))
		})
	}

	imgui.SameLine()
	win.width = imgui.CursorPosX()

	imguiSeparator()

	banking := regs.BankingMode

	setBanking := func(v int) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("bankingmode", fmt.Sprintf("%v", v))
		})
	}

	if imgui.SelectableV("  RAM 3   BIOS##bank0", banking == 0, 0, imgui.Vec2{0, 0}) {
		setBanking(0)
	}
	if imgui.SelectableV("  RAM 1   BIOS##bank1", banking == 1, 0, imgui.Vec2{0, 0}) {
		setBanking(1)
	}
	if imgui.SelectableV("  RAM 3   RAM 1##bank2", banking == 2, 0, imgui.Vec2{0, 0}) {
		setBanking(2)
	}
	if imgui.SelectableV("  RAM 1   RAM 3##bank3", banking == 3, 0, imgui.Vec2{0, 0}) {
		setBanking(3)
	}
	if imgui.SelectableV("  RAM 3   BIOS##bank4", banking == 4, 0, imgui.Vec2{0, 0}) {
		setBanking(4)
	}
	if imgui.SelectableV("  RAM 2   BIOS##bank5", banking == 5, 0, imgui.Vec2{0, 0}) {
		setBanking(5)
	}
	if imgui.SelectableV("  RAM 3   RAM 2##bank6", banking == 6, 0, imgui.Vec2{0, 0}) {
		setBanking(6)
	}
	if imgui.SelectableV("  RAM 2   RAM 3##bank7", banking == 7, 0, imgui.Vec2{0, 0}) {
		setBanking(7)
	}
}
