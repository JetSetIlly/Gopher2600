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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/controllers"
)

const winControllersTitle = "Controllers"

type winControllers struct {
	windowManagement
	img *SdlImgui

	controllerComboDim imgui.Vec2
}

func newWinControllers(img *SdlImgui) (managedWindow, error) {
	win := &winControllers{
		img: img,
	}

	return win, nil
}

func (win *winControllers) init() {
	win.controllerComboDim = imguiGetFrameDim("", controllers.ControllerList...)
}

func (win *winControllers) destroy() {
}

func (win *winControllers) id() string {
	return winControllersTitle
}

func (win *winControllers) draw() {
	if !win.open {
		return
	}

	// don't show the window if either of the controllers are unplugged
	// !!TODO: show something meaningful for unplugged controllers
	if win.img.lz.Controllers.Player0 == nil || win.img.lz.Controllers.Player1 == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{677, 538}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winControllersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	imgui.Spacing()
	imgui.Text("Player 0")
	imgui.Spacing()
	win.drawController(0)
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Player 1")
	imgui.Spacing()
	win.drawController(1)
	imgui.EndGroup()

	imgui.End()
}

func (win *winControllers) drawController(player int) {
	var p ports.Peripheral

	switch player {
	case 0:
		p = win.img.lz.Controllers.Player0
	case 1:
		p = win.img.lz.Controllers.Player1
	}

	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV(fmt.Sprintf("##%d", player), p.Name(), imgui.ComboFlagNoArrowButton) {
		for _, s := range controllers.ControllerList {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("CONTROLLER %d %s", player, s)
				win.img.term.pushCommand(termCmd)
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	_, auto := p.(*controllers.Auto)
	if imgui.Checkbox(fmt.Sprintf("Auto##%d", player), &auto) {
		var termCmd string
		if auto {
			termCmd = fmt.Sprintf("CONTROLLER %d AUTO", player)
		} else {
			termCmd = fmt.Sprintf("CONTROLLER %d %s", player, p.Name())
		}
		win.img.term.pushCommand(termCmd)
	}
}
