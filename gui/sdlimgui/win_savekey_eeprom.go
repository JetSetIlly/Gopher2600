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
	widgetDimensions

	img *SdlImgui

	// height of status line at bottom of window. valid after first frame
	statusHeight float32

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	xPos float32
}

func newWinSaveKeyEEPROM(img *SdlImgui) (managedWindow, error) {
	win := &winSaveKeyEEPROM{img: img}
	return win, nil
}

func (win *winSaveKeyEEPROM) init() {
	win.widgetDimensions.init()
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

	win.drawGrid(winSaveKeyEEPROMTitle, win.img.lz.SaveKey.EEPROMdata)
	win.drawStatusLine()

	imgui.End()
}

func (win *winSaveKeyEEPROM) drawStatusLine() {
	statusHeight := imgui.CursorPosY()

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

func (win *winSaveKeyEEPROM) drawGrid(tag string, a []byte) {
	const numberOfColumns = 16

	height := imguiRemainingWinHeight() - win.statusHeight
	imgui.BeginChildV(tag, imgui.Vec2{X: 0, Y: height}, false, 0)

	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies on win.xPos, which requires
	// one frame before it is accurate.
	headerDim := imgui.Vec2{X: win.xPos, Y: imgui.CursorPosY()}
	for i := 0; i < numberOfColumns; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.twoDigitDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	var clipper imgui.ListClipper
	clipper.Begin(len(a) / numberOfColumns)
	for clipper.Step() {
		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			offset := (i * numberOfColumns)
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%03x- ", i))
			imgui.SameLine()
			win.xPos = imgui.CursorPosX()

			imgui.PushItemWidth(win.twoDigitDim.X)
			for j := 0; j < numberOfColumns; j++ {
				imgui.SameLine()
				win.drawEditByte(tag, uint16(offset+j), a[offset+j])
			}
			imgui.PopItemWidth()
		}
	}

	// finished with spacing setting
	imgui.PopStyleVar()

	imgui.EndChild()
}

func (win *winSaveKeyEEPROM) drawEditByte(tag string, address uint16, data byte) {
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
