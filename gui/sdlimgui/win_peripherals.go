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
	"github.com/jetsetilly/gopher2600/hardware/peripherals"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

const winPeripheralsID = "Peripherals"

type winPeripherals struct {
	img  *SdlImgui
	open bool

	// required dimensions for controller dropdown
	controllerComboDim imgui.Vec2
}

func newWinPeripherals(img *SdlImgui) (window, error) {
	win := &winPeripherals{
		img: img,
	}

	return win, nil
}

func (win *winPeripherals) init() {
	win.controllerComboDim = imguiGetFrameDim("", peripherals.AvailableRightPlayer...)
}

func (win *winPeripherals) id() string {
	return winPeripheralsID
}

func (win *winPeripherals) isOpen() bool {
	return win.open
}

func (win *winPeripherals) setOpen(open bool) {
	win.open = open
}

func (win *winPeripherals) draw() {
	if !win.open {
		return
	}

	// don't show the window if either of the controllers are unplugged
	// !!TODO: show something meaningful for unplugged controllers
	if win.img.lz.Peripherals.LeftPlayer == nil || win.img.lz.Peripherals.RightPlayer == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{858, 503}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	imgui.Spacing()
	imgui.Text("Left")
	imgui.Spacing()
	win.drawPeripheral(win.img.lz.Peripherals.LeftPlayer, peripherals.AvailableLeftPlayer)
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Right")
	imgui.Spacing()
	win.drawPeripheral(win.img.lz.Peripherals.RightPlayer, peripherals.AvailableRightPlayer)
	imgui.EndGroup()

	imgui.End()
}

func (win *winPeripherals) drawPeripheral(p ports.Peripheral, periphList []string) {
	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV(fmt.Sprintf("##%v", p.PortID()), string(p.ID()), imgui.ComboFlagsNoArrowButton) {
		for _, s := range periphList {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("PERIPHERAL %s %s", p.PortID(), s)
				win.img.term.pushCommand(termCmd)
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()
}
