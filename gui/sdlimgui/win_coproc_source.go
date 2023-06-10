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
	"os"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

const winCoProcSourceID = "Coprocessor Source"
const winCoProcSourceMenu = "Source"

type winCoProcSource struct {
	debuggerWin

	img *SdlImgui

	open               bool
	showTooltip        bool
	syntaxHighlighting bool
	optionsHeight      float32

	selection      imguiSelection
	selectionRange developer.InstructionRange

	lineFuzzy fuzzyFilter

	selectedFileFuzzy     fuzzyFilter
	selectedShortFileName string

	// selectedFile will change whenever updateSelectedFile is true
	updateSelectedFile bool
	selectedFile       *developer.SourceFile

	// yield state is checked on every draw whether window is open or not. the
	// window will open if the yield state is new
	yieldState developer.YieldState
	yieldLine  *developer.SourceLine

	// focus source view on current yield line
	focusYieldLine bool

	// focus source view has been requested by the user. normally
	// focusYieldLine is ignore unless emulation is paused but this flag
	// supercedes that condition
	focusYieldLineManual bool

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

	// widths of columns in the source view table
	widthGutter float32
	widthStats  float32
	widthLine   float32

	isPaused bool
}

func newWinCoProcSource(img *SdlImgui) (window, error) {
	win := &winCoProcSource{
		img:                img,
		showTooltip:        true,
		syntaxHighlighting: true,
		focusYieldLine:     true,
	}
	return win, nil
}

func (win *winCoProcSource) init() {
	win.widthGutter = imgui.CalcTextSize(fmt.Sprintf("%c", fonts.Breakpoint), true, 0).X
	win.widthStats = imgui.CalcTextSize("00.0% ", true, 0).X
	win.widthLine = imgui.CalcTextSize("9999 ", true, 0).X
}

func (win *winCoProcSource) id() string {
	return winCoProcSourceID
}

const sourcePopupID = "sourcePopupID"

func (win *winCoProcSource) debuggerDraw() bool {
	if !win.img.lz.Cart.HasCoProcBus {
		return false
	}

	// check yield state and open the window if necessary
	win.img.dbg.CoProcDev.BorrowYieldState(func(yld *developer.YieldState) {
		if !yld.Cmp(&win.yieldState) {
			win.yieldState = *yld

			// open window and focus on yield line if the yield is a breakpoint
			if yld.Reason != mapper.YieldSyncWithVCS && yld.Reason != mapper.YieldProgramEnded {
				win.debuggerOpen = true
				win.focusYieldLine = true
			}
		}
	})

	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{81, 297}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{641, 517}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{2000, 1000})

	flgs := imgui.WindowFlagsNone
	if win.uncollapseNext && win.isCollapsed {
		flgs |= imgui.WindowFlagsNoCollapse
	}
	flgs |= imgui.WindowFlagsNoScrollWithMouse
	win.uncollapseNext = false

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcSourceID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, flgs) {
		win.isCollapsed = false
		win.draw()
	} else {
		win.isCollapsed = true
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcSource) draw() {
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

		// find yield line
		win.yieldLine = src.FindSourceLine(win.yieldState.InstructionPC)

		// focus on yield line (or main function if we don't have a yield line)
		// but only if emulation is paused
		if win.focusYieldLine {
			if win.img.dbg.State() == govern.Paused || win.focusYieldLineManual {
				focusLine := win.yieldLine

				// focusLine is the same as yieldLine. if yieldLine is invalid
				// we instead focus on the main function
				if focusLine == nil || focusLine.IsStub() {
					focusLine = src.MainFunction.DeclLine
				}

				// double check validity of focusLine
				if focusLine != nil && !focusLine.IsStub() {
					win.selectedShortFileName = focusLine.File.ShortFilename
					win.selection.single(focusLine.LineNumber)
					win.updateSelectedFile = true
				}

				// focus has been dealt with
				win.focusYieldLine = false
				win.focusYieldLineManual = false
			}
		}

		// fuzzy file selector
		win.drawFileSelection(src)
		imgui.Separator()

		// change selectedFile if update flag is set
		if win.updateSelectedFile {
			win.selectedFile = src.FilesByShortname[win.selectedShortFileName]
		}

		// source code view
		imgui.BeginGroup()
		win.drawSource(src)
		imgui.EndGroup()

		// we don't need updateSelectedFile after the call to drawSource() so
		// it is safe to reset
		if win.updateSelectedFile {
			win.updateSelectedFile = false
		}

		if imgui.IsMouseDown(1) && imgui.IsItemHovered() {
			imgui.OpenPopup(sourcePopupID)
		}

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			win.drawLineSearch(src)
			imgui.SameLineV(0, 10)

			if imgui.Button(fmt.Sprintf("%c Focus Yield Line", fonts.DisasmGotoCurrent)) {
				win.focusYieldLine = true
				win.focusYieldLineManual = true
			}
			imgui.SameLineV(0, 20)
			imgui.Checkbox("Highlight Comments & String Literals", &win.syntaxHighlighting)
			imgui.SameLineV(0, 20)
			imgui.Checkbox("Show Tooltip", &win.showTooltip)
		})

		if imgui.BeginPopup(sourcePopupID) {
			if imgui.Selectable(fmt.Sprintf("%c Save Source to CSV", fonts.Disk)) {
				win.saveToCSV(src)
			}
			imgui.EndPopup()
		}
	})
}

