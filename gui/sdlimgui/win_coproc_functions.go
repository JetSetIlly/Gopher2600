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
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
)

const winCoProcFunctionsID = "Coprocessor Functions"
const winCoProcFunctionsMenu = "Functions"

type winCoProcFunctions struct {
	debuggerWin

	img *SdlImgui

	showSrcInTooltip bool
	optionsHeight    float32
}

func newWinCoProcFunctions(img *SdlImgui) (window, error) {
	win := &winCoProcFunctions{
		img: img,
	}
	return win, nil
}

func (win *winCoProcFunctions) init() {
}

func (win *winCoProcFunctions) id() string {
	return winCoProcFunctionsID
}

func (win *winCoProcFunctions) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{775, 102}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{400, 655}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{300, 400}, imgui.Vec2{600, 1000})

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcFunctionsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcFunctions) draw() {
	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.FunctionNames) == 0 {
			imgui.Text("No functions defined")
			return
		}

		const numColumns = 2

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsResizable
		flgs |= imgui.TableFlagsHideable
		if !imgui.BeginTableV("##coprocFunctionsTable", numColumns, flgs, imgui.Vec2{}, 0.0) {
			return
		}

		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsPreferSortDescending|imgui.TableColumnFlagsNoHide, width*0.5, 0)
		imgui.TableSetupColumnV("File", imgui.TableColumnFlagsNoSort, width*0.5, 1)

		imgui.TableSetupScrollFreeze(0, 1)
		imgui.TableHeadersRow()

		for _, n := range src.FunctionNames {
			fn := src.Functions[n]
			if !fn.IsStub() {
				imgui.TableNextRow()
				imgui.TableNextColumn()

				if imgui.SelectableV(n, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{}) {
					srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
					srcWin.gotoSourceLine(fn.DeclLine)
				}

				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceFilename)
				imgui.Text(fn.DeclLine.File.ShortFilename)
				imgui.PopStyleColor()
			}
		}

		imgui.EndTable()
	})
}
