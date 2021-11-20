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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const winRAMID = "RAM"

type winRAM struct {
	img  *SdlImgui
	open bool
}

func newWinRAM(img *SdlImgui) (window, error) {
	win := &winRAM{img: img}
	return win, nil
}

func (win *winRAM) init() {
}

func (win *winRAM) id() string {
	return winRAMID
}

func (win *winRAM) isOpen() bool {
	return win.open
}

func (win *winRAM) setOpen(open bool) {
	win.open = open
}

func (win *winRAM) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{1081, 607}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})
	imgui.PushItemWidth(imguiTextWidth(2))

	var cmp []uint8
	if win.img.lz.Rewind.Comparison != nil {
		cmp = win.img.lz.Rewind.Comparison.Mem.RAM.RAM
	}

	drawByteGrid(win.img.lz.RAM.RAM, cmp, win.img.cols.ValueDiff, memorymap.OriginRAM,
		func(addr uint16, data uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Write(addr, data)
			})
		})

	imgui.PopItemWidth()
	imgui.PopStyleVar()

	imgui.End()
}