func (win *winCoProcSource) drawFileSelection(src *developer.Source) {
	if imgui.Button(string(fonts.Disk)) {
		mp := imgui.MousePos()
		mp.X += imgui.FontSize()
		mp.Y -= imgui.FontSize() * 2
		imgui.SetNextWindowPos(mp)
		imgui.OpenPopup("##filefuzzyPopup")
	}

	imgui.SameLineV(0, 15)
	imgui.AlignTextToFramePadding()
	if win.selectedShortFileName == "" {
		imgui.Text("No File Selected")
	} else {
		imgui.Text(win.selectedShortFileName)
	}

	w := imgui.WindowWidth()

	if imgui.BeginPopup("##filefuzzyPopup") {
		imgui.PushItemWidth(w)

		fuzzyFileHook := func(i int) {
			win.selectedShortFileName = src.ShortFilenames[i]
			win.selection.clear()
			win.updateSelectedFile = true
		}

		if !win.selectedFileFuzzy.draw("##selectedFileFuzzy", src.ShortFilenames, fuzzyFileHook, true) {
			imgui.CloseCurrentPopup()
		}

		imgui.PopItemWidth()
		imgui.EndPopup()
	}
}

func (win *winCoProcSource) drawLineSearch(src *developer.Source) {
	if imgui.Button(string(fonts.MagnifyingGlass)) {
		mp := imgui.MousePos()
		mp.X += imgui.FontSize()
		mp.Y -= imgui.FontSize() * 12
		imgui.SetNextWindowPos(mp)
		imgui.OpenPopup("##linefuzzyPopup")
	}

	w := imgui.WindowWidth()

	if imgui.BeginPopup("##linefuzzyPopup") {
		imgui.PushItemWidth(w)

		lineFuzzyHook := func(i int) {
			win.gotoSourceLine(src.AllLines.Get(i))
		}

		if !win.lineFuzzy.draw("##linefuzzy", src.AllLines, lineFuzzyHook, false) {
			imgui.CloseCurrentPopup()
		}

		imgui.PopItemWidth()
		imgui.EndPopup()
	}
}

func (win *winCoProcSource) gotoSourceLine(ln *developer.SourceLine) {
	if ln == nil {
		return
	}
	win.debuggerSetOpen(true)
	win.focusYieldLine = false
	win.selectedShortFileName = ln.File.ShortFilename
	win.selection.single(ln.LineNumber)
	win.uncollapseNext = true
	win.updateSelectedFile = true
}

func (win *winCoProcSource) saveToCSV(src *developer.Source) {
	// open unique file
	fn := unique.Filename("source", win.img.lz.Cart.Shortname)
	fn = fmt.Sprintf("%s.csv", fn)
	f, err := os.Create(fn)
	if err != nil {
		logger.Logf("sdlimgui", "could not save source CSV: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf("sdlimgui", "error saving source CSV: %v", err)
		}
	}()

	// write string to CSV file
	writeEntry := func(s string) {
		f.WriteString(s)
		f.WriteString("\n")
	}

	for _, ln := range win.selectedFile.Content.Lines {
		s := strings.Builder{}
		if ln.Stats.Overall.OverSource.FrameValid {
			s.WriteString(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Frame))
		} else if ln.Stats.Overall.OverSource.AverageValid {
			s.WriteString(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Average))
		} else if ln.Stats.Overall.OverSource.MaxValid {
			s.WriteString(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Max))
		} else {
			s.WriteString(" -")
		}
		s.WriteRune(',')

		// replace comma with "Arabic Decimal Separator" in source code. This
		// is so that the command doesn't interfere with the CSV format
		const arabicDecimalSeparator = '\u066b'
		s.WriteString(strings.ReplaceAll(ln.PlainContent, ",", string(arabicDecimalSeparator)))

		writeEntry(s.String())
	}
}

