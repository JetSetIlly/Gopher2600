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

	"github.com/inkyblackness/imgui-go/v3"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
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
			// draw header info

			win.drawBank(bank, a, b)

			// status line
			statusHeight := imgui.CursorPosY()
			imgui.Text("")
			if a.Mapped {
				imguiBooleanButtonV(win.img.cols, true, " mapped ", win.mappedIndicatorDim)
			} else {
				imguiBooleanButtonV(win.img.cols, false, " unmapped ", win.mappedIndicatorDim)
			}
			win.statusHeight = imgui.CursorPosY() - statusHeight

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()

	imgui.End()
}

func (win *winCartRAM) drawBank(bank int, a mapper.CartRAM, b mapper.CartRAM) {
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})
	imgui.PushItemWidth(imguiTextWidth(2))

	// draw headers for each column
	headerDim := imgui.Vec2{X: imguiTextWidth(5), Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += imguiTextWidth(2)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	height := imguiRemainingWinHeight() - win.statusHeight
	imgui.BeginChildV(a.Label, imgui.Vec2{X: 0, Y: height}, false, 0)

	// show cartridge origin for mapped RAM banks
	origin := a.Origin
	if win.img.lz.Prefs.FxxxMirror {
		origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCartFxxxMirror
	}

	for i := range a.Data {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()

			// row header masked if RAM bank is not mapped
			if a.Mapped {
				imgui.Text(fmt.Sprintf("%03x- ", (i+int(origin))/16))
			} else {
				imgui.Text(fmt.Sprintf("x%02x- ", (i+int(origin&memorymap.CartridgeBits))/16))
			}
		}
		imgui.SameLine()

		// compare current RAM value with value in comparison snapshot and use
		// highlight color if it is different
		if a.Data[i] != b.Data[i] {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.RAMDiff)
		}

		// editable byte
		e := fmt.Sprintf("%02x", a.Data[i])
		if imguiHexInput(fmt.Sprintf("##%d", i), 2, &e) {
			if v, err := strconv.ParseUint(e, 16, 8); err == nil {
				a := i
				k := bank
				win.img.lz.Dbg.PushRawEvent(func() {
					win.img.lz.Dbg.VCS.Mem.Cart.GetRAMbus().PutRAM(k, a, uint8(v))
				})
			}
		}

		// undo any color changes
		if a.Data[i] != b.Data[i] {
			imgui.PopStyleColor()
		}
	}

	imgui.EndChild()

	imgui.PopItemWidth()
	imgui.PopStyleVar()
}
