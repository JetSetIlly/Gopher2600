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
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winCoProcProfilingID = "Coprocessor Profiling"
const winCoProcProfilingMenu = "Profiling"

type winCoProcProfiling struct {
	debuggerWin

	img *SdlImgui

	// which category of execution to focus on
	focus         profiling.Focus
	focusComboDim imgui.Vec2

	// whether to present performance figures as raw counts or as percentages
	percentileFigures bool

	// for the function tab, show cumulative values rather than flat values
	cumulative bool

	// show the number of times the function has been called in the current frame
	numberOfCalls bool

	// whether the sort criteria as specified in the window has changed (ie.
	// focus has changed)
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
	tv coProcProfilingTV
}

// coProceProfilingTV is used to synchronise the "Reset Statistics" button with the television.
type coProcProfilingTV struct {
	img      *SdlImgui
	schedule atomic.Value
}

func (tv *coProcProfilingTV) initialise(img *SdlImgui) {
	// initialise TV Frame Trigger
	tv.img = img
	tv.schedule.Store(false)
	tv.img.dbg.PushFunction(func() {
		tv.img.dbg.TV().AddFrameTrigger(tv)
	})
}

func (tv *coProcProfilingTV) scheduleReset(set bool) {
	tv.schedule.Store(set)
}

// NewFrame implements the television.FrameTrigger interface.
func (tv *coProcProfilingTV) NewFrame(_ television.FrameInfo) error {
	// this code is running in the emulator goroutine and NOT the GUI goroutine
	if tv.schedule.Load().(bool) {
		tv.schedule.Store(false)
		tv.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
			src.ResetStatistics()
		})
	}
	return nil
}

func newWinCoProcProfiling(img *SdlImgui) (window, error) {
	win := &winCoProcProfiling{
		img:               img,
		focus:             profiling.FocusAll,
		percentileFigures: true,
	}

	win.tv.initialise(img)

	return win, nil
}

func (win *winCoProcProfiling) init() {
	win.focusComboDim = imguiGetFrameDim("", profiling.FocusOptions...)
}

func (win *winCoProcProfiling) id() string {
	return winCoProcProfilingID
}

func (win *winCoProcProfiling) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 858, Y: 319}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 641, Y: 517}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcProfilingID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw(coproc)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcProfiling) draw(coproc coprocessor.CartCoProc) {
	// safely iterate over top execution information
	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
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

		imgui.BeginChildV("##coprocProfilingMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)
		imgui.BeginTabBarV("##coprocProfilingFunctions", imgui.TabBarFlagsAutoSelectNewTabs)

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

			imguiLabel("Focus")
			imgui.PushItemWidth(win.focusComboDim.X + imgui.FrameHeight())
			if imgui.BeginCombo("##focus", win.focus.String()) {
				if imgui.Selectable(profiling.FocusAll.String()) {
					win.focus = profiling.FocusAll
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(profiling.FocusVBLANK.String()) {
					win.focus = profiling.FocusVBLANK
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(profiling.FocusScreen.String()) {
					win.focus = profiling.FocusScreen
					win.windowSortSpecDirty = true
				}
				if imgui.Selectable(profiling.FocusOverscan.String()) {
					win.focus = profiling.FocusOverscan
					win.windowSortSpecDirty = true
				}
				imgui.EndCombo()
			}

			// reset statistics
			imgui.SameLineV(0, 15)
			if win.img.dbg.State() == govern.Paused {
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
				if imgui.Button(fmt.Sprintf("%c Reset Statistics", fonts.Trash)) {
					win.tv.scheduleReset(true)
				}
			}

			imgui.Spacing()

			imgui.Checkbox("Hide Unexecuted Items", &win.hideUnusedEntries)

			if win.tabSelected == functionTab {
				imgui.SameLineV(0, 15)
				if imgui.Checkbox("Number of Calls", &win.numberOfCalls) {
					win.windowSortSpecDirty = true
				}
			}

			drawDisabled(win.numberOfCalls, func() {
				imgui.SameLineV(0, 15)
				if imgui.Checkbox("Percentile Figures", &win.percentileFigures) {
					win.windowSortSpecDirty = true
				}

				if win.tabSelected == functionTab {
					imgui.SameLineV(0, 15)
					if imgui.Checkbox("Cumulative Figures", &win.cumulative) {
						win.windowSortSpecDirty = true
					}
				}
			})

			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			if src.Optimised {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
				imgui.Text(string(fonts.Warning))
				imgui.PopStyleColor()

				win.img.imguiTooltipSimple(`Source compiled with optimisation. Some figures may be misleading`)

				imgui.SameLineV(0, 15)
			}

			win.drawFrameStats(coproc)
		})
	})
}

