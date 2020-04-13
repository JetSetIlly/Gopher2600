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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/riot/input"
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
	win.controllerComboDim = imguiGetFrameDim("", input.ControllerTypeList...)
}

func (win *winControllers) destroy() {
}

func (win *winControllers) id() string {
	return winControllersTitle
}

// draw is called by service loop
func (win *winControllers) draw() {
	if !win.open {
		return
	}

	if win.img.lazy.Controller.HandController0 == nil ||
		win.img.lazy.Controller.HandController1 == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{677, 538}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winControllersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	imgui.Spacing()
	imgui.Text("Left")
	imgui.Spacing()

	c := win.img.lazy.Controller.HandController0.ControllerType.String()

	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV("##handController0", c, imgui.ComboFlagNoArrowButton) {
		for _, s := range input.ControllerTypeList {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("CONTROLLER 0 %s", s)
				win.img.term.pushCommand(termCmd)
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	auto := win.img.lazy.Controller.HandController0.AutoControllerType
	if imgui.Checkbox("Auto##auto0", &auto) {
		var termCmd string
		if auto {
			termCmd = fmt.Sprintf("CONTROLLER 0 AUTO")
		} else {
			termCmd = fmt.Sprintf("CONTROLLER 0 NOAUTO")
		}
		win.img.term.pushCommand(termCmd)
	}
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Right")
	imgui.Spacing()

	c = win.img.lazy.Controller.HandController1.ControllerType.String()

	imgui.PushItemWidth(win.controllerComboDim.X)
	if imgui.BeginComboV("##handController1", c, imgui.ComboFlagNoArrowButton) {
		for _, s := range input.ControllerTypeList {
			if imgui.Selectable(s) {
				termCmd := fmt.Sprintf("CONTROLLER 1 %s", s)
				win.img.term.pushCommand(termCmd)
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	auto = win.img.lazy.Controller.HandController1.AutoControllerType
	if imgui.Checkbox("Auto##auto1", &auto) {
		var termCmd string
		if auto {
			termCmd = fmt.Sprintf("CONTROLLER 1 AUTO")
		} else {
			termCmd = fmt.Sprintf("CONTROLLER 1 NOAUTO")
		}
		win.img.term.pushCommand(termCmd)
	}
	imgui.EndGroup()

	imgui.End()
}
