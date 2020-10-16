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
	"strconv"

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
)

const winSaveKeyEEPROMTitle = "SaveKey EEPROM"
const menuSaveKeyEEPROMTitle = "EEPROM"

type winSaveKeyEEPROM struct {
	windowManagement

	img *SdlImgui

	// height of status line at bottom of window. valid after first frame
	statusHeight float32
}

func newWinSaveKeyEEPROM(img *SdlImgui) (managedWindow, error) {
	win := &winSaveKeyEEPROM{img: img}
	return win, nil
}

func (win *winSaveKeyEEPROM) init() {
}

func (win *winSaveKeyEEPROM) destroy() {
}

func (win *winSaveKeyEEPROM) id() string {
	return winSaveKeyEEPROMTitle
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
	imgui.BeginV(winSaveKeyEEPROMTitle, &win.open, 0)

	win.drawGrid(win.img.lz.SaveKey.EEPROMdata)
	win.drawStatusLine()

	imgui.End()
}

func (win *winSaveKeyEEPROM) drawStatusLine() {
	statusHeight := imgui.CursorPosY()

	imgui.Spacing()
	imgui.Spacing()
	imgui.Spacing()

	if imgui.Button("Save to disk") {
		win.img.lz.Dbg.PushRawEvent(func() {
			if sk, ok := win.img.lz.Dbg.VCS.RIOT.Ports.Player1.(*savekey.SaveKey); ok {
				sk.EEPROM.Write()
			}
		})
	}
	imgui.SameLine()

	imgui.AlignTextToFramePadding()
	if win.img.lz.SaveKey.Dirty {
		imgui.Text("Data is NOT saved")
	} else {
		imgui.Text("Data is saved")
	}

	win.statusHeight = imgui.CursorPosY() - statusHeight
}

func (win *winSaveKeyEEPROM) drawGrid(a []byte) {
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})
	imgui.PushItemWidth(imguiTextWidth(2))

	// draw headers for each column
	headerDim := imgui.Vec2{X: imguiTextWidth(5), Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += imguiTextWidth(2)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	height := imguiRemainingWinHeight() - win.statusHeight
	imgui.BeginChildV("eeprom", imgui.Vec2{X: 0, Y: height}, false, 0)

	// draw rows
	var clipper imgui.ListClipper
	clipper.Begin(len(a) / 16)
	for clipper.Step() {
		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			offset := (i * 16)
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%03x- ", i))
			imgui.SameLine()

			for j := 0; j < 16; j++ {
				imgui.SameLine()
				win.drawEditByte(uint16(offset+j), a[offset+j])
			}
		}
	}

	imgui.EndChild()

	imgui.PopItemWidth()
	imgui.PopStyleVar()
}

func (win *winSaveKeyEEPROM) drawEditByte(address uint16, data byte) {
	l := fmt.Sprintf("##%d", address)
	content := fmt.Sprintf("%02x", data)

	if imguiHexInput(l, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() {
				if sk, ok := win.img.lz.Dbg.VCS.RIOT.Ports.Player1.(*savekey.SaveKey); ok {
					sk.EEPROM.Poke(address, uint8(v))
				}
			})
		}
	}
}
