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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"strconv"

	"github.com/inkyblackness/imgui-go/v2"
)

const winStaticTitle = "Static"

type winStatic struct {
	windowManagement
	img *SdlImgui

	// widget dimensions
	byteDim imgui.Vec2

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	headerStartX float32
}

func newWinStatic(img *SdlImgui) (managedWindow, error) {
	win := &winStatic{img: img}

	return win, nil
}

func (win *winStatic) init() {
	win.byteDim = imguiGetFrameDim("FF")
}

func (win *winStatic) destroy() {
}

func (win *winStatic) id() string {
	return winStaticTitle
}

func (win *winStatic) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{469, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{394, 356}, imgui.ConditionFirstUseEver)

	imgui.BeginV(winStaticTitle, &win.open, 0)

	if win.img.lz.Cart.StaticAreaPresent {
		win.drawGrid()
	} else {
		imgui.Text("Cartridge has no static memory")
	}

	imgui.End()
}

func (win *winStatic) drawGrid() {
	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies headerStartX, which requires
	// one frame before it is accurate.
	headerDim := imgui.Vec2{X: win.headerStartX, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.byteDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	imgui.PushItemWidth(win.byteDim.X)
	i := uint16(0)
	for addr := 0; addr < win.img.lz.Cart.StaticArea.StaticSize(); addr++ {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", addr/16))
			imgui.SameLine()
			win.headerStartX = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}
		win.drawEditByte(uint16(addr))
		i++
	}
	imgui.PopItemWidth()

	// finished with spacing setting
	imgui.PopStyleVar()
}

func (win *winStatic) drawEditByte(addr uint16) {
	d, _ := win.img.lz.Cart.StaticArea.StaticRead(addr)

	label := fmt.Sprintf("##%d", addr)
	content := fmt.Sprintf("%02x", d)

	if imguiHexInput(label, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() {
				win.img.lz.Cart.StaticArea.StaticWrite(addr, uint8(v))
			})
		}
	}
}
