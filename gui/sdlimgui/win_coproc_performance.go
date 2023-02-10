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
	"sync/atomic"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winCoProcPerformanceID = "Coprocessor Performance"
const winCoProcPerformanceMenu = "Performance"

type winCoProcPerformance struct {
	debuggerWin

	img *SdlImgui

	// source shown in tooltip
	showTooltip bool

	// which kernel to focus on
	kernelFocus         developer.KernelVCS
	kernelFocusComboDim imgui.Vec2

	// whether to present performance figures as raw counts or as percentages
	percentileFigures bool

	// for the function tab, show cumulative values rather than flat values
	cumulative bool

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

	// FrameTrigger interface used to synchronise reset event with the TV
	tv coProcPerformanceTV
}

// coProcePerformanceTV is used to synchronise the "Reset Statistics" button
// with the television.
type coProcPerformanceTV struct {
	img      *SdlImgui
	schedule atomic.Value
}

func (tv *coProcPerformanceTV) initialise(img *SdlImgui) {
	// initialise TV Frame Trigger
	tv.img = img
	tv.schedule.Store(false)
	tv.img.dbg.PushFunction(func() {
		tv.img.dbg.TV().AddFrameTrigger(tv)
	})
}

func (tv *coProcPerformanceTV) scheduleReset(set bool) {
	tv.schedule.Store(set)
}

// NewFrame implements the television.FrameTrigger interface.
func (tv *coProcPerformanceTV) NewFrame(_ television.FrameInfo) error {
	// this code is running in the emulator goroutine and NOT the GUI goroutine
	if tv.schedule.Load().(bool) {
		tv.schedule.Store(false)
		tv.img.dbg.CoProcDev.ResetStatistics()
	}
	return nil
}

