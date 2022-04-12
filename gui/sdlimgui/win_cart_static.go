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

	imgui.SetNextWindowPosV(imgui.Vec2{117, 248}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{473, 552}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{473, 300}, imgui.Vec2{473, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.ID, winCartStaticID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)

	// get comparison data. assuming that there is such a thing and that it's
	// safe to get StaticData from.
	compStatic := win.img.lz.Rewind.Comparison.Mem.Cart.GetStaticBus().GetStatic()

	imgui.BeginTabBar("")
	for _, seg := range win.img.lz.Cart.Static.Segments() {
		// skip any segments that are empty for whatever reason
		if seg.Origin == seg.Memtop {
			continue
		}

		if imgui.BeginTabItemV(seg.Name, nil, 0) {
			imgui.BeginChildV("scrollable", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)

			currData, ok := win.img.lz.Cart.Static.Reference(seg.Name)

			if ok {
				compData, ok := compStatic.Reference(seg.Name)
				if ok {
					// take copy of seg.Name because we'll be accessing it in a
					// PushRawEvent() below
					segname := seg.Name

					drawByteGrid(currData, compData, win.img.cols.ValueDiff, 0,
						func(addr uint16, data uint8) {
							win.img.dbg.PushRawEvent(func() {
								idx := int(addr)
								win.img.vcs.Mem.Cart.GetStaticBus().PutStatic(segname, uint16(idx), data)
							})
						})
				}
			}

			imgui.EndChild()

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()

	imgui.End()
}