func (win *winCoProcSource) drawSource(src *developer.Source) {
	// new child that contains the main scrollable table
	imgui.BeginChildV("##coprocSourceMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)
	defer imgui.EndChild()

	if win.selectedFile == nil {
		imgui.Text("No source file selected")
		return
	}

	imgui.PushFont(win.img.glsl.fonts.code)
	lineSpacing := float32(win.img.prefs.codeFontLineSpacing.Get().(int))

	// push cell padding and item spacing style such that we can have
	// variable height rows (according to lineSpacing setting) and a
	// Selectable() that leaves no gaps between the rows
	style := imgui.CurrentStyle()
	rowSize := style.CellPadding()
	rowSize.Y = lineSpacing
	imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, rowSize) // affects table row height
	rowSize.Y += lineSpacing
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, rowSize) // affects selectable height

	const numColumns = 5
	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsScrollX
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsNoPadInnerX
	if imgui.BeginTableV("##coprocSourceTable", numColumns, flgs, imgui.Vec2{}, 0.0) {
		// first column is a dummy column so that Selectable (span all columns)
		// works correctly. if we drew the icon column for example, as a
		// Selectable then the drag selection won't work correctly. this is
		// because the icon can change and IsItemHovered() won't recognise a
		// selectable if it has a different label. by using this dummy column
		// we can ensure that every selectable has the same label
		imgui.TableSetupColumnV("##selection", imgui.TableColumnFlagsNone, win.widthGutter*0.75, 0)

		// next three columns have fixed width
		imgui.TableSetupColumnV("##gutter", imgui.TableColumnFlagsNone, win.widthGutter*1.5, 1)
		imgui.TableSetupColumnV("##load", imgui.TableColumnFlagsNone, win.widthStats, 2)
		imgui.TableSetupColumnV("##number", imgui.TableColumnFlagsNone, win.widthLine, 3)

		// content column width is set to the maximum line width for the
		// selected file. this is so that the horizontal scroll bar doesn't
		// change size as we scroll
		var w float32
		if win.selectedFile != nil {
			w = imguiTextWidth(win.selectedFile.Content.MaxLineWidth)
		}
		imgui.TableSetupColumnV("Content", imgui.TableColumnFlagsNone, w, 4)

		// draw execution indicator
		executionIndicatorStart := imgui.CursorScreenPos()
		executionIndicatorCol := win.img.cols.windowBg
		executionIndicatorAdj := imgui.CurrentStyle().FramePadding()
		drawExecutionIndicator := func(col imgui.PackedColor) {
			executionIndicatorEnd := imgui.CursorScreenPos()
			executionIndicatorEnd.X = executionIndicatorStart.X
			executionIndicatorEnd.Y -= executionIndicatorAdj.Y

			dl := imgui.WindowDrawList()
			dl.AddLineV(executionIndicatorStart, executionIndicatorEnd, executionIndicatorCol, 5.0)

			executionIndicatorStart = executionIndicatorEnd
			executionIndicatorCol = col
		}

		if win.selectedFile != nil {
			var clipper imgui.ListClipper
			clipper.Begin(win.selectedFile.Content.Len())
			for clipper.Step() {
				for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
					if i >= win.selectedFile.Content.Len() {
						break
					}

					ln := win.selectedFile.Content.Lines[i]
					imgui.TableNextRow()

					// highlight selected line(s)
					if win.selection.inRange(ln.LineNumber) {
						imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.CoProcSourceSelected)
					}

					// highlight yield line
					if win.yieldLine != nil && win.yieldLine.File != nil {
						if win.yieldLine.LineNumber == ln.LineNumber && win.yieldLine.File == win.selectedFile {
							if win.yieldLine.Bug {
								imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.CoProcSourceYieldBug)
							} else {
								imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.CoProcSourceYield)
							}
						}
					}

					imgui.TableNextColumn()
					imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
					imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
					imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
					imgui.PopStyleColorV(2)

					// allow breakpoint toggle for lines with executable entries
					if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
						src.ToggleBreakpoint(ln)
					}

					// is the sourceline an executable line
					var hoverExecutableLine bool

					// select source lines with mouse click and drag
					if imgui.IsItemHovered() {
						// add/remove lines before showing the tooltip. this
						// produces better visual results
						if imgui.IsMouseClicked(0) {
							win.selection.dragStart(ln.LineNumber)
						}
						if imgui.IsMouseDragging(0, 0.0) {
							win.selection.drag(ln.LineNumber)
							win.selectionRange.Clear()
							s, e := win.selection.limits()
							for i := s; i <= e; i++ {
								win.selectionRange.Add(win.selectedFile.Content.Lines[i-1])
							}
						}

						multiline := !win.selection.isSingle() && win.selection.inRange(ln.LineNumber)
						hoverExecutableLine = (!multiline && len(ln.Instruction) > 0) ||
							(multiline && !win.selectionRange.IsEmpty())

						if win.showTooltip {
							// how we show the asm depends on whether there are
							// multiple lines selected and whether there is any
							// diassembly for those lines.
							//
							// if only a single line is selected then we simply
							// check that there is are asm entries for that line
							if hoverExecutableLine {
								imgui.PopFont()

								imguiTooltip(func() {
									// remove cell/item styling for the duration of the tooltip
									pad := style.CellPadding()
									item := style.ItemSpacing()
									imgui.PopStyleVarV(2)
									defer imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, pad)
									defer imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, item)

									// this block is a more developed version of win.img.drawFilenameAndLineNumber()
									// there is no need to complicate that function
									if multiline && !win.selection.isSingle() {
										s, e := win.selection.limits()
										win.img.drawFilenameAndLineNumber(ln.File.Filename, s, e)
									} else {
										win.img.drawFilenameAndLineNumber(ln.File.Filename, ln.LineNumber, -1)
									}

									imgui.Spacing()
									imgui.Separator()
									imgui.Spacing()

									// choose which disasm list to use
									disasm := ln.Instruction
									if multiline {
										disasm = win.selectionRange.Instructions
									}

									win.img.drawDisasmForCoProc(disasm, ln, multiline)

									if ln.Function.IsInlined() {
										imgui.Spacing()
										imgui.Separator()
										imgui.Spacing()
										imgui.Text(fmt.Sprintf("%c This function is inlined", fonts.Inlined))
									}
								}, false)

								imgui.PushFont(win.img.glsl.fonts.code)
							}
						}
					}

					// show appropriate icon in the gutter
					imgui.TableNextColumn()
					if src.CheckBreakpoint(ln) {
						// the presence of a breakpoint is the most important information
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
						imgui.Text(string(fonts.Breakpoint))
						imgui.PopStyleColor()
					} else if hoverExecutableLine && src.CanBreakpoint(ln) {
						// prioritise showing the breakpoint indicator over the bug symbol
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress.Times(0.6))
						imgui.Text(string(fonts.Breakpoint))
						imgui.PopStyleColor()
					} else if ln.Bug {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceBug)
						imgui.Text(string(fonts.CoProcBug))
						imgui.PopStyleColor()
					}

					// execution state
					imgui.TableNextColumn()
					if ln.Stats.Overall.HasExecuted() {
						if ln.Stats.Overall.OverSource.FrameValid {
							drawExecutionIndicator(win.img.cols.coProcSourceLoad)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Frame))
							imgui.PopStyleColor()
						} else if ln.Stats.Overall.OverSource.AverageValid {
							drawExecutionIndicator(win.img.cols.coProcSourceAvgLoad)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Average))
							imgui.PopStyleColor()
						} else if ln.Stats.Overall.OverSource.MaxValid {
							drawExecutionIndicator(win.img.cols.coProcSourceMaxLoad)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Max))
							imgui.PopStyleColor()
						} else {
							drawExecutionIndicator(win.img.cols.coProcSourceNoLoad)
						}
					} else if len(ln.Instruction) > 0 {
						drawExecutionIndicator(win.img.cols.coProcSourceNoLoad)
					} else {
						drawExecutionIndicator(win.img.cols.windowBg)
					}

					// line numbering
					imgui.TableNextColumn()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
					imgui.Text(fmt.Sprintf("%d", ln.LineNumber))
					imgui.PopStyleColor()

					// source line
					imgui.TableNextColumn()

					if win.syntaxHighlighting {
						win.img.drawSourceLine(ln, false)
					} else {
						imgui.Text(ln.PlainContent)
					}

				}
				drawExecutionIndicator(win.img.cols.windowBg)
			}

			// scroll to correct line
			if win.updateSelectedFile {
				s, _ := win.selection.limits()
				imgui.SetScrollY(clipper.ItemsHeight * float32(s-10))
			}
		}

		imgui.EndTable()
	}

	imgui.PopStyleVarV(2)
	imgui.PopFont()
}
