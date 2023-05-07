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
	debuggerWin
	img *SdlImgui
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

func (win *winRAM) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{1081, 607}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winRAM) draw() {
	var diff []uint8
	if win.img.lz.Rewind.Comparison != nil {
		diff = win.img.lz.Rewind.Comparison.Mem.RAM.RAM
	} else {
		diff = win.img.lz.RAM.RAM
	}

	// pos is retreived in before() and used in after()
	var pos imgui.Vec2

	// number of colors to pop in afer()
	popColor := 0

	before := func(idx int) {
		pos = imgui.CursorScreenPos()

		a := diff[idx]
		b := win.img.lz.RAM.RAM[idx]
		if a != b {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
			popColor++
		}

		// idx is based on original values of type uint16 so the type conversion is safe
		if uint16(win.img.lz.CPU.SP.Value())-memorymap.OriginRAM < uint16(idx) {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueStack)
			popColor++
		}
	}

	after := func(idx int) {
		imgui.PopStyleColorV(popColor)
		popColor = 0

		// idx is based on original values of type uint16 so the type conversion is safe
		addr := memorymap.OriginRAM + uint16(idx)

		dl := imgui.WindowDrawList()
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

		if okr && okw && read.Symbol == write.Symbol {
			imguiTooltip(func() {
				imguiColorLabelSimple(read.Symbol, win.img.cols.ValueSymbol)
			}, true)
		} else {
			if okr {
				imguiTooltip(func() {
					imguiColorLabelSimple(read.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
			if okw {
				imguiTooltip(func() {
					imguiColorLabelSimple(write.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
		}

		a := diff[idx]
		b := win.img.lz.RAM.RAM[idx]
		if a != b {
			imguiTooltip(func() {
				imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", a, fonts.ByteChange, b), win.img.cols.ValueDiff)
			}, true)
		}

		// not using Address() function because the stackpointer is hardwired to
		// page one addresses. the value in the register is what we need
		sp := uint16(win.img.lz.CPU.SP.Value())

		// idx is based on original values of type uint16 so the type conversion is safe
		if sp-memorymap.OriginRAM < uint16(idx) {
			imguiTooltip(func() {
				imguiColorLabelSimple("in stack", win.img.cols.ValueStack)
				if win.img.lz.CPU.RTSPredictionValid {
					imgui.Spacing()
					imgui.Text(fmt.Sprintf("PC address in event of RTS: %04x", win.img.lz.CPU.RTSPrediction))
				}
			}, true)
		}
	}

	commit := func(idx int, data uint8) {
		win.img.dbg.PushFunction(func() {
			win.img.vcs.Mem.Write(memorymap.OriginRAM+uint16(idx), data)
		})
	}

	drawByteGrid("ramByteGrid", win.img.lz.RAM.RAM, uint32(memorymap.OriginRAM), before, after, commit)
}
