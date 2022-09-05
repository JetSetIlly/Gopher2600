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
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcPerformanceID = "Coprocessor Performance"
const winCoProcPerformanceMenu = "Performance"

type winCoProcPerformance struct {
	debuggerWin

	img *SdlImgui

	// source shown in tooltip
	showSrcAsmInTooltip bool

	// which kernel to focus on
	kernelFocus         developer.KernelVCS
	kernelFocusComboDim imgui.Vec2

	// whether to present performance figures as raw counts or as percentages
	percentileFigures bool

	// whether the sort criteria as specified in the window has changed (ie.
	// kernelFocus of percentileFigures widgets has been altered)
	windowSortSpecDirty bool

	// the currently selected tab. used to dirty the sort flag when a new tab
	// is selected
	tabSelected string

	// whether to included unexecuted entries (functions or lines) in the list
	hideUnusedEntries bool

	// scale load statistics in function filters to the function level (as
	// opposed to the program level)
	functionTabScale bool

	// height of all options
	optionsHeight float32

	// function tab is newly opened/changed. this means that the stats should
	// be resorted
	functionTabSelect string
}

func newWinCoProcPerformance(img *SdlImgui) (window, error) {
	win := &winCoProcPerformance{
		img:                 img,
		showSrcAsmInTooltip: true,
		kernelFocus:         developer.KernelAny,
		percentileFigures:   true,
	}
	return win, nil
}

func (win *winCoProcPerformance) init() {
	win.kernelFocusComboDim = imguiGetFrameDim("", developer.AvailableInKernelOptions...)
}

func (win *winCoProcPerformance) id() string {
	return winCoProcPerformanceID
}

