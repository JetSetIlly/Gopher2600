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

	optionsHeight     float32
	showLocatableOnly bool
	filter            filter

	openNodes map[string]bool
}

func newWinCoProcLocals(img *SdlImgui) (window, error) {
	win := &winCoProcLocals{
		img:       img,
		filter:    newFilter(img, filterFlagsVariableNamesC),
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

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcLocalsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcLocals) draw() {
	// take copy of yield state before borrowing the source
	var yieldState yield.State
	win.img.dbg.CoProcDev.BorrowYieldState(func(yld yield.State) {
		yieldState = yld
	})

	// borrow source only so that we can check if whether to draw the window fully
	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
		if src == nil || len(src.Filenames) == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text("No source files available")
			return
		}

		// the local variables list we use depends on whether the emulation is
		// running and whether a strobe is active. we use the LocalVariableView()
		// function for this
		viewedAddr, viewedLocals := yieldState.LocalVariableView(win.img.dbg.State())

		// exit draw early leaving a message to indicate that there are no local
		// variables available to display
		if len(viewedLocals) == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text("No local variables currently visible")
			return
		}

		// strobe information at the top of the window
		ln := src.SourceLineByAddr(viewedAddr)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("%s line %d", ln.File.ShortFilename, ln.LineNumber))
		if imgui.IsItemClicked() && imgui.IsMouseDoubleClicked(0) {
			srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}
		imgui.SameLineV(0, 15)
		if yieldState.Strobe {
			if imgui.Button("Remove Strobe") {
				win.img.dbg.CoProcDev.EnableStrobe(false, viewedAddr)
			}

			// the addr and the line from that address are for the current yield
			// point. for the tooltip however, we want to use the strobe addr
			//
			// if strobe line is different to yield line then show the tooltip
			sln := src.SourceLineByAddr(yieldState.StrobeAddr)
			if sln != ln {
				imguiTooltipSimple(fmt.Sprintf("Strobe is currently set to:\n%s line %d", sln.File.ShortFilename, sln.LineNumber), true)
			}
		} else {
			if imgui.Button("Set Strobe") {
				win.img.dbg.CoProcDev.EnableStrobe(true, viewedAddr)
			}
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		const numColumns = 3

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsResizable
		flgs |= imgui.TableFlagsHideable

		if !imgui.BeginTableV("##localsTable", numColumns, flgs, imgui.Vec2{Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0) {
			return
		}

		// setup columns. the labelling column 2 depends on whether the coprocessor
		// development instance has source available to it
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsNoSort, width*0.40, 1)
		imgui.TableSetupColumnV("Type", imgui.TableColumnFlagsNoSort, width*0.30, 2)
		imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNoSort, width*0.30, 3)

		imgui.TableSetupScrollFreeze(0, 1)
		imgui.TableHeadersRow()

		for i, varb := range viewedLocals {
			if !win.filter.isFiltered(varb.Name) {
				win.drawVariableLocal(varb, fmt.Sprint(i))
			}
		}

		imgui.EndTable()

		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.Checkbox("Hide unlocatable variables", &win.showLocatableOnly)
			win.img.imguiTooltipSimple(`A unlocatable variable is a variable has been
removed by the compiler's optimisation process`)
			win.filter.draw("##localsFilter")
		})
	})
}

func (win *winCoProcLocals) drawVariableLocal(local *dwarf.SourceVariableLocal, nodeID string) {
	win.drawVariable(local.SourceVariable, 0, nodeID)
}

func (win *winCoProcLocals) drawVariable(varb *dwarf.SourceVariable, indentLevel int, nodeID string) {
	// update variable
	win.img.dbg.PushFunction(func() {
		win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
			varb.Update()
		})
	})

	const IndentDepth = 2

	// name of variable as presented. added bug icon as appropriate
	name := fmt.Sprintf("%s%s", strings.Repeat(" ", IndentDepth*indentLevel), varb.Name)
	if !varb.IsValid() {
		if win.showLocatableOnly {
			return
		}
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesNotVisible)
		defer imgui.PopStyleColor()
	}

	imgui.TableNextRow()
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
	imgui.SelectableV(name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	if !varb.IsValid() {
		imgui.TableNextColumn()
		imgui.Text(varb.Type.Name)
		imgui.TableNextColumn()
		imgui.Text("not locatable")
		return
	}

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
		if varb.Error != nil {
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
			if varb.Error != nil {
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
		if varb.Error != nil {
			imgui.Text(string(fonts.CoProcBug))
		} else if varb.Type.Conversion != nil {
			imgui.Text(fmt.Sprintf(varb.Type.Conversion(varb.Value())))
		} else {
			imgui.Text(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
		}
	}
}
