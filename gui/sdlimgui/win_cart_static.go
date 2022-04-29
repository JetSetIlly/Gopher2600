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
)

const winCartStaticID = "Static Areas"

type winCartStatic struct {
	debuggerWin

	img *SdlImgui
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

func (win *winCartStatic) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	if !win.img.lz.Cart.HasStaticBus {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{117, 248}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{468, 552}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{468, 271}, imgui.Vec2{529, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.ID, winCartStaticID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	imgui.End()
}

func (win *winCartStatic) draw() {
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
			imgui.Spacing()
			imgui.Text("Origin:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
			imgui.Text(fmt.Sprintf("%08x", seg.Origin))
			imgui.PopStyleColor()
			imgui.SameLineV(0, 20)
			imgui.Text("Memtop:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
			imgui.Text(fmt.Sprintf("%08x", seg.Memtop))
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Spacing()

			imgui.BeginChildV("cartstatic", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)

			currData, ok := win.img.lz.Cart.Static.Reference(seg.Name)

			if ok {
				compData, ok := compStatic.Reference(seg.Name)
				if ok {
					// take copy of seg.Name because we'll be accessing it in a
					// PushRawEvent() below
					segname := seg.Name

					// number of colors to pop in afer()
					popColor := 0

					before := func(offset uint32) {
						// difference colour
						a := currData[offset]
						b := compData[offset]
						if a != b {
							imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
							popColor++
						}
					}

					after := func(offset uint32) {
						imgui.PopStyleColorV(popColor)
						popColor = 0

						imguiTooltip(func() {
							imgui.Text("Address:")
							imgui.SameLine()
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
							imgui.Text(fmt.Sprintf("%08x", seg.Origin+offset))
							imgui.PopStyleColor()

							a := currData[offset]
							b := compData[offset]
							if a != b {
								imgui.Spacing()
								imguiColorLabel(fmt.Sprintf("%02x %c %02x", b, fonts.ByteChange, a), win.img.cols.ValueDiff)
							}
						}, true)
					}

					commit := func(addr uint32, data uint8) {
						win.img.dbg.PushRawEvent(func() {
							idx := int(addr)
							win.img.vcs.Mem.Cart.GetStaticBus().PutStatic(segname, uint16(idx), data)
						})
					}

					drawByteGrid("cartStaticByteGrid", currData, seg.Origin, before, after, commit)
				}
			}

			imgui.EndChild()

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()
}
