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
	"github.com/jetsetilly/gopher2600/hardware/peripherals/controllers"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
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
	imgui.AlignTextToFramePadding()
	imgui.BeginGroup()
	imgui.Text("Left")
	win.drawSelection(win.img.cache.VCS.RIOT.Ports.LeftPlayer, peripherals.AvailableLeftPlayer)
	imgui.EndGroup()

	imgui.SameLine()

	imgui.BeginGroup()
	imgui.Text("Right")
	win.drawSelection(win.img.cache.VCS.RIOT.Ports.RightPlayer, peripherals.AvailableRightPlayer)
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

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	x, ok := win.drawPeripheral(false, true)
	imgui.SameLineV(x+imgui.CurrentStyle().ItemSpacing().X*4, 0.0)
	_, _ = win.drawPeripheral(true, ok)
}

func (win *winPeripherals) drawSelection(p ports.Peripheral, periphList []string) {
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

// drawNoVisualisation indicates that the "no visualisation" message should be printed if no
// suitable visualisation is available
func (win *winPeripherals) drawPeripheral(right bool, drawNoVisualisation bool) (float32, bool) {
	x := imgui.CursorScreenPos().X

	var p ports.Peripheral
	var port string
	var id string

	if right {
		p = win.img.cache.VCS.RIOT.Ports.RightPlayer
		id = "##rightStickVisualisation"
		port = "RIGHT"
	} else {
		p = win.img.cache.VCS.RIOT.Ports.LeftPlayer
		id = "##leftStickVisualisation"
		port = "LEFT"
	}

	switch p.ID() {
	case plugging.PeriphStick, plugging.PeriphGamepad:
		var axis [4]bool
		var fire bool
		var second bool
		if p.ID() == plugging.PeriphStick {
			axis, fire = p.(*controllers.Stick).State()
		}
		if p.ID() == plugging.PeriphGamepad {
			axis, fire, second = p.(*controllers.Gamepad).State()
		}

		if imgui.BeginTable(id, 3) {
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			if imgui.Checkbox(fmt.Sprintf("%sUp", id), &axis[0]) {
				if axis[0] {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s UP", port))
				} else {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s NOUP", port))
				}
			}
			imgui.TableNextColumn()

			imgui.TableNextRow()
			imgui.TableNextColumn()
			if imgui.Checkbox(fmt.Sprintf("%sLeft", id), &axis[1]) {
				if axis[1] {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s LEFT", port))
				} else {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s NOLEFT", port))
				}
			}
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			if imgui.Checkbox(fmt.Sprintf("%sRight", id), &axis[2]) {
				if axis[2] {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s RIGHT", port))
				} else {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s NORIGHT", port))
				}
			}

			// we take the x position from the right most widget in the table
			imgui.SameLine()
			x = imgui.CursorScreenPos().X - x

			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			if imgui.Checkbox(fmt.Sprintf("%sDown", id), &axis[3]) {
				if axis[3] {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s DOWN", port))
				} else {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s NODOWN", port))
				}
			}
			imgui.TableNextColumn()

			imgui.TableNextRow()
			imgui.TableNextColumn()
			if imgui.Checkbox(fmt.Sprintf("%sFire", id), &fire) {
				if fire {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s FIRE", port))
				} else {
					win.img.term.pushCommand(fmt.Sprintf("STICK %s NOFIRE", port))
				}
			}

			if p.ID() == plugging.PeriphGamepad {
				imgui.TableNextColumn()
				imgui.TableNextColumn()
				if imgui.Checkbox(fmt.Sprintf("%sSecond", id), &second) {
					if second {
						win.img.term.pushCommand(fmt.Sprintf("STICK %s SECOND", port))
					} else {
						win.img.term.pushCommand(fmt.Sprintf("STICK %s NOSECOND", port))
					}
				}
			}

			imgui.EndTable()
		}
		return x, true
	}

	if drawNoVisualisation {
		imgui.Text(p.String())
		imgui.SameLine()
		x = imgui.CursorScreenPos().X - x
		return x, false
	}

	return 0, false
}
