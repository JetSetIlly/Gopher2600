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
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/yield"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

const winCoProcLocalsID = "Coprocessor Local Variables"
const winCoProcLocalsMenu = "Locals"

type winCoProcLocals struct {
	debuggerWin
	img *SdlImgui

	optionsHeight   float32
	showVisibleOnly bool

	openNodes map[string]bool
}

func newWinCoProcLocals(img *SdlImgui) (window, error) {
	win := &winCoProcLocals{
		img:       img,
		openNodes: make(map[string]bool),
	}
	return win, nil
}

func (win *winCoProcLocals) init() {
}

func (win *winCoProcLocals) id() string {
	return winCoProcLocalsID
}

func (win *winCoProcLocals) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	if !win.img.lz.Cart.HasCoProcBus {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{700, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcLocalsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcLocals) draw() {
	var noSource bool

	// borrow source only so that we can check if whether to draw the window fully
	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
		noSource = src == nil || len(src.Filenames) == 0 || len(src.SortedLocals.Locals) == 0
	})

	// exit draw early leaving a message to indicate that there are no local
	// variables available to display
	if noSource {
		imgui.Text("No local variables in the source")
		return
	}

	const numColumns = 3

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsResizable
	flgs |= imgui.TableFlagsHideable

	imgui.BeginTableV("##localsTable", numColumns, flgs, imgui.Vec2{Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0)

	// setup columns. the labelling column 2 depends on whether the coprocessor
	// development instance has source available to it
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsNoSort, width*0.40, 1)
	imgui.TableSetupColumnV("Type", imgui.TableColumnFlagsNoSort, width*0.30, 2)
	imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNoSort, width*0.30, 3)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	win.img.dbg.CoProcDev.BorrowYieldState(func(yld yield.State) {
		for i, varb := range yld.LocalVariables {
			win.drawVariableLocal(varb, fmt.Sprint(i))
		}
	})

	imgui.EndTable()

	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Checkbox("Only show visible variables", &win.showVisibleOnly)
	})
}

func (win *winCoProcLocals) drawVariableLocal(local *dwarf.SourceVariableLocal, nodeID string) {
	win.drawVariable(local.SourceVariable, 0, nodeID)
}

func (win *winCoProcLocals) drawVariable(varb *dwarf.SourceVariable, indentLevel int, nodeID string) {
	// update variable
	win.img.dbg.PushFunction(varb.Update)

	const IndentDepth = 2

	// name of variable as presented. added bug icon as appropriate
	name := fmt.Sprintf("%s%s", strings.Repeat(" ", IndentDepth*indentLevel), varb.Name)

	imgui.TableNextRow()
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
	imgui.SelectableV(name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	if varb.NumChildren() > 0 {
		// we could show a tooltip for variables with children but this needs
		// work. for instance, how do we illustrate a composite type or an
		// array?
		win.img.imguiTooltip(func() {
			drawVariableTooltipShort(varb, win.img.cols)
		}, true)

		if imgui.IsItemClicked() {
			win.openNodes[nodeID] = !win.openNodes[nodeID]
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		// value column shows tree open/close icon unless there was an error
		// during variable resolution
		imgui.TableNextColumn()
		if varb.ErrorOnResolve != nil {
			imgui.Text(string(fonts.CoProcBug))
		} else {
			if win.openNodes[nodeID] {
				imgui.Text(string(fonts.TreeOpen))
			} else {
				imgui.Text(string(fonts.TreeClosed))
			}

			if win.openNodes[nodeID] {
				for i := 0; i < varb.NumChildren(); i++ {
					win.drawVariable(varb.Child(i), indentLevel+1, fmt.Sprint(nodeID, i))
				}
			}
		}

	} else {
		win.img.imguiTooltip(func() {
			if varb.ErrorOnResolve != nil {
				drawVariableTooltipShort(varb, win.img.cols)
			} else {
				drawVariableTooltip(varb, varb.Value(), win.img.cols)
			}
		}, true)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if varb.ErrorOnResolve != nil {
			imgui.Text(string(fonts.CoProcBug))
		} else {
			imgui.Text(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
		}
	}
}