func (win *winCoProcProfiling) drawFrameStats(coproc coprocessor.CartCoProc) {
	accumulate := func(s coprocessor.CoProcSynchronisation) int {
		switch s {
		case coprocessor.CoProcIdle:
		case coprocessor.CoProcNOPFeed:
			return 1
		case coprocessor.CoProcStrongARMFeed:
		case coprocessor.CoProcParallel:
			return 1
		}
		return 0
	}

	win.img.screen.crit.section.Lock()
	defer win.img.screen.crit.section.Unlock()

	// decide what to focus on
	var focus string
	var focusClocks float32

	switch win.focus {
	case profiling.FocusAll:
		focusClocks = float32(win.img.screen.crit.frameInfo.TotalClocks())
		focus = "TV Frame"
	case profiling.FocusScreen:
		focusClocks = float32(win.img.screen.crit.frameInfo.ScreenClocks())
		focus = "Screen"
	case profiling.FocusVBLANK:
		focusClocks = float32(win.img.screen.crit.frameInfo.VBLANKClocks())
		focus = "VBLANK"
	case profiling.FocusOverscan:
		focusClocks = float32(win.img.screen.crit.frameInfo.OverscanClocks())
		focus = "Overscan"
	}

	// frame statistics are taken from reflection information
	var clockCount float32

	for i, r := range win.img.screen.crit.reflection {
		sl := i / specification.ClksScanline

		switch win.focus {
		case profiling.FocusAll:
			clockCount += float32(accumulate(r.CoProcSync))
		case profiling.FocusScreen:
			if sl >= win.img.screen.crit.frameInfo.VisibleTop && sl <= win.img.screen.crit.frameInfo.VisibleBottom {
				clockCount += float32(accumulate(r.CoProcSync))
			}
		case profiling.FocusVBLANK:
			if sl < win.img.screen.crit.frameInfo.VisibleTop {
				clockCount += float32(accumulate(r.CoProcSync))
			}
		case profiling.FocusOverscan:
			if sl > win.img.screen.crit.frameInfo.VisibleBottom {
				clockCount += float32(accumulate(r.CoProcSync))
			}
		}
	}

	if clockCount > 0 {
		imgui.Text(fmt.Sprintf("%s activity in most recent %s:", coproc.ProcessorID(), focus))
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
		imgui.Text(fmt.Sprintf("%.02f%%", clockCount/focusClocks*100))
		imgui.PopStyleColor()
	} else if win.focus == profiling.FocusAll {
		imgui.Text(fmt.Sprintf("No %s activity in the most recent frame", coproc.ProcessorID()))
	} else {
		imgui.Text(fmt.Sprintf("No %s activity during %s", coproc.ProcessorID(), focus))
	}
}

