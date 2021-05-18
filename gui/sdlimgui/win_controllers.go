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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/controllers"
)

const winControllersID = "Controllers"

type winControllers struct {
	img  *SdlImgui
	open bool

	// required dimensions for controller dropdown
	controllerComboDim imgui.Vec2
}

func newWinControllers(img *SdlImgui) (window, error) {
	win := &winControllers{
		img: img,
	}

	return win, nil
}

func (win *winControllers) init() {
	win.controllerComboDim = imguiGetFrameDim("", controllers.ControllerList...)
}

func (win *winControllers) id() string {
	return winControllersID
}

func (win *winControllers) isOpen() bool {
	return win.open
}

func (win *winControllers) setOpen(open bool) {
	win.open = open
}

func (win *winControllers) draw() {
	if !win.open {
		return
	}

	// don't show the window if either of the controllers are unplugged
	// !!TODO: show something meaningful for unplugged controllers
	if win.img.lz.Controllers.LeftPlayer == nil || win.img.lz.Controllers.RightPlayer == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{858, 503}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	imgui.Spacing()
	imgui.Text("Left")
	imgui.Spacing()
	win.drawController(win.img.lz.Controllers.LeftPlayer)
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Right")
	imgui.Spacing()
	win.drawController(win.img.lz.Controllers.RightPlayer)
	imgui.EndGroup()

	imgui.End()
}

func (win *winControllers) drawController(p ports.Peripheral) {
	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV(fmt.Sprintf("##%v", p.PortID()), p.Name(), imgui.ComboFlagsNoArrowButton) {
		for _, s := range controllers.ControllerList {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("CONTROLLER %s %s", p.PortID(), s)
				win.img.term.pushCommand(termCmd)
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	_, auto := p.(*controllers.Auto)
	if imgui.Checkbox(fmt.Sprintf("Auto##%v", p.PortID()), &auto) {
		var termCmd string
		if auto {
			termCmd = fmt.Sprintf("CONTROLLER %s AUTO", p.PortID())
		} else {
			termCmd = fmt.Sprintf("CONTROLLER %s %s", p.PortID(), p.Name())
		}
		win.img.term.pushCommand(termCmd)
	}
}
