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

func (win *winRAM) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 1081, Y: 607}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winRAM) draw() {
	var diff []uint8
	if win.img.cache.Rewind.Comparison.State != nil {
		diff = win.img.cache.Rewind.Comparison.State.VCS.Mem.RAM.RAM
	} else {
		diff = win.img.cache.VCS.Mem.RAM.RAM
	}

	// pos is retreived in before() and used in after()
	var pos imgui.Vec2

	// number of colors to pop in afer()
	popColor := 0

	before := func(idx int) {
		pos = imgui.CursorScreenPos()

		a := diff[idx]
		b := win.img.cache.VCS.Mem.RAM.RAM[idx]
		if a != b {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
			popColor++
		}

		// idx is based on original values of type uint16 so the type conversion is safe
		if uint16(win.img.cache.VCS.CPU.SP.Value())-memorymap.OriginRAM < uint16(idx) {
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
		read, okr := win.img.dbg.Disasm.Sym.GetReadSymbol(addr, false)
		write, okw := win.img.dbg.Disasm.Sym.GetWriteSymbol(addr)
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
			win.img.imguiTooltip(func() {
				imguiColorLabelSimple(read.Symbol, win.img.cols.ValueSymbol)
			}, true)
		} else {
			if okr {
				win.img.imguiTooltip(func() {
					imguiColorLabelSimple(read.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
			if okw {
				win.img.imguiTooltip(func() {
					imguiColorLabelSimple(write.Symbol, win.img.cols.ValueSymbol)
				}, true)
			}
		}

		a := diff[idx]
		b := win.img.cache.VCS.Mem.RAM.RAM[idx]
		if a != b {
			win.img.imguiTooltip(func() {
				imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", a, fonts.ByteChange, b), win.img.cols.ValueDiff)
			}, true)
		}

		// not using Address() function because the stackpointer is hardwired to
		// page one addresses. the value in the register is what we need
		sp := uint16(win.img.cache.VCS.CPU.SP.Value())

		// idx is based on original values of type uint16 so the type conversion is safe
		if sp-memorymap.OriginRAM < uint16(idx) {
			win.img.imguiTooltip(func() {
				imguiColorLabelSimple("in stack", win.img.cols.ValueStack)
				if v, ok := win.img.cache.VCS.CPU.PredictRTS(); ok {
					imgui.Spacing()
					imgui.Text(fmt.Sprintf("PC address in event of RTS: %04x", v))
				}
			}, true)
		}
	}

	commit := func(idx int, data uint8) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().Mem.Poke(memorymap.OriginRAM+uint16(idx), data)
		})
	}

	drawByteGrid("ramByteGrid", win.img.cache.VCS.Mem.RAM.RAM, uint32(memorymap.OriginRAM), before, after, commit)
}