func (win *winCoProcProfiling) drawFunctions(src *dwarf.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedFunctions.Functions) == 0 {
		imgui.Text("No performance profile")
		return
	}

	if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
		if !src.Stats.VBLANK.HasExecuted() {
			imgui.Text("No functions have been executed during VBLANK yet")
			return
		}
	} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
		if !src.Stats.Screen.HasExecuted() {
			imgui.Text("No functions have been executed during the visible screen yet")
			return
		}
	} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
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

	// the layout of the table changes significantly if the numberOfCalls option
	// is set. it is good to indicate that this is a different table in this
	// case because it means the sort column and column sizing will change along
	// with the option
	var title string
	if win.numberOfCalls {
		title = "##coprocProfilingTableFunctionsNumCalls"
	} else {
		title = "##coprocProfilingTableFunctions"
	}

	imgui.BeginTableV(title, numColumns, flgs, imgui.Vec2{}, 0.0)

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("File", imgui.TableColumnFlagsPreferSortDescending, width*0.275, 0)
	imgui.TableSetupColumnV("Line", imgui.TableColumnFlagsNoSort, width*0.05, 1)
	imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsPreferSortDescending, width*0.320, 2)
	if win.numberOfCalls {
		coords := win.img.cache.TV.GetCoords()

		var title [3]string

		switch coords.Frame {
		case 0:
			title[0] = fmt.Sprintf("Frame %d##current", coords.Frame)
			title[1] = "Frame -##previous"
			title[2] = "Frame -##prior"
		case 1:
			title[0] = fmt.Sprintf("Frame %d##current", coords.Frame)
			title[1] = fmt.Sprintf("Frame %d##previous", coords.Frame-1)
			title[2] = "Frame -##prior"
		default:
			title[0] = fmt.Sprintf("Frame %d##current", coords.Frame)
			title[1] = fmt.Sprintf("Frame %d##previous", coords.Frame-1)
			title[2] = fmt.Sprintf("Frame %d##prior", coords.Frame-2)
		}

		imgui.TableSetupColumnV(title[0], imgui.TableColumnFlagsNoSortAscending, width*0.1, 3)
		imgui.TableSetupColumnV(title[1], imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 4)
		imgui.TableSetupColumnV(title[2], imgui.TableColumnFlagsNoSortAscending, width*0.1, 5)
	} else {
		imgui.TableSetupColumnV("Frame", imgui.TableColumnFlagsNoSortAscending|imgui.TableColumnFlagsDefaultSort, width*0.1, 3)
		imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNoSortAscending, width*0.1, 4)
		imgui.TableSetupColumnV("Max", imgui.TableColumnFlagsNoSortAscending, width*0.1, 5)
	}
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
				if win.numberOfCalls {
					src.SortedFunctions.SortByNumCalls(true, 0)
				} else {
					src.SortedFunctions.SortByFrameCycles(true)
				}
			case 4:
				if win.numberOfCalls {
					src.SortedFunctions.SortByNumCalls(true, 1)
				} else {
					src.SortedFunctions.SortByAverageCycles(true)
				}
			case 5:
				if win.numberOfCalls {
					src.SortedFunctions.SortByNumCalls(true, 2)
				} else {
					src.SortedFunctions.SortByMaxCycles(true)
				}
			}
		}
		sort.ClearSpecsDirty()
	})

	for _, fn := range src.SortedFunctions.Functions {
		if fn.Kernel&win.focus != win.focus {
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
		var statsGroup profiling.StatsGroup
		if win.cumulative {
			statsGroup = fn.CumulativeStats
		} else {
			statsGroup = fn.FlatStats
		}

		var stats profiling.Stats
		if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
			stats = statsGroup.VBLANK
		} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
			stats = statsGroup.Screen
		} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
			stats = statsGroup.Overscan
		} else {
			stats = statsGroup.Overall
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
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
		win.tooltip(fn.FlatStats.Overall.OverProgram, fn, fn.DeclLine, false)

		if optimisedWarning {
			win.img.imguiTooltip(func() {
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

		if win.numberOfCalls {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceCurrent)
			imgui.TableNextColumn()
			if fn.NumCallsInFrame[0] > 0 {
				imgui.Text(fmt.Sprintf("%d", fn.NumCallsInFrame[0]))
			} else {
				imgui.Text("-")
			}
			imgui.PopStyleColor()

			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourcePrior)
			imgui.TableNextColumn()
			if fn.NumCallsInFrame[1] > 0 {
				imgui.Text(fmt.Sprintf("%d", fn.NumCallsInFrame[1]))
			} else {
				imgui.Text("-")
			}

			imgui.TableNextColumn()
			if fn.NumCallsInFrame[2] > 0 {
				imgui.Text(fmt.Sprintf("%d", fn.NumCallsInFrame[2]))
			} else {
				imgui.Text("-")
			}
			imgui.PopStyleColor()
		} else {
			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
			if stats.OverProgram.FrameValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Frame))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.FrameCount))
				}
			} else {
				imgui.Text("-")
			}
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
			if stats.OverProgram.AverageValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Average))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.AverageCount))
				}
			} else {
				imgui.Text("-")
			}
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
			if stats.OverProgram.MaxValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Max))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.MaxCount))
				}
			} else {
				imgui.Text("-")
			}
			imgui.PopStyleColor()

		}

		imgui.TableNextColumn()
		if fn.Kernel&profiling.FocusVBLANK == profiling.FocusVBLANK {
			imgui.Text("V")
			imgui.SameLineV(0, 1)
		}
		if fn.Kernel&profiling.FocusScreen == profiling.FocusScreen {
			imgui.Text("S")
			imgui.SameLineV(0, 1)
		}
		if fn.Kernel&profiling.FocusOverscan == profiling.FocusOverscan {
			imgui.Text("O")
			imgui.SameLineV(0, 1)
		}
	}

	imgui.EndTable()
}