func (win *winCoProcPerformance) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{858, 319}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{641, 517}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcPerformanceID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winCoProcPerformance) draw() {
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

		// ExecutionProfileChanged is used to decide whether to sort
		// statistics. make sure we set it to false by the end of the draw()
		// function
		if src.ExecutionProfileChanged {
			defer func() {
				src.ExecutionProfileChanged = false
			}()
		}

		imgui.BeginChildV("##coprocPerformanceMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)
		imgui.BeginTabBarV("##coprocSourceTabBar", imgui.TabBarFlagsAutoSelectNewTabs)

		tabName := "Functions"
		if imgui.BeginTabItemV(tabName, nil, imgui.TabItemFlagsNone) {
			if win.tabSelected != tabName {
				win.tabSelected = tabName
				win.windowSortSpecDirty = true
			}
			win.drawFunctions(src)
			imgui.EndTabItem()
		}

		tabName = "Source Lines"
		if imgui.BeginTabItemV(tabName, nil, imgui.TabItemFlagsNone) {
			if win.tabSelected != tabName {
				win.tabSelected = tabName
				win.windowSortSpecDirty = true
			}
			win.drawSourceLines(src)
			imgui.EndTabItem()
		}

		for _, ff := range src.FunctionFilters {
			flgs := imgui.TabItemFlagsNone
			open := true
			if ff.FunctionName == win.functionTabSelect {
				flgs |= imgui.TabItemFlagsSetSelected
			}

			tabName = fmt.Sprintf("%c %s", fonts.MagnifyingGlass, ff.FunctionName)
			if imgui.BeginTabItemV(tabName, &open, flgs) {
				if win.tabSelected != tabName {
					win.tabSelected = tabName
					win.windowSortSpecDirty = true
				}
				win.drawFunctionFilter(src, ff)
				imgui.EndTabItem()
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
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			imguiLabel("Kernel Focus")
			imgui.PushItemWidth(win.kernelFocusComboDim.X + imgui.FrameHeight())
			if imgui.BeginCombo("##kernelFocus", win.kernelFocus.String()) {
				if imgui.Selectable(developer.KernelAny.String()) {
					win.kernelFocus = developer.KernelAny
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(developer.KernelVBLANK.String()) {
					win.kernelFocus = developer.KernelVBLANK
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(developer.KernelScreen.String()) {
					win.kernelFocus = developer.KernelScreen
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(developer.KernelOverscan.String()) {
					win.kernelFocus = developer.KernelOverscan
					win.windowSortSpecDirty = true
				}
				imgui.EndCombo()
			}

			imgui.SameLineV(0, 15)
			if imgui.Checkbox("Percentile Figures", &win.percentileFigures) {
				win.windowSortSpecDirty = true
			}

			imgui.SameLineV(0, 15)
			imgui.Checkbox("Hide Unexecuted Items", &win.hideUnusedEntries)

			// scale statistics to function is in drawFunctionFilter()
			imgui.Spacing()
			imgui.Checkbox("Show Source in Tooltip", &win.showSrcAsmInTooltip)

			// reset statistics
			imgui.SameLineV(0, 15)
			if imgui.Button(fmt.Sprintf("%c Reset Statistics", fonts.Trash)) {
				src.ResetStatistics()
			}

			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			win.drawFrameStats()
		})
	})
}

func (win *winCoProcPerformance) drawFrameStats() {
	accumulate := func(s mapper.CoProcState) int {
		switch s {
		case mapper.CoProcIdle:
		case mapper.CoProcNOPFeed:
			return 1
		case mapper.CoProcStrongARMFeed:
		case mapper.CoProcParallel:
			return 1
		}
		return 0
	}

	win.img.screen.crit.section.Lock()
	defer win.img.screen.crit.section.Unlock()

	// decide which kernel we're using
	var kernel string
	var kernelClocks float32

	switch win.kernelFocus {
	case developer.KernelAny:
		kernelClocks = float32(win.img.screen.crit.frameInfo.TotalClocks())
		kernel = "TV Frame"
	case developer.KernelScreen:
		kernelClocks = float32(win.img.screen.crit.frameInfo.ScreenClocks())
		kernel = "Screen"
	case developer.KernelVBLANK:
		kernelClocks = float32(win.img.screen.crit.frameInfo.VBLANKClocks())
		kernel = "VBLANK"
	case developer.KernelOverscan:
		kernelClocks = float32(win.img.screen.crit.frameInfo.OverscanClocks())
		kernel = "Overscan"
	}

	// frame statistics are taken from reflection information
	var clockCount float32

	for i, r := range win.img.screen.crit.reflection {
		sl := i / specification.ClksScanline

		switch win.kernelFocus {
		case developer.KernelAny:
			clockCount += float32(accumulate(r.CoProcState))
		case developer.KernelScreen:
			if sl >= win.img.screen.crit.frameInfo.VisibleTop && sl <= win.img.screen.crit.frameInfo.VisibleBottom {
				clockCount += float32(accumulate(r.CoProcState))
			}
		case developer.KernelVBLANK:
			if sl < win.img.screen.crit.frameInfo.VisibleTop {
				clockCount += float32(accumulate(r.CoProcState))
			}
		case developer.KernelOverscan:
			if sl > win.img.screen.crit.frameInfo.VisibleBottom {
				clockCount += float32(accumulate(r.CoProcState))
			}
		}
	}

	if clockCount > 0 {
		imgui.Text(fmt.Sprintf("%s activity in most recent %s:", win.img.lz.Cart.CoProcID, kernel))
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		imgui.Text(fmt.Sprintf("%.02f%%", clockCount/kernelClocks*100))
		imgui.PopStyleColor()
	} else if win.kernelFocus == developer.KernelAny {
		imgui.Text(fmt.Sprintf("No %s activity in the most recent frame", win.img.lz.Cart.CoProcID))
	} else {
		imgui.Text(fmt.Sprintf("No %s activity in the %s kernel", win.img.lz.Cart.CoProcID, kernel))
	}
}

func (win *winCoProcPerformance) drawFunctions(src *developer.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedFunctions.Functions) == 0 {
		imgui.Text("No performance profile")
		return
	}

	if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
		if !src.StatsVBLANK.IsValid() {
			imgui.Text("No functions have been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !src.StatsScreen.IsValid() {
			imgui.Text("No functions have been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !src.StatsOverscan.IsValid() {
			imgui.Text("No functions have been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.IsValid() {
			imgui.Text("No functions have been executed yet")
			return
		}
	}

	const numColumns = 7

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable
	flgs |= imgui.TableFlagsHideable

	imgui.BeginTableV("##coprocPerformanceTableFunctions", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsPreferSortDescending, width*0.275, 0)
	imgui.TableSetupColumnV("Lines", imgui.TableColumnFlagsNoSort, width*0.05, 1)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsPreferSortDescending, width*0.320, 2)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.1, 5)
	imgui.TableSetupColumnV(string(fonts.CoProcKernel), imgui.TableColumnFlagsNoSort, width*0.05, 6)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	win.sort(src, func(sort imgui.TableSortSpecs) {
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
	})

	for _, fn := range src.SortedFunctions.Functions {
		if fn.Kernel&win.kernelFocus != win.kernelFocus {
			continue
		}

		if win.hideUnusedEntries && !fn.Stats.IsValid() {
			continue
		}

		// is the function is a stub or not. some facilities are not available
		// without source
		isStub := fn.IsStub()

		// select which stats to focus on
		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = fn.StatsVBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = fn.StatsScreen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = fn.StatsOverscan
		} else {
			stats = fn.Stats
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		if isStub {
			imgui.SelectableV("-", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		} else {
			imgui.SelectableV(fn.DeclLine.File.ShortFilename, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		}
		imgui.PopStyleColorV(2)

		// tooltip
		if isStub {
			imguiTooltipSimple("Function has no underlying source code")
		} else {
			win.tooltip(fn.Stats.OverSource, fn.DeclLine, false)
		}

		// open/select function filter on click
		if imgui.IsItemClicked() && !isStub {
			win.windowSortSpecDirty = true
			src.AddFunctionFilter(fn.Name)
			win.functionTabSelect = fn.Name
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		if isStub {
			imgui.Text("-")
		} else {
			imgui.Text(fmt.Sprintf("%d", fn.DeclLine.LineNumber))
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.Text(fmt.Sprintf("%s", fn.Name))

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if stats.OverSource.FrameValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Frame))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.FrameCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if stats.OverSource.AverageValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Average))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.AverageCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if stats.OverSource.MaxValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Max))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.MaxCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if fn.Kernel == developer.KernelUnstable {
			imgui.Text("R")
		} else {
			if fn.Kernel&developer.KernelVBLANK == developer.KernelVBLANK {
				imgui.Text("V")
				imgui.SameLineV(0, 1)
			}
			if fn.Kernel&developer.KernelScreen == developer.KernelScreen {
				imgui.Text("S")
				imgui.SameLineV(0, 1)
			}
			if fn.Kernel&developer.KernelOverscan == developer.KernelOverscan {
				imgui.Text("O")
				imgui.SameLineV(0, 1)
			}
		}
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawSourceLines(src *developer.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedLines.Lines) == 0 {
		imgui.Text("No performance profile")
		return
	}

	if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
		if !src.StatsVBLANK.IsValid() {
			imgui.Text("No lines have been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !src.StatsScreen.IsValid() {
			imgui.Text("No lines have been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !src.StatsOverscan.IsValid() {
			imgui.Text("No lines have been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.IsValid() {
			imgui.Text("No lines have been executed yet")
			return
		}
	}

	const numColumns = 7

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable
	flgs |= imgui.TableFlagsHideable

	imgui.BeginTableV("##coprocPerformanceTableSourceLines", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsPreferSortDescending, width*0.20, 0)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.05, 1)
	imgui.TableSetupColumnV("Content", imgui.TableColumnFlagsNoSort, width*0.30, 2)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.07, 3)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.07, 4)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.07, 5)
	imgui.TableSetupColumnV(string(fonts.CoProcKernel), imgui.TableColumnFlagsNoSort, width*0.05, 6)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	win.sort(src, func(sort imgui.TableSortSpecs) {
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
	})

	for _, ln := range src.SortedLines.Lines {
		if ln.Kernel&win.kernelFocus != win.kernelFocus {
			continue
		}

		if win.hideUnusedEntries && !ln.Stats.IsValid() {
			continue
		}

		// is the line is a stub or not. some facilities are not available
		// without source
		isStub := ln.IsStub()

		// select which stats to focus on
		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = ln.StatsVBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = ln.StatsScreen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = ln.StatsOverscan
		} else {
			stats = ln.Stats
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV(ln.Function.Name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// tooltip
		if isStub {
			imguiTooltipSimple(fmt.Sprintf("This entry represent all lines of code in %s", ln.Function.Name))
		} else {
			win.tooltip(ln.Stats.OverSource, ln, true)
		}

		// open source window on click
		if !isStub && imgui.IsItemClicked() {
			srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		if isStub {
			imgui.Text("-")
		} else {
			imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		displaySourceFragments(ln, win.img.cols, true)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if stats.OverSource.FrameValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Frame))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.FrameCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if stats.OverSource.AverageValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Average))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.AverageCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if stats.OverSource.MaxValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Max))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.MaxCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if ln.Kernel == developer.KernelUnstable {
			imgui.Text("R")
		} else {
			if ln.Kernel&developer.KernelVBLANK == developer.KernelVBLANK {
				imgui.Text("V")
				imgui.SameLineV(0, 1)
			}
			if ln.Kernel&developer.KernelScreen == developer.KernelScreen {
				imgui.Text("S")
				imgui.SameLineV(0, 1)
			}
			if ln.Kernel&developer.KernelOverscan == developer.KernelOverscan {
				imgui.Text("O")
				imgui.SameLineV(0, 1)
			}
		}
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) drawFunctionFilter(src *developer.Source, functionFilter *developer.FunctionFilter) {
	imgui.Spacing()

	if len(functionFilter.Lines.Lines) == 0 {
		imgui.Text(fmt.Sprintf("%s contains no executable lines", functionFilter.FunctionName))
		return
	}

	// validity check is done on function filter stats - drawFunctions() and
	// drawSourceLines() equivalents of this block perform the validity check
	// on the source level statistics
	if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
		if !functionFilter.Function.StatsVBLANK.IsValid() {
			imgui.Text("This function has not been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !functionFilter.Function.StatsScreen.IsValid() {
			imgui.Text("This function has not been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !functionFilter.Function.StatsOverscan.IsValid() {
			imgui.Text("This function has not been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.IsValid() {
			imgui.Text("Ths function has not been executed yet")
			return
		}
	}

	// function summary in relation to the program
	imgui.AlignTextToFramePadding()
	switch win.kernelFocus {
	case developer.KernelVBLANK:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the VBLANK last frame", functionFilter.Function.StatsVBLANK.OverSource.Frame))
	case developer.KernelScreen:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Visible Screen last frame", functionFilter.Function.StatsScreen.OverSource.Frame))
	case developer.KernelOverscan:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Overscan last frame", functionFilter.Function.StatsOverscan.OverSource.Frame))
	default:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time last frame", functionFilter.Function.Stats.OverSource.Frame))
	}

	imgui.SameLineV(0, 15)
	if imgui.Checkbox("Scale Statistics", &win.functionTabScale) {
		win.windowSortSpecDirty = true
	}
	imguiTooltipSimple(`When selected the % values in the table are
scaled so they are relative to the function rather
thean to the program as a whole.`)

	imgui.Spacing()

	// table of function lines
	const numColumns = 6

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
	flgs |= imgui.TableFlagsNoHostExtendX
	flgs |= imgui.TableFlagsResizable
	flgs |= imgui.TableFlagsHideable

	imgui.BeginTableV("##coprocPerformanceTableFunctionFilter", numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsPreferSortDescending, width*0.05, 0)
	imgui.TableSetupColumnV("Source", imgui.TableColumnFlagsNoSort, width*0.55, 1)
	imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 2)
	imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 3)
	imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
	imgui.TableSetupColumnV(string(fonts.CoProcKernel), imgui.TableColumnFlagsNoSort, width*0.05, 6)

	imgui.TableSetupScrollFreeze(0, 1)
	imgui.TableHeadersRow()

	win.sort(src, func(sort imgui.TableSortSpecs) {
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
	})

	for _, ln := range functionFilter.Lines.Lines {
		if ln.Kernel&win.kernelFocus != win.kernelFocus {
			continue
		}

		if win.hideUnusedEntries && !ln.Stats.IsValid() {
			continue
		}

		if ln.IsStub() {
			continue
		}

		// select which stats to focus on
		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = ln.StatsVBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = ln.StatsScreen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = ln.StatsOverscan
		} else {
			stats = ln.Stats
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.SelectableV(fmt.Sprintf("%d", ln.LineNumber), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(3)

		// source on tooltip
		if win.functionTabScale {
			win.tooltip(ln.Stats.OverFunction, ln, true)
		} else {
			win.tooltip(ln.Stats.OverSource, ln, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		displaySourceFragments(ln, win.img.cols, true)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		if win.functionTabScale {
			if stats.OverFunction.FrameValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverFunction.Frame))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverFunction.FrameCount))
				}
			} else {
				imgui.Text("-")
			}
		} else {
			if stats.OverSource.FrameValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Frame))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.FrameCount))
				}
			} else {
				imgui.Text("-")
			}
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if win.functionTabScale {
			if stats.OverFunction.AverageValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverFunction.Average))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverFunction.AverageCount))
				}
			} else {
				imgui.Text("-")
			}
		} else {
			if stats.OverSource.AverageValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Average))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.AverageCount))
				}
			} else {
				imgui.Text("-")
			}
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if win.functionTabScale {
			if stats.OverFunction.MaxValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverFunction.Max))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverFunction.MaxCount))
				}
			} else {
				imgui.Text("-")
			}
		} else {
			if stats.OverSource.MaxValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverSource.Max))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverSource.MaxCount))
				}
			} else {
				imgui.Text("-")
			}
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if ln.Kernel == developer.KernelUnstable {
			imgui.Text("R")
		} else {
			if ln.Kernel&developer.KernelVBLANK == developer.KernelVBLANK {
				imgui.Text("V")
				imgui.SameLineV(0, 1)
			}
			if ln.Kernel&developer.KernelScreen == developer.KernelScreen {
				imgui.Text("S")
				imgui.SameLineV(0, 1)
			}
			if ln.Kernel&developer.KernelOverscan == developer.KernelOverscan {
				imgui.Text("O")
				imgui.SameLineV(0, 1)
			}
		}
	}

	imgui.EndTable()
}

