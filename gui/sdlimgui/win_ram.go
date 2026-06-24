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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/imgui-go/v5"
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
		var isPXE bool
		if ef, ok := win.img.cache.VCS.Mem.Cart.GetCoProcBus().(coprocessor.CartCoProcELF); ok {
			isPXE, _ = ef.PXE()
		}
		win.draw(isPXE)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winRAM) draw(isPXE bool) {
	var diff []uint8
	if win.img.cache.Rewind.Comparison.State != nil {
		diff = win.img.cache.Rewind.Comparison.State.VCS.Mem.RAM.RAM
	} else {
		diff = win.img.cache.VCS.Mem.RAM.RAM
	}

	var pos imgui.Vec2
	var highlight bool

	before := func(idx int) {
		pos = imgui.CursorScreenPos()
		if uint16(win.img.cache.VCS.CPU.SP.Value())-memorymap.OriginRAM < uint16(idx) {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueStack)
			highlight = true
		}
	}

	after := func(idx int) bool {
		var tooltipDrawn bool

		if highlight {
			imgui.PopStyleColor()
			highlight = false
		}

		// idx is based on original values of type uint16 so the type conversion is safe
		addr := memorymap.OriginRAM + uint16(idx)

		if !isPXE {
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
					imguiColorLabel(read.Symbol, win.img.cols.ValueSymbol)
				}, true)
				tooltipDrawn = true
			} else {
				if okr {
					win.img.imguiTooltip(func() {
						imguiColorLabel(read.Symbol, win.img.cols.ValueSymbol)
					}, true)
					tooltipDrawn = true
				}
				if okw {
					win.img.imguiTooltip(func() {
						imguiColorLabel(write.Symbol, win.img.cols.ValueSymbol)
					}, true)
					tooltipDrawn = true
				}
			}
		}

		// not using Address() function because the stackpointer is hardwired to
		// page one addresses. the value in the register is what we need
		sp := uint16(win.img.cache.VCS.CPU.SP.Value())

		// idx is based on original values of type uint16 so the type conversion is safe
		if sp-memorymap.OriginRAM < uint16(idx) {
			win.img.imguiTooltip(func() {
				imguiColorLabel("in stack", win.img.cols.ValueStack)
				if v, ok := win.img.cache.VCS.CPU.PredictRTS(); ok {
					imgui.Spacing()
					imgui.Text(fmt.Sprintf("PC address in event of RTS: %04x", v))
				}
			}, true)
			tooltipDrawn = true
		}

		return tooltipDrawn
	}

	commit := func(idx int, data uint8) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().Mem.Poke(memorymap.OriginRAM+uint16(idx), data)
		})
	}

	win.img.drawByteGrid("ramByteGrid", byteGridConfig{
		origin: uint32(memorymap.OriginRAM),
		data:   win.img.cache.VCS.Mem.RAM.RAM,
		diff:   diff,
		commit: commit,
		before: before,
		after:  after,
	})
}
