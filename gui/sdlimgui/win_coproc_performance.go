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
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcPerformanceID = "Coprocessor Performance"
const winCoProcPerformanceMenu = "Performance"

type winCoProcPerformance struct {
	img           *SdlImgui
	open          bool
	showSrc       bool
	optionsHeight float32

	// function tab is newly opened/changed
	functionTabDirty  bool
	functionTabSelect string

	// scale load statistics in function filters to the function level (as
	// opposed to the program level)
	functionTabScale bool
}

func newWinCoProcPerformance(img *SdlImgui) (window, error) {
	win := &winCoProcPerformance{
		img:     img,
		showSrc: true,
	}
	return win, nil
}

func (win *winCoProcPerformance) init() {
}

func (win *winCoProcPerformance) id() string {
	return winCoProcPerformanceID
}

func (win *winCoProcPerformance) isOpen() bool {
	return win.open
}

func (win *winCoProcPerformance) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcPerformance) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcPerformanceID)
	if !imgui.BeginV(title, &win.open, imgui.WindowFlagsNone) {
		imgui.End()
		return
	}
	defer imgui.End()

	// safely iterate over top execution information
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.Filenames) == 0 {
			imgui.Text("No source files available")
			return
		}

		imgui.BeginChildV("##coprocPerformanceMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)
		imgui.BeginTabBarV("##coprocSourceTabBar", imgui.TabBarFlagsAutoSelectNewTabs)

		if imgui.BeginTabItemV("Functions", nil, imgui.TabItemFlagsNone) {
			win.drawFunctions(src)
			imgui.EndTabItem()
		}

		if imgui.BeginTabItemV("Source Line", nil, imgui.TabItemFlagsNone) {
			win.drawSourceLines(src)
			imgui.EndTabItem()
		}

		functionFilterActive := false

		for _, ff := range src.FunctionFilters {
			flgs := imgui.TabItemFlagsNone
			open := true
			if ff.FunctionName == win.functionTabSelect {
				flgs |= imgui.TabItemFlagsSetSelected
			}
			if imgui.BeginTabItemV(fmt.Sprintf("%c %s", fonts.MagnifyingGlass, ff.FunctionName), &open, flgs) {
				win.drawFunctionFilter(src, ff)
				imgui.EndTabItem()
				functionFilterActive = true
			}
			if !open {
				src.DropFunctionFilter(ff.FunctionName)
			}
		}
		win.functionTabSelect = ""

		imgui.EndTabBar()
		imgui.EndChild()

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			if src.UnsupportedOptimisation != "" {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
				imgui.AlignTextToFramePadding()
				imgui.Text(fmt.Sprintf(" %c", fonts.Warning))
				imgui.PopStyleColor()
				imguiTooltip(func() {
					imgui.Text(src.UnsupportedOptimisation)
					imgui.Text("performance analysis may be misleading")
				}, true)
				imgui.SameLineV(0, 20)
			}

			imgui.Checkbox("Show Source in Tooltip", &win.showSrc)
			if functionFilterActive {
				imgui.SameLineV(0, 20)
				imgui.Checkbox("Scale Statistics", &win.functionTabScale)
			}
		})
	})
}

