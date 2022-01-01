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
	"github.com/jetsetilly/gopher2600/coprocessor/objdump"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcSourceID = "Coprocessor Source"
const winCoProcSourceMenu = "Source"

type winCoProcSource struct {
	img  *SdlImgui
	open bool
}

func newWinCoProcSource(img *SdlImgui) (window, error) {
	win := &winCoProcSource{
		img: img,
	}
	return win, nil
}

func (win *winCoProcSource) init() {
}

func (win *winCoProcSource) id() string {
	return winCoProcSourceID
}

func (win *winCoProcSource) isOpen() bool {
	return win.open
}

func (win *winCoProcSource) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcSource) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcSourceID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	if win.img.dbg.CoProcDev.Source == nil {
		imgui.Text("No source files available")
		return
	}

	it := win.img.dbg.CoProcDev.Source.NewIteration()

	imgui.BeginTabBar("##coprocSourceTabBar")
	defer imgui.EndTabBar()

	var i objdump.IterationItem

	var done bool
	var next bool

	for !done {
		if next {
			i = <-it.Next
		}
		next = true

		switch i.ID {
		case objdump.SourceFile:
			if i.Detail {
				if imgui.BeginTabItem(i.Content) {
					i = win.drawFile(it)
					imgui.EndTabItem()
					next = false
				}
			}
		case objdump.End:
			done = true
		}
	}
}

func (win *winCoProcSource) drawFile(it objdump.Iteration) objdump.IterationItem {
	imgui.BeginChildV("lastexecution", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)
	imgui.BeginTableV("##coprocSourceTable", 2, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0.0)

	defer imgui.EndChild()
	defer imgui.EndTable()

	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 20, 0)

	iterating := true
	for iterating {
		i := <-it.Next
		switch i.ID {
		case objdump.SourceLine:
			imgui.TableNextRow()
			imgui.TableNextColumn()
			if i.Detail {
				imgui.Text(string(fonts.Chip))
			}
			imgui.TableNextColumn()
			imgui.Text(i.Content)
		case objdump.AsmLine:
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(i.Content)
			imgui.PopStyleColor()
		default:
			return i
		}
	}

	return objdump.IterationItem{}
}