func (win *winCoProcPerformance) tooltip(load developer.Load, ln *developer.SourceLine, withAsm bool) {
	imguiTooltip(func() {
		imgui.Text(ln.File.ShortFilename)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
		imgui.PopStyleColor()

		if win.showSrcAsmInTooltip {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			displaySourceFragments(ln, win.img.cols, true)
		}

		if ln.Kernel != developer.KernelAny {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			imgui.Text("Executed in: ")
			if ln.Kernel == developer.KernelUnstable {
				imgui.Text("   ROM Setup only")
			} else {
				if ln.Kernel&developer.KernelVBLANK == developer.KernelVBLANK {
					imgui.Text("   VBLANK")
				}
				if ln.Kernel&developer.KernelScreen == developer.KernelScreen {
					imgui.Text("   Visible Screen")
				}
				if ln.Kernel&developer.KernelOverscan == developer.KernelOverscan {
					imgui.Text("   Overscan")
				}
			}
		}

		if win.showSrcAsmInTooltip && withAsm && len(ln.Disassembly) > 0 {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.BeginTable("##disasmTable", 2)
			for _, asm := range ln.Disassembly {
				imgui.TableNextRow()

				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
				imgui.Text(fmt.Sprintf("%08x", asm.Addr))
				imgui.PopStyleColor()

				imgui.TableNextColumn()
				imgui.Text(asm.Instruction)
			}
			imgui.EndTable()
		}

	}, true)
}

// helpfunction to sort profiling data according to current spec
func (win *winCoProcPerformance) sort(src *developer.Source, f func(imgui.TableSortSpecs)) {
	sort := imgui.TableGetSortSpecs()
	if src.ExecutionProfileChanged || sort.SpecsDirty() || win.windowSortSpecDirty {
		//  always set which kernel to sort by
		win.windowSortSpecDirty = false
		src.SortedFunctions.SetKernel(win.kernelFocus)
		src.SortedFunctions.UseRawCyclesCounts(!win.percentileFigures)
		src.SortedLines.SetKernel(win.kernelFocus)
		src.SortedLines.UseRawCyclesCounts(!win.percentileFigures)
		f(sort)
		sort.ClearSpecsDirty()
	}
}
