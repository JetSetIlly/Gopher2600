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
	"github.com/inkyblackness/imgui-go/v3"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const winCartRAMTitle = "Cartridge RAM"

type winCartRAM struct {
	img  *SdlImgui
	open bool

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

func (win *winCartRAM) destroy() {
}

func (win *winCartRAM) id() string {
	return winCartRAMTitle
}

func (win *winCartRAM) menuLabel() string {
	return winCartRAMTitle
}

func (win *winCartRAM) isOpen() bool {
	return win.open
}

func (win *winCartRAM) setOpen(open bool) {
	win.open = open
}

func (win *winCartRAM) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasRAMbus {
		return
	}

	// get comparison data. assuming that there is such a thing and that it's
	// safe to get StaticData from.
	comp := win.img.lz.Rewind.Comparison.Mem.Cart.GetRAMbus().GetRAM()

	imgui.SetNextWindowPosV(imgui.Vec2{616, 524}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 402, Y: 232}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winCartRAMTitle, &win.open, 0)

	imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)
	for bank := range win.img.lz.Cart.RAM {
		a := win.img.lz.Cart.RAM[bank]
		b := comp[bank]
		if imgui.BeginTabItem(a.Label) {
			imgui.BeginChildV("scrollable", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.statusHeight}, false, 0)

			// show cartridge origin for mapped RAM banks
			origin := a.Origin
			if win.img.lz.Prefs.FxxxMirror {
				origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCartFxxxMirror
			}

			bnk := bank
			drawByteGrid(a.Data, b.Data, win.img.cols.ValueDiff, origin,
				func(addr uint16, data uint8) {
					win.img.lz.Dbg.PushRawEvent(func() {
						idx := int(addr - origin)
						win.img.lz.Dbg.VCS.Mem.Cart.GetRAMbus().PutRAM(bnk, idx, data)
					})
				})

			imgui.EndChild()

			// status line
			win.statusHeight = measureHeight(func() {
				if a.Mapped {
					imguiBooleanButtonV(win.img.cols, true, " mapped ", win.mappedIndicatorDim)
				} else {
					imguiBooleanButtonV(win.img.cols, false, " unmapped ", win.mappedIndicatorDim)
				}
			})

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()

	imgui.End()
}
