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
)

const winCartStaticID = "Static Areas"

type winCartStatic struct {
	img  *SdlImgui
	open bool
}

func newWinCartStatic(img *SdlImgui) (window, error) {
	win := &winCartStatic{img: img}

	return win, nil
}

func (win *winCartStatic) init() {
}

func (win *winCartStatic) id() string {
	return winCartStaticID
}

func (win *winCartStatic) isOpen() bool {
	return win.open
}

func (win *winCartStatic) setOpen(open bool) {
	win.open = open
}

func (win *winCartStatic) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasStaticBus {
		return
	}

	// get comparison data. assuming that there is such a thing and that it's
	// safe to get StaticData from.
	comp := win.img.lz.Rewind.Comparison.Mem.Cart.GetStaticBus().GetStatic()

	imgui.SetNextWindowPosV(imgui.Vec2{117, 248}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{394, 356}, imgui.ConditionFirstUseEver)

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.ID, winCartStaticID)
	imgui.BeginV(title, &win.open, 0)

	imgui.BeginTabBar("")
	for s := range win.img.lz.Cart.Static {
		a := win.img.lz.Cart.Static[s]
		b := comp[s]
		if imgui.BeginTabItemV(a.Segment, nil, 0) {
			imgui.BeginChildV("scrollable", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)

			drawByteGrid(a.Data, b.Data, win.img.cols.ValueDiff, 0,
				func(addr uint16, data uint8) {
					win.img.lz.Dbg.PushRawEvent(func() {
						idx := int(addr)
						win.img.lz.Dbg.VCS.Mem.Cart.GetStaticBus().PutStatic(a.Segment, uint16(idx), data)
					})
				})

			imgui.EndChild()

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()

	imgui.End()
}
