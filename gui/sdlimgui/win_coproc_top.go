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
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcTopID = "Coprocessor Top"
const winCoProcTopMenu = "Top"

type winCoProcTop struct {
	img           *SdlImgui
	open          bool
	showAsm       bool
	showNumbering bool
	optionsHeight float32
}

func newWinCoProcTop(img *SdlImgui) (window, error) {
	win := &winCoProcTop{
		img:           img,
		showNumbering: true,
	}
	return win, nil
}

func (win *winCoProcTop) init() {
}

func (win *winCoProcTop) id() string {
	return winCoProcTopID
}

func (win *winCoProcTop) isOpen() bool {
	return win.open
}

func (win *winCoProcTop) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcTop) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcTopID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	if win.img.dbg.CoProcDev == nil {
		imgui.Text("No source files available")
		return
	}

	const top = 25

	// safely iterate over top execution information
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		imgui.BeginTableV("##coprocTop", 5, imgui.TableFlagsSizingFixedFit|imgui.TableFlagsBorders, imgui.Vec2{}, 0.0)

		// first column is a dummy column so that Selectable (span all columns) works correctly
		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 1, 0)
		imgui.TableSetupColumnV("File", imgui.TableColumnFlagsNone, -1, 1)
		imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNone, 40, 2)
		imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNone, -1, 3)
		imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNone, 40, 4)

		if src == nil {
			imgui.Text("No source files available")
			return
		}

		imgui.TableHeadersRow()

		for i := 0; i < top; i++ {
			imgui.TableNextRow()
			ln := src.SrcLinesAll.Ordered[i]

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
			imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
			imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
			imgui.PopStyleColorV(2)

			// asm on tooltip
			imguiTooltip(func() {
				limit := 0
				for _, asm := range ln.Asm {
					imgui.Text(asm.Instruction)
					limit++
					if limit > 10 {
						imgui.Text("...more")
						break // for loop
					}
				}
			}, true)

			// open source window on click
			if imgui.IsItemClicked() {
				srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
				srcWin.gotoSource(ln)
			}

			imgui.TableNextColumn()
			imgui.Text(ln.File.Filename)

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
			imgui.TableNextColumn()
			if ln.Function == "" {
				imgui.Text("unknown function")
			} else {
				imgui.Text(fmt.Sprintf("%s()", ln.Function))
			}
			imgui.TableNextColumn()
			if ln.CycleCount > 0 {
				imgui.Text(fmt.Sprintf("%0.1f%%", ln.CycleCount/src.TotalCycleCount*100.0))
			}
		}

		imgui.EndTable()
	})
}
