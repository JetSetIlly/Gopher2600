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
	img  *SdlImgui
	open bool

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

func (win *winSuperchargerRegisters) isOpen() bool {
	return win.open
}

func (win *winSuperchargerRegisters) setOpen(open bool) {
	win.open = open
}

func (win *winSuperchargerRegisters) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(supercharger.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{203, 134}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	val := fmt.Sprintf("%02x", r.Value)
	imguiLabel("Value")
	if imguiHexInput("##value", 2, &val) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("value", val)
		})
	}

	imgui.SameLine()

	if r.LastWriteAddress != 0x0000 {
		imgui.Text(fmt.Sprintf("last write %#02x to %#04x", r.LastWriteValue, r.LastWriteAddress))
	} else {
		imgui.Text("no writes yet")
	}

	imgui.Spacing()

	imgui.PushItemWidth(250.0)
	delay := int32(r.Delay)
	if imgui.SliderInt("Delay##delay", &delay, 1, 6) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("delay", fmt.Sprintf("%d", delay))
		})
	}
	imgui.PopItemWidth()

	imguiSeparator()

	rw := r.RAMwrite
	imguiLabel("RAM Write")
	if imgui.Checkbox("##ramwrite", &rw) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("ramwrite", fmt.Sprintf("%v", rw))
		})
	}

	imgui.SameLine()

	rp := r.ROMpower
	imguiLabel("ROM Power")
	if imgui.Checkbox("##rompower", &rp) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("rompower", fmt.Sprintf("%v", rp))
		})
	}

	imgui.SameLine()
	win.width = imgui.CursorPosX()

	imguiSeparator()

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
