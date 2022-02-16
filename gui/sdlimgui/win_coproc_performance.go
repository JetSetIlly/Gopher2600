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
	functionTabNew bool
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
		imgui.BeginTabBar("##coprocSourceTabBar")

		options := false

		if imgui.BeginTabItemV("Functions", nil, imgui.TabItemFlagsNone) {
			win.drawFunctions(src)
			imgui.EndTabItem()
		}

		if imgui.BeginTabItemV("Source Line", nil, imgui.TabItemFlagsNone) {
			win.drawSourceLines(src)
			imgui.EndTabItem()
			options = true
		}

		if src.HasFunctionFilter() {
			flgs := imgui.TabItemFlagsNone
			if win.functionTabNew {
				flgs = imgui.TabItemFlagsSetSelected
				win.functionTabNew = false
			}
			open := true
			if imgui.BeginTabItemV(fmt.Sprintf("%c %s", fonts.MagnifyingGlass, src.FunctionFilter), &open, flgs) {
				win.drawFunctionFilter(src)
				imgui.EndTabItem()
			}
			if !open {
				src.DropFunctionFilter()
			}
		}

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

			if options {
				imgui.Checkbox("Show Source in Tooltip", &win.showSrc)
			} else {
				imgui.AlignTextToFramePadding()
				imgui.Text("")
			}
		})
	})
}

func (win *winCoProcPerformance) drawFunctions(src *developer.Source) {
	if src == nil || len(src.SortedFunctions.Functions) == 0 {
		imgui.Text("No performance profile")
		return
	}

	const numColumns = 6

	imgui.Spacing()
	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsSortable
	imgui.BeginTableV("##coprocPerformanceTableFunctions", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsNoSort, width*0.30, 0)
	imgui.TableSetupColumnV("Start Line", imgui.TableColumnFlagsNoSort, width*0.1, 1)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNoSort, width*0.35, 2)
	imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNoSort, 0, 5)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() {
		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 3:
				src.SortedFunctions.SortByFrameCycles(true)
			case 4:
				src.SortedFunctions.SortByAverageCycles(true)
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

		// open source window on click
		if imgui.IsItemClicked() {
			win.functionTabNew = true
			src.SetFunctionFilter(fn.Name)
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
		if ld, ok := fn.Stats.FrameLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if ld, ok := fn.Stats.AverageLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawSourceLines(src *developer.Source) {
	if src == nil || len(src.SortedLines.Lines) == 0 {
		imgui.Text("No performance profile")
		return
	}

	const numColumns = 6

	imgui.Spacing()
	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsSortable
	imgui.BeginTableV("##coprocPerformanceTableSourceLines", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsNoSort, width*0.30, 0)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.1, 1)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNoSort, width*0.35, 2)
	imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNoSort, 0, 5)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() {
		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 3:
				src.SortedLines.SortByFrameCycles(true)
			case 4:
				src.SortedLines.SortByAverageCycles(true)
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
		imgui.SelectableV(ln.File.ShortFilename, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// source on tooltip
		if win.showSrc {
			imguiTooltip(func() {
				imgui.Text(ln.File.ShortFilename)
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
				imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
				imgui.PopStyleColor()
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.Text(strings.TrimSpace(ln.Content))
			}, true)
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
		imgui.Text(fmt.Sprintf("%s", ln.Function.Name))

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if ld, ok := ln.Stats.FrameLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if ld, ok := ln.Stats.AverageLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawFunctionFilter(src *developer.Source) {
	const numColumns = 4

	imgui.Spacing()

	if src == nil || len(src.FunctionFilteredLines.Lines) == 0 {
		imgui.Text("Function contains no executable lines")
		return
	}

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsSortable
	imgui.BeginTableV("##coprocPerformanceTableFunctionFilter", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Source Line", imgui.TableColumnFlagsNoSort, width*0.75, 0)
	imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 1)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 2)
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNoSort, 0, 3)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	sort := imgui.TableGetSortSpecs()
	if sort.SpecsDirty() {
		for _, s := range sort.Specs() {
			switch s.ColumnUserID {
			case 3:
				src.FunctionFilteredLines.SortByFrameCycles(true)
			case 4:
				src.FunctionFilteredLines.SortByAverageCycles(true)
			}
		}
		sort.ClearSpecsDirty()
	}

	for _, ln := range src.FunctionFilteredLines.Lines {
		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV(fmt.Sprintf("%s", strings.TrimSpace(ln.Content)), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// source on tooltip
		if win.showSrc {
			imguiTooltip(func() {
				imgui.Text(ln.File.ShortFilename)
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
				imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
				imgui.PopStyleColor()
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.Text(strings.TrimSpace(ln.Content))
			}, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if ld, ok := ln.Stats.FrameLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if ld, ok := ln.Stats.AverageLoad(src); ok {
			imgui.Text(fmt.Sprintf("%.02f", ld))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
	}

	imgui.EndTable()
}
