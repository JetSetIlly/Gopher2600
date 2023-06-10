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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
)

const winSaveKeyEEPROMID = "SaveKey EEPROM"
const winSaveKeyEEPROMMenu = "EEPROM"

type winSaveKeyEEPROM struct {
	debuggerWin

	img *SdlImgui

	// height of status line at bottom of window. valid after first frame
	statusHeight float32
}

func newWinSaveKeyEEPROM(img *SdlImgui) (window, error) {
	win := &winSaveKeyEEPROM{img: img}
	return win, nil
}

func (win *winSaveKeyEEPROM) init() {
}

func (win *winSaveKeyEEPROM) id() string {
	return winSaveKeyEEPROMID
}

func (win *winSaveKeyEEPROM) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	if !win.img.lz.SaveKey.SaveKeyActive {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{469, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{478, 356}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{478, 271}, imgui.Vec2{529, 1000})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSaveKeyEEPROM) draw() {
	imgui.BeginChildV("eepromData", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.statusHeight}, false, 0)

	drawByteGridSimple("eepromByteGrid", win.img.lz.SaveKey.EEPROMdata, win.img.lz.SaveKey.EEPROMdiskData, win.img.cols.ValueDiff, 0x00, func(idx int, data uint8) {
		win.img.dbg.PushFunction(func() {
			var sk *savekey.SaveKey

			if av, ok := win.img.vcs.RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
				sk = av.SaveKey.(*savekey.SaveKey)
			} else {
				sk = win.img.vcs.RIOT.Ports.RightPlayer.(*savekey.SaveKey)
			}

			if sk != nil {
				// eeprom space is maximum of uint16 so the type conversion is safe
				sk.EEPROM.Poke(uint16(idx), data)
			}
		})
	})

	imgui.EndChild()

	win.statusHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Spacing()

		if imgui.Button("Save to disk") {
			win.img.dbg.PushFunction(func() {
				var sk *savekey.SaveKey

				if av, ok := win.img.vcs.RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
					sk = av.SaveKey.(*savekey.SaveKey)
				} else {
					sk = win.img.vcs.RIOT.Ports.RightPlayer.(*savekey.SaveKey)
				}

				if sk != nil {
					sk.EEPROM.Write()
				}
			})
		}
	})
}