func (win *winCoProcPerformance) drawFunctions(src *developer.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedFunctions.Functions) == 0 {
		imgui.Text("No performance profile")
		return
	}

	const numColumns = 6

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable

	imgui.BeginTableV("##coprocPerformanceTableFunctions", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsPreferSortDescending, width*0.275, 0)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.05, 1)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsPreferSortDescending, width*0.325, 2)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.1, 5)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() {
		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 0:
				src.SortedFunctions.SortByFile(s.SortDirection == imgui.SortDirectionAscending)
			case 2:
				src.SortedFunctions.SortByFunction(s.SortDirection == imgui.SortDirectionAscending)
			case 3:
				src.SortedFunctions.SortByFrameCycles(true)
			case 4:
				src.SortedFunctions.SortByAverageCycles(true)
			case 5:
				src.SortedFunctions.SortByMaxCycles(true)
			}
		}
		sort.ClearSpecsDirty()
	}

	for _, fn := range src.SortedFunctions.Functions {
		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		if fn.DeclLine != nil {
			imgui.SelectableV(fn.DeclLine.File.ShortFilename, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		} else {
			imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		}
		imgui.PopStyleColorV(2)

		// source on tooltip
		if win.showSrc {
			win.sourceLineTooltip(fn.DeclLine, false)
		}

		// open/select function filter on click
		if imgui.IsItemClicked() {
			win.functionTabDirty = true
			src.AddFunctionFilter(fn.Name)
			win.functionTabSelect = fn.Name
		}

		imgui.TableNextColumn()
		if fn.DeclLine != nil {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
			imgui.Text(fmt.Sprintf("%d", fn.DeclLine.LineNumber))
			imgui.PopStyleColor()
		}

		imgui.TableNextColumn()
		imgui.Text(fmt.Sprintf("%s", fn.Name))

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if fn.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", fn.Stats.OverSource.Frame))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if fn.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", fn.Stats.OverSource.Average))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if fn.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", fn.Stats.OverSource.Max))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawSourceLines(src *developer.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedLines.Lines) == 0 {
		imgui.Text("No performance profile")
		return
	}

	const numColumns = 6

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable

	imgui.BeginTableV("##coprocPerformanceTableSourceLines", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsPreferSortDescending, width*0.20, 0)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.05, 1)
	imgui.TableSetupColumnV("Content", imgui.TableColumnFlagsNoSort, width*0.35, 2)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.07, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.07, 4)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.07, 5)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() {
		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 0:
				src.SortedLines.SortByFunction(s.SortDirection == imgui.SortDirectionAscending)
			case 3:
				src.SortedLines.SortByFrameLoadOverSource(true)
			case 4:
				src.SortedLines.SortByAverageLoadOverSource(true)
			case 5:
				src.SortedLines.SortByMaxLoadOverSource(true)
			}
		}
		sort.ClearSpecsDirty()
	}

	for _, ln := range src.SortedLines.Lines {
		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV(ln.Function.Name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// source on tooltip
		if win.showSrc {
			win.sourceLineTooltip(ln, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.Text(strings.TrimSpace(ln.Content))

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if ln.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Frame))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if ln.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Average))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if ln.Stats.IsValid() {
			imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Max))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawFunctionFilter(src *developer.Source, functionFilter *developer.FunctionFilter) {
	imgui.Spacing()

	if len(functionFilter.Lines.Lines) == 0 {
		imgui.Text(fmt.Sprintf("%s contains no executable lines", functionFilter.FunctionName))
		return
	}

	const numColumns = 5

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable

	imgui.BeginTableV("##coprocPerformanceTableFunctionFilter", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsPreferSortDescending, width*0.05, 0)
	imgui.TableSetupColumnV("Source", imgui.TableColumnFlagsNoSort, width*0.60, 1)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 2)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 3)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() || win.functionTabDirty {
		win.functionTabDirty = false

		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 0:
				functionFilter.Lines.SortByLineNumber(s.SortDirection == imgui.SortDirectionAscending)
			case 2:
				if win.functionTabScale {
					functionFilter.Lines.SortByFrameLoadOverFunction(true)
				} else {
					functionFilter.Lines.SortByFrameLoadOverSource(true)
				}
			case 3:
				if win.functionTabScale {
					functionFilter.Lines.SortByAverageLoadOverFunction(true)
				} else {
					functionFilter.Lines.SortByAverageLoadOverSource(true)
				}
			case 4:
				if win.functionTabScale {
					functionFilter.Lines.SortByMaxLoadOverFunction(true)
				} else {
					functionFilter.Lines.SortByMaxLoadOverSource(true)
				}
			}
		}
		sort.ClearSpecsDirty()
	}

	for _, ln := range functionFilter.Lines.Lines {
		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.SelectableV(fmt.Sprintf("%d", ln.LineNumber), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(3)

		// source on tooltip
		if win.showSrc {
			win.sourceLineTooltip(ln, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		imgui.Text(strings.TrimSpace(ln.Content))

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if ln.Stats.IsValid() {
			if win.functionTabScale {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverFunction.Frame))
			} else {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Frame))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if ln.Stats.IsValid() {
			if win.functionTabScale {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverFunction.Average))
			} else {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Average))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if ln.Stats.IsValid() {
			if win.functionTabScale {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverFunction.Max))
			} else {
				imgui.Text(fmt.Sprintf("%.02f", ln.Stats.OverSource.Max))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) sourceLineTooltip(ln *developer.SourceLine, withAsm bool) {
	imguiTooltip(func() {
		imgui.Text(ln.File.ShortFilename)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
		imgui.PopStyleColor()
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(strings.TrimSpace(ln.Content))

		if withAsm && len(ln.Disassembly) > 0 {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.BeginTable("##disasmTable", 3)
			for _, asm := range ln.Disassembly {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%08x", asm.Addr))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%04x", asm.Opcode))
				imgui.TableNextColumn()
				imgui.Text(asm.Instruction)
			}
			imgui.EndTable()
		}
	}, true)
}
