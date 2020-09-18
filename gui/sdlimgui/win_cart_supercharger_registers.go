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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
)

const winSuperchargerRegistersTitle = "AR Registers"

type winSuperchargerRegisters struct {
	windowManagement

	img *SdlImgui
}

func newWinSuperchargerRegisters(img *SdlImgui) (managedWindow, error) {
	win := &winSuperchargerRegisters{
		img: img,
	}

	return win, nil
}

func (win *winSuperchargerRegisters) init() {
}

func (win *winSuperchargerRegisters) destroy() {
}

func (win *winSuperchargerRegisters) id() string {
	return winSuperchargerRegistersTitle
}

func (win *winSuperchargerRegisters) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	_, ok := win.img.lz.Cart.Registers.(supercharger.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 451}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winSuperchargerRegistersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	r, ok := win.img.lz.Cart.Registers.(supercharger.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	val := fmt.Sprintf("%02x", r.Value)
	imguiText("Value")
	if imguiHexInput("##value", !win.img.paused, 2, &val) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("value", val)
		})
	}

	imgui.SameLine()

	delay := int32(r.Delay)
	imguiText("Delay")
	imgui.PushItemWidth(imgui.WindowWidth() / 2)
	if imgui.SliderInt("##delay", &delay, 1, 6) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("delay", fmt.Sprintf("%d", delay))
		})
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	if r.LastWriteAddress != 0x0000 {
		imgui.Text(fmt.Sprintf("last write %#02x to %#04x", r.LastWriteValue, r.LastWriteAddress))
	} else {
		imgui.Text("")
	}

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	rw := r.RAMwrite
	imguiText("RAM Write")
	if imgui.Checkbox("##ramwrite", &rw) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("ramwrite", fmt.Sprintf("%v", rw))
		})
	}

	imgui.SameLine()

	rp := r.ROMpower
	imguiText("ROM Power")
	if imgui.Checkbox("##rompower", &rp) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("rompower", fmt.Sprintf("%v", rp))
		})
	}

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	banking := r.BankingMode

	setBanking := func(v int) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
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

	imgui.End()
}