func (win *winCoProcProfiling) drawSourceLines(src *dwarf.Source) {
	imgui.Spacing()

	if src == nil || len(src.SortedLines.Lines) == 0 {
		imgui.Text("No performance profile")
		return
	}

	if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
		if !src.Stats.VBLANK.HasExecuted() {
			imgui.Text("No lines have been executed during VBLANK yet")
			return
		}
	} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
		if !src.Stats.Screen.HasExecuted() {
			imgui.Text("No lines have been executed during the visible screen yet")
			return
		}
	} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
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

	imgui.BeginTableV("##coprocProfilingTableSourceLines", numColumns, flgs, imgui.Vec2{}, 0.0)

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
		if ln.Kernel&win.focus != win.focus {
			continue
		}

		if win.hideUnusedEntries && !ln.Stats.Overall.HasExecuted() {
			continue
		}

		// is the line is a stub or not. some facilities are not available
		// without source
		isStub := ln.IsStub()

		// select which stats to focus on
		var stats profiling.Stats
		if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
			stats = ln.Stats.VBLANK
		} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
			stats = ln.Stats.Screen
		} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
			stats = ln.Stats.Overscan
		} else {
			stats = ln.Stats.Overall
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
		imgui.SelectableV(ln.Function.Name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// tooltip
		if isStub {
			win.img.imguiTooltipSimple(fmt.Sprintf("This entry represent all lines of code in %s", ln.Function.Name))
		} else {
			win.tooltip(ln.Stats.Overall.OverProgram, ln.Function, ln, true)
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
		if stats.OverProgram.FrameValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Frame))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.FrameCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
		if stats.OverProgram.AverageValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Average))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.AverageCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
		if stats.OverProgram.MaxValid {
			if win.percentileFigures {
				imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Max))
			} else {
				imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.MaxCount))
			}
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if ln.Kernel&profiling.FocusVBLANK == profiling.FocusVBLANK {
			imgui.Text("V")
			imgui.SameLineV(0, 1)
		}
		if ln.Kernel&profiling.FocusScreen == profiling.FocusScreen {
			imgui.Text("S")
			imgui.SameLineV(0, 1)
		}
		if ln.Kernel&profiling.FocusOverscan == profiling.FocusOverscan {
			imgui.Text("O")
			imgui.SameLineV(0, 1)
		}
	}

	imgui.EndTable()
}

