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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
)

const winSaveKeyEEPROMID = "SaveKey EEPROM"
const winSaveKeyEEPROMMenu = "EEPROM"

type winSaveKeyEEPROM struct {
	img  *SdlImgui
	open bool

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

func (win *winSaveKeyEEPROM) isOpen() bool {
	return win.open
}

func (win *winSaveKeyEEPROM) setOpen(open bool) {
	win.open = open
}

func (win *winSaveKeyEEPROM) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.SaveKey.SaveKeyActive {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{469, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{394, 356}, imgui.ConditionFirstUseEver)
	imgui.BeginV(win.id(), &win.open, 0)

	imgui.BeginChildV("eepromData", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.statusHeight}, false, 0)

	drawByteGrid(win.img.lz.SaveKey.EEPROMdata, win.img.lz.SaveKey.EEPROMdiskData, win.img.cols.ValueDiff, 0x00,
		func(addr uint16, data uint8) {
			win.img.dbg.PushRawEvent(func() {
				if sk, ok := win.img.vcs.RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
					sk.EEPROM.Poke(addr, data)
				}
			})
		})

	imgui.EndChild()

	win.statusHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Spacing()

		if imgui.Button("Save to disk") {
			win.img.dbg.PushRawEvent(func() {
				if sk, ok := win.img.vcs.RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
					sk.EEPROM.Write()
				}
			})
		}
	})

	imgui.End()
}
