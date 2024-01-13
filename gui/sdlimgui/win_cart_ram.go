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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const winCartRAMID = "Cartridge RAM"

type winCartRAM struct {
	debuggerWin

	img *SdlImgui

	// height of status line at bottom of frame. valid after first frame of a
	// tab (although it should be same for each tab)
	statusHeight float32

	// required dimensions of mapped/unmapped indicator
	mappedIndicatorDim imgui.Vec2
}

func newWinCartRAM(img *SdlImgui) (window, error) {
	win := &winCartRAM{img: img}
	return win, nil
}

func (win *winCartRAM) init() {
	win.mappedIndicatorDim = imguiGetFrameDim(" mapped ", " unmapped ")
}

func (win *winCartRAM) id() string {
	return winCartRAMID
}

func (win *winCartRAM) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no valid cartridge debug bus available
	bus := win.img.cache.VCS.Mem.Cart.GetRAMbus()
	if bus == nil {
		return false
	}
	ram := bus.GetRAM()

	imgui.SetNextWindowPosV(imgui.Vec2{533, 430}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{478, 271}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", win.img.cache.VCS.Mem.Cart.ID(), win.id())
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw(ram)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCartRAM) draw(ram []mapper.CartRAM) {
	// get comparison data. assuming that there is such a thing and that it's
	// safe to get StaticData from.
	comp := win.img.cache.Rewind.Comparison.State.Mem.Cart.GetRAMbus().GetRAM()

	imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)
	for bank := range ram {
		current := ram[bank]
		diff := comp[bank]
		if imgui.BeginTabItem(current.Label) {
			imgui.BeginChildV("cartram", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.statusHeight}, false, 0)

			// show cartridge origin for mapped RAM banks
			origin := current.Origin
			if win.img.dbg.Disasm.Prefs.FxxxMirror.Get().(bool) {
				origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCartFxxxMirror
			}

			// pos is retreived in before() and used in after()
			var pos imgui.Vec2

			// number of colors to pop in afer()
			popColor := 0

			before := func(idx int) {
				pos = imgui.CursorScreenPos()

				a := diff.Data[idx]
				b := current.Data[idx]
				if a != b {
					imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
					popColor++
				}
			}

			after := func(idx int) {
				imgui.PopStyleColorV(popColor)
				popColor = 0

				dl := imgui.WindowDrawList()

				// idx is based on original values of type uint16 so the type conversion is safe
				addr := memorymap.OriginCart + uint16(idx)

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

				a := diff.Data[idx]
				b := current.Data[idx]
				if a != b {
					win.img.imguiTooltip(func() {
						imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", a, fonts.ByteChange, b), win.img.cols.ValueDiff)
					}, true)
				}
			}

			commitBank := bank
			commit := func(idx int, data uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Cart.GetRAMbus().PutRAM(commitBank, idx, data)
				})
			}

			drawByteGrid("cartRamByteGrid", current.Data, uint32(origin), before, after, commit)
			imgui.EndChild()

			// status line
			win.statusHeight = imguiMeasureHeight(func() {
				imgui.PushStyleVarFloat(imgui.StyleVarFrameRounding, readOnlyButtonRounding)
				if current.Mapped {
					imguiColourButton(win.img.cols.True, " mapped ", win.mappedIndicatorDim)
				} else {
					imguiColourButton(win.img.cols.False, " unmapped ", win.mappedIndicatorDim)
				}
				imgui.PopStyleVar()
			})

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()
}