func (win *winCoProcProfiling) drawFunctionFilter(src *dwarf.Source, functionFilter *dwarf.FunctionFilter) {
	imgui.Spacing()

	if len(functionFilter.Lines.Lines) == 0 {
		imgui.Text(fmt.Sprintf("%s contains no executable lines", functionFilter.FunctionName))
		return
	}

	// validity check is done on function filter stats - drawFunctions() and
	// drawSourceLines() equivalents of this block perform the validity check
	// on the source level statistics
	if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
		if !functionFilter.Function.FlatStats.VBLANK.HasExecuted() {
			imgui.Text("This function has not been executed during VBLANK yet")
			return
		}
	} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
		if !functionFilter.Function.FlatStats.Screen.HasExecuted() {
			imgui.Text("This function has not been executed during the visible screen yet")
			return
		}
	} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
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
	switch win.focus {
	case profiling.FocusVBLANK:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the VBLANK last frame", functionFilter.Function.FlatStats.VBLANK.OverProgram.Frame))
	case profiling.FocusScreen:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Visible Screen last frame", functionFilter.Function.FlatStats.Screen.OverProgram.Frame))
	case profiling.FocusOverscan:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time in the Overscan last frame", functionFilter.Function.FlatStats.Overscan.OverProgram.Frame))
	default:
		imgui.Text(fmt.Sprintf("Function accounted for %.02f%% of ARM time last frame", functionFilter.Function.FlatStats.Overall.OverProgram.Frame))
	}

	imgui.SameLineV(0, 15)
	if imgui.Checkbox("Scale Statistics", &win.functionTabScale) {
		win.windowSortSpecDirty = true
	}
	win.img.imguiTooltipSimple(`When selected the % values in the table are
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

	imgui.BeginTableV("##coprocProfilingTableFunctionFilter", numColumns, flgs, imgui.Vec2{}, 0.0)

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
		if ln.Kernel&win.focus != win.focus {
			continue
		}

		if win.hideUnusedEntries && !ln.Stats.Overall.HasExecuted() {
			continue
		}

		if ln.IsStub() {
			continue
		}

		// select which stats to focus on
		var stats profiling.Stats
		if win.focus&profiling.FocusVBLANK == profiling.FocusVBLANK {
			stats = ln.Stats.VBLANK
		} else if win.focus&profiling.FocusScreen == profiling.FocusScreen {
			stats = ln.Stats.Screen
		} else if win.focus&profiling.FocusOverscan == profiling.FocusOverscan {
			stats = ln.Stats.Overscan
		} else {
			stats = ln.Stats.Overall
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()

		// selectable across entire width of table
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
		imgui.SelectableV(fmt.Sprintf("%d", ln.LineNumber), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(3)

		// source on tooltip
		if win.functionTabScale {
			win.tooltip(ln.Stats.Overall.OverFunction, ln.Function, ln, true)
		} else {
			win.tooltip(ln.Stats.Overall.OverProgram, ln.Function, ln, true)
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
			if stats.OverProgram.FrameValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Frame))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.FrameCount))
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
			if stats.OverProgram.AverageValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Average))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.AverageCount))
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
			if stats.OverProgram.MaxValid {
				if win.percentileFigures {
					imgui.Text(fmt.Sprintf("%.02f", stats.OverProgram.Max))
				} else {
					imgui.Text(fmt.Sprintf("%.0f", stats.OverProgram.MaxCount))
				}
			} else {
				imgui.Text("-")
			}
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if ln.Kernel&profiling.FocusVBLANK == profiling.FocusVBLANK {
			imgui.Text("V")
			imgui.SameLineV(0, 1)
		}
		if ln.Kernel&profiling.FocusScreen == profiling.FocusScreen {
			imgui.Text("S")
			imgui.SameLineV(0, 1)
		}
		if ln.Kernel&profiling.FocusOverscan == profiling.FocusOverscan {
			imgui.Text("O")
			imgui.SameLineV(0, 1)
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
func (win *winCoProcProfiling) tooltip(load profiling.Load,
	fn *dwarf.SourceFunction, ln *dwarf.SourceLine,
	showDisasm bool) {

	win.img.imguiTooltip(func() {
		if fn.IsStub() {
			if fn.Name == dwarf.DriverFunctionName {
				imgui.Text("Instructions that are executed")
				imgui.Text("outside of the ROM and in the driver")
			} else {
				imgui.Text("Function has no source code")
			}
		} else {
			win.img.drawFilenameAndLineNumber(ln.File.Filename, ln.LineNumber, -1)
		}

		if fn.Kernel != profiling.FocusAll {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			imgui.Text(fmt.Sprintf("%c Executed in: ", fonts.CoProcKernel))
			if fn.Kernel&profiling.FocusVBLANK == profiling.FocusVBLANK {
				imgui.Text("   VBLANK")
			}
			if fn.Kernel&profiling.FocusScreen == profiling.FocusScreen {
				imgui.Text("   Visible Screen")
			}
			if fn.Kernel&profiling.FocusOverscan == profiling.FocusOverscan {
				imgui.Text("   Overscan")
			}
		}

		if showDisasm && len(ln.Instruction) > 0 {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			win.img.drawDisasmForCoProc(ln.Instruction, ln, false, false, 0, shortDisasmWindow)
		}

	}, true)
}

// helpfunction to sort profiling data according to current spec
func (win *winCoProcProfiling) sort(src *dwarf.Source, f func(imgui.TableSortSpecs)) {
	sort := imgui.TableGetSortSpecs()
	if src.ExecutionProfileChanged || sort.SpecsDirty() || win.windowSortSpecDirty {
		win.windowSortSpecDirty = false
		src.SortedFunctions.SetFocus(win.focus)
		src.SortedFunctions.UseRawCyclesCounts(!win.percentileFigures)
		src.SortedFunctions.SetCumulative(win.cumulative)
		src.SortedLines.SetKernel(win.focus)
		src.SortedLines.UseRawCyclesCounts(!win.percentileFigures)
		f(sort)
		sort.ClearSpecsDirty()
	}
}
