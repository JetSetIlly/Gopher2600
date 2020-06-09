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
	"strconv"

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const winRAMTitle = "RAM"

type winRAM struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	xPos float32

	// height required to display VCS RAM in its entirity (we know this value
	// after the first pass)
	gridHeight float32
}

func newWinRAM(img *SdlImgui) (managedWindow, error) {
	win := &winRAM{img: img}
	return win, nil
}

func (win *winRAM) init() {
	win.widgetDimensions.init()
}

func (win *winRAM) destroy() {
}

func (win *winRAM) id() string {
	return winRAMTitle
}

func (win *winRAM) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{890, 29}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winRAMTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies xPos, which requires
	// one frame before it is accurate.
	headerDim := imgui.Vec2{X: win.xPos, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.twoDigitDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	imgui.PushItemWidth(win.twoDigitDim.X)
	i := uint16(0)
	for addr := memorymap.OriginRAM; addr <= memorymap.MemtopRAM; addr++ {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", addr/16))
			imgui.SameLine()
			win.xPos = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}

		// editable byte
		b := fmt.Sprintf("%02x", win.img.lz.RAM.Read(addr))
		if imguiHexInput(fmt.Sprintf("##%d", addr), !win.img.paused, 2, &b) {
			if v, err := strconv.ParseUint(b, 16, 8); err == nil {
				a := addr // we have to make a copy of the address
				win.img.lz.Dbg.PushRawEvent(func() {
					win.img.lz.Dbg.VCS.Mem.Write(a, uint8(v))
				})
			}
		}

		i++
	}
	imgui.PopItemWidth()

	imgui.PopStyleVar()
	imgui.End()
}
