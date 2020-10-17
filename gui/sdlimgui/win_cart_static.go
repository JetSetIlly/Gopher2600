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
	"github.com/jetsetilly/gopher2600/logger"
)

const winCartStaticTitle = "Static Areas"

type winCartStatic struct {
	windowManagement

	img *SdlImgui

	// the X position of the grid header. based on the width of the column
	// headers (we know this value after the first pass)
	xPos float32
}

func newWinCartStatic(img *SdlImgui) (managedWindow, error) {
	win := &winCartStatic{img: img}

	return win, nil
}

func (win *winCartStatic) init() {
}

func (win *winCartStatic) destroy() {
}

func (win *winCartStatic) id() string {
	return winCartStaticTitle
}

func (win *winCartStatic) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasStaticBus {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{469, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{394, 356}, imgui.ConditionFirstUseEver)

	imgui.BeginV(winCartStaticTitle, &win.open, 0)

	imgui.BeginTabBar("")
	for _, s := range win.img.lz.Cart.Static {
		if imgui.BeginTabItemV(s.Label, nil, 0) {
			win.drawGrid(s.Label, s.Data)
			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()

	imgui.End()
}

func (win *winCartStatic) drawGrid(tag string, a []byte) {
	imgui.BeginChild(tag)

	// no spacing between any of the drawEditByte() objects
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies on win.xPos, which requires
	// one frame before it is accurate.
	headerDim := imgui.Vec2{X: win.xPos, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += imguiTextWidth(2)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	imgui.PushItemWidth(imguiTextWidth(2))
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
		win.drawEditByte(tag, uint16(addr), a[i])
		i++
	}
	imgui.PopItemWidth()

	// finished with spacing setting
	imgui.PopStyleVar()

	imgui.EndChild()
}

func (win *winCartStatic) drawEditByte(tag string, addr uint16, b byte) {
	l := fmt.Sprintf("##%d", addr)
	content := fmt.Sprintf("%02x", b)

	if imguiHexInput(l, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetStaticBus()
				err := b.PutStatic(tag, addr, uint8(v))
				if err != nil {
					logger.Log("sdlimgui", err.Error())
				}
			})
		}
	}
}
