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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const winRAMTitle = "RAM"

type winRAM struct {
	windowManagement

	img *SdlImgui
}

func newWinRAM(img *SdlImgui) (managedWindow, error) {
	win := &winRAM{img: img}
	return win, nil
}

func (win *winRAM) init() {
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

	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})
	imgui.PushItemWidth(imguiTextWidth(2))

	// draw headers for each column
	headerDim := imgui.Vec2{X: imguiTextWidth(4), Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += imguiTextWidth(2)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	for i := 0; i < len(win.img.lz.RAM.RAM); i++ {
		addr := memorymap.OriginRAM + uint16(i)

		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", addr/16))
			imgui.SameLine()
		} else {
			imgui.SameLine()
		}

		// editable byte
		d := win.img.lz.RAM.RAM[i]

		// compare current RAM value with value in comparison snapshot and use
		// highlight color if it is different
		e := d
		if win.img.lz.Rewind.Comparison != nil {
			e = win.img.lz.Rewind.Comparison.Mem.RAM.RAM[i]
		}
		if d != e {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.RAMDiff)
		}

		b := fmt.Sprintf("%02x", d)
		if imguiHexInput(fmt.Sprintf("##%d", addr), !win.img.paused, 2, &b) {
			if v, err := strconv.ParseUint(b, 16, 8); err == nil {
				a := addr // we have to make a copy of the address
				win.img.lz.Dbg.PushRawEvent(func() {
					win.img.lz.Dbg.VCS.Mem.Write(a, uint8(v))
				})
			}
		}

		// undo any color changes
		if imgui.IsItemHovered() {
			win.drawSnapshotInfo(d, e)
		}

		if d != e {
			imgui.PopStyleColor()
		}
	}

	imgui.PopItemWidth()
	imgui.PopStyleVar()

	imgui.End()
}

func (win *winRAM) drawSnapshotInfo(current, snapshot uint8) {
	imgui.BeginTooltip()
	imgui.Text(fmt.Sprintf("%02x -> %02x", snapshot, current))
	imgui.EndTooltip()
}
