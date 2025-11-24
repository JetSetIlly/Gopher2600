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

	"github.com/jetsetilly/gopher2600/hardware/peripherals"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/imgui-go/v5"
)

const winPeripheralsID = "Peripherals"

type winPeripherals struct {
	debuggerWin

	img *SdlImgui

	// required dimensions for controller dropdown
	controllerComboDim imgui.Vec2
	keyportariComboDim imgui.Vec2
}

func newWinPeripherals(img *SdlImgui) (window, error) {
	win := &winPeripherals{
		img: img,
	}

	return win, nil
}

func (win *winPeripherals) init() {
	win.controllerComboDim = imguiGetFrameDim("", peripherals.AvailableRightPlayer...)
	win.keyportariComboDim = imguiGetFrameDim("", peripherals.AvailableKeyportari...)
}

func (win *winPeripherals) id() string {
	return winPeripheralsID
}

func (win *winPeripherals) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// don't show the window if either of the controllers are unplugged
	// !!TODO: show something meaningful for unplugged controllers
	if win.img.cache.VCS.RIOT.Ports.LeftPlayer == nil || win.img.cache.VCS.RIOT.Ports.RightPlayer == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 858, Y: 503}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPeripherals) draw() {
	imgui.BeginGroup()
	imgui.Spacing()
	imgui.Text("Left")
	imgui.Spacing()
	win.drawPeripheral(win.img.cache.VCS.RIOT.Ports.LeftPlayer, peripherals.AvailableLeftPlayer)
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Right")
	imgui.Spacing()
	win.drawPeripheral(win.img.cache.VCS.RIOT.Ports.RightPlayer, peripherals.AvailableRightPlayer)
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	var protocol string
	if shim, ok := win.img.cache.VCS.RIOT.Ports.LeftPlayer.(ports.PeripheralShim); ok {
		protocol = shim.Protocol()
	} else {
		protocol = "None"
	}

	imgui.PushItemWidth(win.keyportariComboDim.X)
	imguiLabel("Keyportari")
	if imgui.BeginComboV("##keyportari", protocol, imgui.ComboFlagsNoArrowButton) {
		for _, s := range peripherals.AvailableKeyportari {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("KEYPORTARI %s", s)
				win.img.term.pushCommand(termCmd)
			}
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
}

func (win *winPeripherals) drawPeripheral(p ports.Peripheral, periphList []string) {
	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV(fmt.Sprintf("##controllers_%v", p.PortID()), string(p.ID()), imgui.ComboFlagsNoArrowButton) {
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
