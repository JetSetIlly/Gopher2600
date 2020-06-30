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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
)

const winDPCstaticTitle = "DPC Static Areas"

type winDPCstatic struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	xPos float32
}

func newWinDPCstatic(img *SdlImgui) (managedWindow, error) {
	win := &winDPCstatic{img: img}

	return win, nil
}

func (win *winDPCstatic) init() {
	win.widgetDimensions.init()
}

func (win *winDPCstatic) destroy() {
}

func (win *winDPCstatic) id() string {
	return winDPCstaticTitle
}

func (win *winDPCstatic) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	sa, ok := win.img.lz.Cart.Static.(cartridge.DPCstatic)
	if !win.img.lz.Cart.HasStaticBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{469, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{394, 356}, imgui.ConditionFirstUseEver)

	imgui.BeginV(winDPCstaticTitle, &win.open, 0)

	imgui.BeginTabBar("")
	if imgui.BeginTabItemV("Gfx", nil, 0) {
		win.drawGrid(sa.Gfx)
		imgui.EndTabItem()
	}
	imgui.EndTabBar()

	imgui.End()
}

func (win *winDPCstatic) drawGrid(a []byte) {
	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies on win.xPos, which requires
	// one frame before it is accurate.
	headerDim := imgui.Vec2{X: win.xPos, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.twoDigitDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	imgui.PushItemWidth(win.twoDigitDim.X)
	i := uint16(0)
	for addr := 0; addr < len(a); addr++ {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", addr/16))
			imgui.SameLine()
			win.xPos = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}
		win.drawEditByte(uint16(addr), a[i])
		i++
	}
	imgui.PopItemWidth()

	// finished with spacing setting
	imgui.PopStyleVar()
}

func (win *winDPCstatic) drawEditByte(addr uint16, b byte) {
	label := fmt.Sprintf("##%d", addr)
	content := fmt.Sprintf("%02x", b)

	if imguiHexInput(label, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetStaticBus()
				b.PutStatic(addr, uint8(v))
			})
		}
	}
}
