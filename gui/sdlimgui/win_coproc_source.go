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
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcSourceID = "Coprocessor Source"
const winCoProcSourceMenu = "Source"

type winCoProcSource struct {
	img           *SdlImgui
	open          bool
	showAsm       bool
	optionsHeight float32

	scrollToFile string
	selectedLine int
	scrollTo     bool

	selectedFile          *developer.SourceFile
	selectedFileComboOpen bool

	// we pay special attention to the collapsed state of this window. this is
	// because we want the gotoSourceLine() function to uncollapse the window
	// when selected
	//
	// 1. isCollapsed is set on imgui.Begin()
	// 2. uncollapseNext is set to true on gotoSourceLine()
	// 3. it is set to false when scrollToCounter reaches zero
	// 4. imgui.Begin() is called with WindowFlagsNoCollapse if both
	//       isCollapsed and uncollapseNext are true
	isCollapsed    bool
	uncollapseNext bool
}

func newWinCoProcSource(img *SdlImgui) (window, error) {
	win := &winCoProcSource{
		img:     img,
		showAsm: true,
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
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{1200, 1000})

	var flgs imgui.WindowFlags
	if win.uncollapseNext && win.isCollapsed {
		flgs = imgui.WindowFlagsNoCollapse
		win.uncollapseNext = false
	} else {
		flgs = imgui.WindowFlagsNone
	}

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcSourceID)
	if !imgui.BeginV(title, &win.open, flgs) {
		win.isCollapsed = true
		imgui.End()
		return
	}
	win.isCollapsed = false
	defer imgui.End()

	// safely iterate over source code
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.Filenames) == 0 {
			imgui.Text("No source files available")
			return
		}

		if win.scrollTo && (win.selectedFile == nil || win.scrollToFile != win.selectedFile.Filename) {
			win.selectedFile = src.Files[win.scrollToFile]
		} else if win.selectedFile == nil {
			win.selectedFile = src.Files[src.Filenames[0]]
		}

		imgui.AlignTextToFramePadding()
		imgui.Text("Filename")
		imgui.SameLine()
		imgui.PushItemWidth(imgui.ContentRegionAvail().X)
		if imgui.BeginComboV("##selectedFile", win.selectedFile.ShortFilename, imgui.ComboFlagsHeightLargest) {
			for _, fn := range src.Filenames {
				if imgui.Selectable(src.Files[fn].ShortFilename) {
					win.selectedFile = src.Files[fn]
				}

				// set scroll on the first frame that the combo is open
				if !win.selectedFileComboOpen && fn != win.selectedFile.Filename {
					imgui.SetScrollHereY(0.0)
				}
			}

			imgui.EndCombo()

			// note that combo is open *after* it has been drawn
			win.selectedFileComboOpen = true
		} else {
			win.selectedFileComboOpen = false
		}
		imgui.PopItemWidth()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		// new child that contains the main scrollable table
		imgui.BeginChildV("##coprocSourceMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)

		imgui.PushFont(win.img.glsl.fonts.code)
		style := imgui.CurrentStyle()
		rowSize := style.CellPadding()
		rowSize.Y = float32(win.img.prefs.codeFontLineSpacing.Get().(int))
		imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, rowSize) // affects table row height
		imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, rowSize) // affects selectable height

		const numColumns = 5
		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingFixedFit
		imgui.BeginTableV("##coprocSourceTable", numColumns, flgs, imgui.Vec2{}, 0.0)

		// first column is a dummy column so that Selectable (span all columns) works correctly
		imgui.TableSetupColumnV("Icon", imgui.TableColumnFlagsNone, imgui.CalcTextSize("   ", true, 0.0).X, 0)
		imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNone, imgui.CalcTextSize("0.00% ", true, 0.0).X, 1)
		imgui.TableSetupColumnV("Avg", imgui.TableColumnFlagsNone, imgui.CalcTextSize("0.00% ", true, 0.0).X, 2)
		imgui.TableSetupColumnV("LineNumber", imgui.TableColumnFlagsNone, imgui.CalcTextSize("0000 ", true, 0.0).X, 3)

		var clipper imgui.ListClipper
		clipper.Begin(len(win.selectedFile.Lines))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				if i >= len(win.selectedFile.Lines) {
					break
				}

				ln := win.selectedFile.Lines[i]
				imgui.TableNextRow()

				// highlight selected line
				if ln.LineNumber == win.selectedLine {
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.CoProcSourceSelected)
				}

				// show chip icon and also tooltip if mouse is hovered on selectable
				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
				imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
				if len(ln.Disassembly) > 0 {
					if ln.IllegalCount > 0 {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceBug)
						imgui.SelectableV(string(fonts.CoProcBug), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
						imgui.PopStyleColor()
					} else {
						imgui.SelectableV(string(fonts.Chip), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
					}

					if win.showAsm {
						imguiTooltip(func() {
							// remove cell/item styling for the duration of the tooltip
							pad := style.CellPadding()
							item := style.ItemSpacing()
							imgui.PopStyleVarV(2)
							defer imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, pad)
							defer imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, item)

							imgui.Text(ln.File.ShortFilename)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
							imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
							imgui.PopStyleColor()
							imgui.Spacing()
							imgui.Separator()
							imgui.Spacing()
							imgui.BeginTable("##disasmTable", 3)
							for _, asm := range ln.Disassembly {
								imgui.TableNextRow()
								imgui.TableNextColumn()
								imgui.Text(fmt.Sprintf("%#08x", asm.Addr))
								imgui.TableNextColumn()
								imgui.Text(fmt.Sprintf("%04x", asm.Opcode))
								imgui.TableNextColumn()
								imgui.Text(asm.Instruction)
							}
							imgui.EndTable()
						}, true)
					}
				} else {
					imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
				}
				imgui.PopStyleColorV(2)

				// percentage of time taken by this line
				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
				if ld, ok := ln.Stats.FrameLoad(); ok {
					imgui.Text(fmt.Sprintf("%.02f", ld))
				} else if len(ln.Disassembly) > 0 {
					imgui.Text(" -")
				}
				imgui.PopStyleColor()

				// percentage of time taken by this line
				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
				if ld, ok := ln.Stats.AverageLoad(); ok {
					imgui.Text(fmt.Sprintf("%.02f", ld))
				} else if len(ln.Disassembly) > 0 {
					imgui.Text(" -")
				}
				imgui.PopStyleColor()

				// line numbering
				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
				imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
				imgui.PopStyleColor()

				// source line
				imgui.TableNextColumn()
				imgui.Text(ln.Content)
			}
		}

		// scroll to correct line
		if win.scrollTo {
			imgui.SetScrollY(clipper.ItemsHeight * float32(win.selectedLine-10))
			win.scrollTo = false
			win.uncollapseNext = false
		}

		imgui.EndTable()

		imgui.PopStyleVarV(2)
		imgui.PopFont()

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
					imgui.Text("source code analysis may be misleading")
				}, true)
				imgui.SameLineV(0, 20)
			}

			imgui.Checkbox("Show ASM in Tooltip", &win.showAsm)
		})
	})
}

func (win *winCoProcSource) gotoSourceLine(ln *developer.SourceLine) {
	win.setOpen(true)
	win.scrollTo = true
	win.scrollToFile = ln.File.Filename
	win.selectedLine = ln.LineNumber
	win.uncollapseNext = true
}
