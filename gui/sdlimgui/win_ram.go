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
	"github.com/jetsetilly/gopher2600/gui/fonts"
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

	var diff []uint8
	if win.img.lz.Rewind.Comparison != nil {
		diff = win.img.lz.Rewind.Comparison.Mem.RAM.RAM
	} else {
		diff = win.img.lz.RAM.RAM
	}

	// item spacing is altered in drawByteGrid(). note value now so we can set
	// it for tooltips in after()
	tooltipSpacing := imgui.CurrentStyle().ItemSpacing()

	// pos is retreived in before() and used in after()
	var pos imgui.Vec2

	// number of colors to pop in afer()
	popColor := 0

	before := func(offset uint16) {
		pos = imgui.CursorScreenPos()

		a := diff[offset]
		b := win.img.lz.RAM.RAM[offset]
		if a != b {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
			popColor++
		}

		if uint16(win.img.lz.CPU.SP.Value())-memorymap.OriginRAM < offset {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueStack)
			popColor++
		}
	}

	after := func(offset uint16) {
		imgui.PopStyleColorV(popColor)
		popColor = 0

		dl := imgui.WindowDrawList()
		addr := memorymap.OriginRAM + offset
		read, okr := win.img.dbg.Disasm.Sym.GetSymbol(addr, true)
		write, okw := win.img.dbg.Disasm.Sym.GetSymbol(addr, false)
		if okr || okw {
			sz := imgui.FontSize() * 0.4
			pos.X += 1.0
			pos.Y += 1.0
			p1 := pos
			p1.Y += sz
			p2 := pos
			p2.X += sz
			dl.AddTriangleFilled(pos, p1, p2, imgui.PackedColorFromVec4(win.img.cols.ValueSymbol))
		}

		imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, tooltipSpacing)
		defer imgui.PopStyleVar()

		if okr && okw && read.Symbol == write.Symbol {
			imguiTooltip(func() {
				imguiColorLabel(read.Symbol, win.img.cols.ValueSymbol)
			}, true)
		} else {
			if okr {
				imguiTooltip(func() {
					imguiColorLabel(read.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
			if okw {
				imguiTooltip(func() {
					imguiColorLabel(write.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
		}

		a := diff[addr^memorymap.OriginRAM]
		b := win.img.lz.RAM.RAM[addr^memorymap.OriginRAM]
		if a != b {
			imguiTooltip(func() {
				imguiColorLabel(fmt.Sprintf("%02x %c %02x", a, fonts.ByteChange, b), win.img.cols.ValueDiff)
			}, true)
		}

		sp := win.img.lz.CPU.SP.Address()
		if sp-memorymap.OriginRAM < offset {
			imguiTooltip(func() {
				imguiColorLabel("in stack", win.img.cols.ValueStack)
				imgui.Spacing()
				imgui.Text(fmt.Sprintf("PC address in event of RTS: %04x", win.img.lz.CPU.RTSPrediction))
			}, true)
		}
	}

	commit := func(addr uint16, data uint8) {
		win.img.dbg.PushRawEvent(func() {
			win.img.vcs.Mem.Write(addr, data)
		})
	}

	drawByteGrid(win.img.lz.RAM.RAM, memorymap.OriginRAM, before, after, commit)

	imgui.End()
}