func newWinCoProcPerformance(img *SdlImgui) (window, error) {
	win := &winCoProcPerformance{
		img:               img,
		showTooltip:       true,
		kernelFocus:       developer.KernelAny,
		percentileFigures: true,
	}

	win.tv.initialise(img)

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

	if !win.img.lz.Cart.HasCoProcBus {
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
		imgui.BeginTabBarV("##coprocPerformanceFunctions", imgui.TabBarFlagsAutoSelectNewTabs)

		functionTab := "Functions"
		if imgui.BeginTabItemV(functionTab, nil, imgui.TabItemFlagsNone) {
			if win.tabSelected != functionTab {
				win.tabSelected = functionTab
				win.windowSortSpecDirty = true
			}
			win.drawFunctions(src)
			imgui.EndTabItem()
		}

		sourceLineTab := "Source Lines"
		if imgui.BeginTabItemV(sourceLineTab, nil, imgui.TabItemFlagsNone) {
			if win.tabSelected != sourceLineTab {
				win.tabSelected = sourceLineTab
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

			funcFilterTab := fmt.Sprintf("%c %s", fonts.MagnifyingGlass, ff.FunctionName)
			if imgui.BeginTabItemV(funcFilterTab, &open, flgs) {
				if win.tabSelected != funcFilterTab {
					win.tabSelected = funcFilterTab
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

			if win.tabSelected == functionTab {
				imgui.SameLineV(0, 15)
				if imgui.Checkbox("Cumulative Figures", &win.cumulative) {
					win.windowSortSpecDirty = true
				}
			}

			// scale statistics to function is in drawFunctionFilter()
			imgui.Spacing()
			imgui.Checkbox("Show Tooltip", &win.showTooltip)

			// reset statistics
			if win.img.dbg.State() == govern.Paused {
				imgui.SameLineV(0, 15)
				if win.tv.schedule.Load().(bool) {
					if imgui.Button(fmt.Sprintf("%c Reset Now", fonts.Trash)) {
						win.tv.scheduleReset(false)
						src.ResetStatistics()
					}

					imgui.SameLineV(0, 15)
					imgui.BeginGroup()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Cancel)
					imgui.Text(string(fonts.Cancel))
					imgui.PopStyleColor()
					imgui.SameLineV(0, 5)
					imgui.Text("Statistics will be reset for the next TV frame")
					imgui.EndGroup()
					if imgui.IsItemClicked() {
						win.tv.scheduleReset(false)
					}
				} else {
					if imgui.Button(fmt.Sprintf("%c Reset Statistics", fonts.Trash)) {
						win.tv.scheduleReset(true)
					}
				}
			} else {
				imgui.SameLineV(0, 15)
				if imgui.Button(fmt.Sprintf("%c Reset Statistics", fonts.Trash)) {
					win.tv.scheduleReset(true)
				}
			}

			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			if src.Optimised {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
				imgui.Text(string(fonts.Warning))
				imgui.PopStyleColor()

				imguiTooltipSimple(`Source compiled with optimisation. Some figures may
be misleading`)

				imgui.SameLineV(0, 15)
			}

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
		if !src.Stats.VBLANK.HasExecuted() {
			imgui.Text("No functions have been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !src.Stats.Screen.HasExecuted() {
			imgui.Text("No functions have been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !src.Stats.Overscan.HasExecuted() {
			imgui.Text("No functions have been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.Overall.HasExecuted() {
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
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.05, 1)
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

		// using flat stats even if cumulative option is set - it amounts to
		// the same thing in this case
		if win.hideUnusedEntries && !fn.FlatStats.Overall.HasExecuted() {
			continue
		}

		// is the function is a stub or not. some facilities are not available
		// without source
		isStub := fn.IsStub()

		// select which stats to focus on
		var statsGroup developer.StatsGroup
		if win.cumulative {
			statsGroup = fn.CumulativeStats
		} else {
			statsGroup = fn.FlatStats
		}

		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = statsGroup.VBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = statsGroup.Screen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = statsGroup.Overscan
		} else {
			stats = statsGroup.Overall
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceFilename)
		if isStub {
			imgui.SelectableV("-", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		} else {
			imgui.SelectableV(fn.DeclLine.File.ShortFilename, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		}
		imgui.PopStyleColorV(3)

		// whether to show the optimisation warning for a function
		optimisedWarning := win.cumulative && fn.OptimisedCallStack

		// tooltip
		win.tooltip(fn.FlatStats.Overall.OverSource, fn, fn.DeclLine, false)

		if optimisedWarning {
			imguiTooltip(func() {
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
				imgui.Text(string(fonts.Warning))
				imgui.PopStyleColor()
				imgui.SameLineV(0, 5)
				imgui.Text("This function has been called as part of a call stack that could")
				imgui.Text("not be discerned accurately. The figures will be inaccurate.")
			}, true)
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

		if optimisedWarning {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
			imgui.Text(string(fonts.Warning))
			imgui.PopStyleColor()
			imgui.SameLineV(0, 5)
			imgui.Text(fn.Name)
		} else {
			imgui.Text(fn.Name)
		}

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
		if !src.Stats.VBLANK.HasExecuted() {
			imgui.Text("No lines have been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !src.Stats.Screen.HasExecuted() {
			imgui.Text("No lines have been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !src.Stats.Overscan.HasExecuted() {
			imgui.Text("No lines have been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.Overall.HasExecuted() {
			imgui.Text("No lines have been executed yet")
			return
		}
	}

	const numColumns = 7

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchProp
	flgs |= imgui.TableFlagsSortable
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

		if win.hideUnusedEntries && !ln.Stats.Overall.HasExecuted() {
			continue
		}

		// is the line is a stub or not. some facilities are not available
		// without source
		isStub := ln.IsStub()

		// select which stats to focus on
		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = ln.Stats.VBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = ln.Stats.Screen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = ln.Stats.Overscan
		} else {
			stats = ln.Stats.Overall
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
			win.tooltip(ln.Stats.Overall.OverSource, ln.Function, ln, true)
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
		win.img.drawSourceLine(ln, true)

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
		if !functionFilter.Function.FlatStats.VBLANK.HasExecuted() {
			imgui.Text("This function has not been executed during VBLANK yet")
			return
		}
	} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
		if !functionFilter.Function.FlatStats.Screen.HasExecuted() {
			imgui.Text("This function has not been executed during the visible screen yet")
			return
		}
	} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
		if !functionFilter.Function.FlatStats.Overscan.HasExecuted() {
			imgui.Text("This function has not been executed during Overscan yet")
			return
		}
	} else {
		if win.hideUnusedEntries && !src.Stats.Overall.HasExecuted() {
			imgui.Text("Ths function has not been executed yet")
			return
		}
	}

	// function summary in relation to the program
	imgui.AlignTextToFramePadding()
	switch win.kernelFocus {
	case developer.KernelVBLANK:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the VBLANK last frame", functionFilter.Function.FlatStats.VBLANK.OverSource.Frame))
	case developer.KernelScreen:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Visible Screen last frame", functionFilter.Function.FlatStats.Screen.OverSource.Frame))
	case developer.KernelOverscan:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Overscan last frame", functionFilter.Function.FlatStats.Overscan.OverSource.Frame))
	default:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time last frame", functionFilter.Function.FlatStats.Overall.OverSource.Frame))
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

		if win.hideUnusedEntries && !ln.Stats.Overall.HasExecuted() {
			continue
		}

		if ln.IsStub() {
			continue
		}

		// select which stats to focus on
		var stats developer.Stats
		if win.kernelFocus&developer.KernelVBLANK == developer.KernelVBLANK {
			stats = ln.Stats.VBLANK
		} else if win.kernelFocus&developer.KernelScreen == developer.KernelScreen {
			stats = ln.Stats.Screen
		} else if win.kernelFocus&developer.KernelOverscan == developer.KernelOverscan {
			stats = ln.Stats.Overscan
		} else {
			stats = ln.Stats.Overall
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
			win.tooltip(ln.Stats.Overall.OverFunction, ln.Function, ln, true)
		} else {
			win.tooltip(ln.Stats.Overall.OverSource, ln.Function, ln, true)
		}

		// open source window on click
		if imgui.IsItemClicked() {
			srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

		imgui.TableNextColumn()
		win.img.drawSourceLine(ln, true)

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

// single tooltip for all the differnt contexts in the performance window
//
// the load and fn arguments should never be nil. the ln argument can be nil if
// the function is a stub function
//
// showDisasm should be false if the context doesn't require disassembly detail
func (win *winCoProcPerformance) tooltip(load developer.Load,
	fn *developer.SourceFunction, ln *developer.SourceLine,
	showDisasm bool) {

	if !win.showTooltip {
		return
	}

	imguiTooltip(func() {
		if fn.IsStub() {
			if fn.Name == developer.DriverFunctionName {
				imgui.Text("Instructions that are executed")
				imgui.Text("outside of the ROM and in the driver")
			} else {
				imgui.Text("Function has no source code")
			}
		} else {
			win.img.drawFilenameAndLineNumber(ln.File.Filename, ln.LineNumber, -1)
		}

		if fn.Kernel != developer.KernelAny {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			imgui.Text(fmt.Sprintf("%c Executed in: ", fonts.CoProcKernel))
			if fn.Kernel == developer.KernelUnstable {
				imgui.Text("   ROM Setup only")
			} else {
				if fn.Kernel&developer.KernelVBLANK == developer.KernelVBLANK {
					imgui.Text("   VBLANK")
				}
				if fn.Kernel&developer.KernelScreen == developer.KernelScreen {
					imgui.Text("   Visible Screen")
				}
				if fn.Kernel&developer.KernelOverscan == developer.KernelOverscan {
					imgui.Text("   Overscan")
				}
			}
		}

		if showDisasm && len(ln.Disassembly) > 0 {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			win.img.drawDisasmForCoProc(ln.Disassembly, ln, false)
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
		src.SortedFunctions.SetCumulative(win.cumulative)
		src.SortedLines.SetKernel(win.kernelFocus)
		src.SortedLines.UseRawCyclesCounts(!win.percentileFigures)
		f(sort)
		sort.ClearSpecsDirty()
	}
}
