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

const winCartRAMTitle = "Cartridge RAM"

type winCartRAM struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	xPos float32

	// height of status line at bottom of frame. valid after first frame of a
	// tab (although it should be same for each tab)
	statusHeight float32

	// height required to display VCS RAM in its entirity (we know this value
	// after the first pass)
	gridHeight float32

	// width of mapped/unmapped indicator
	mappedIndicatorDim imgui.Vec2
}

func newWinCartRAM(img *SdlImgui) (managedWindow, error) {
	win := &winCartRAM{img: img}
	return win, nil
}

func (win *winCartRAM) init() {
	win.widgetDimensions.init()
	win.mappedIndicatorDim = imguiGetFrameDim(" mapped ", " unmapped ")
}

func (win *winCartRAM) destroy() {
}

func (win *winCartRAM) id() string {
	return winCartRAMTitle
}

func (win *winCartRAM) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasRAMbus {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{616, 524}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 402, Y: 232}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winCartRAMTitle, &win.open, 0)

	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)
	for bank := 0; bank < len(win.img.lz.Cart.RAM); bank++ {
		if imgui.BeginTabItem(win.img.lz.Cart.RAM[bank].Label) {
			// draw header info

			win.drawBank(bank)

			// status line
			statusHeight := imgui.CursorPosY()
			imgui.Text("")
			if win.img.lz.Cart.RAM[bank].Mapped {
				imguiBooleanButtonV(win.img.cols, true, " mapped ", win.mappedIndicatorDim)
			} else {
				imguiBooleanButtonV(win.img.cols, false, " unmapped ", win.mappedIndicatorDim)
			}
			imgui.Spacing()
			win.statusHeight = imgui.CursorPosY() - statusHeight

			imgui.EndTabItem()
		}

	}
	imgui.EndTabBar()

	imgui.PopStyleVar()
	imgui.End()
}

func (win *winCartRAM) drawBank(bank int) {
	// draw headers for each column. this relies xPos, which requires one frame
	// before it is accurate. we also need to add some padding because the xPos
	// was calculated inside a child element.
	x := win.xPos + imgui.CurrentStyle().FramePadding().X*2

	headerDim := imgui.Vec2{X: x, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.twoDigitDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// put rest of window in a child element which will scroll, leaving the
	// header static
	height := imguiRemainingWinHeight() - win.statusHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", bank), imgui.Vec2{X: 0, Y: height}, false, 0)

	// draw rows
	imgui.PushItemWidth(win.twoDigitDim.X)

	// show cartridge origin for mapped RAM banks
	origin := win.img.lz.Cart.RAM[bank].Origin
	if win.img.lz.Prefs.FxxxMirror {
		origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCartFxxxMirror
	}

	for i := 0; i < len(win.img.lz.Cart.RAM[bank].Data); i++ {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()

			// row header masked if RAM bank is not mapped
			if win.img.lz.Cart.RAM[bank].Mapped {
				imgui.Text(fmt.Sprintf("%03x- ", (i+int(origin))/16))
			} else {
				imgui.Text(fmt.Sprintf("x%02x- ", (i+int(origin&memorymap.CartridgeBits))/16))
			}

			imgui.SameLine()
			win.xPos = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}

		// editable byte
		b := fmt.Sprintf("%02x", win.img.lz.Cart.RAM[bank].Data[i])
		if imguiHexInput(fmt.Sprintf("##%d", i), !win.img.paused, 2, &b) {
			if v, err := strconv.ParseUint(b, 16, 8); err == nil {
				a := i
				k := bank
				win.img.lz.Dbg.PushRawEvent(func() {
					win.img.lz.Cart.RAMbus.PutRAM(k, a, uint8(v))
				})
			}
		}
	}
	imgui.PopItemWidth()

	imgui.EndChild()
}
