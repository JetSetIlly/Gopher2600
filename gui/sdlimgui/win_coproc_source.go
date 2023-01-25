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

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcSourceID = "Coprocessor Source"
const winCoProcSourceMenu = "Source"

// type that can test whether a number is in between (incluside) of the two
// values in the array
//
// the start of the range is index 0 and the end of the range is index 1
//
// the inRange() function should be used for inclusion testing and the
// ordered() function will return the start and end values such that start
// is always less than or equal to the end value.
//
// also handles the developer.DisasmRange
type lineRange struct {
	start  int
	end    int
	disasm developer.DisasmRange
}

func (r *lineRange) single(lineNumber int) {
	r.start = lineNumber
	r.end = lineNumber
	r.disasm.Clear()
}

func (r lineRange) isSingle() bool {
	return r.start == r.end
}

func (r lineRange) inRange(l int) bool {
	if r.end < r.start {
		return l >= r.end && l <= r.start
	}
	return l >= r.start && l <= r.end
}

func (r lineRange) ordered() (int, int) {
	if r.start < r.end {
		return r.start, r.end
	}
	return r.end, r.start
}

type winCoProcSource struct {
	debuggerWin

	img *SdlImgui

	open               bool
	showTooltip        bool
	syntaxHighlighting bool
	optionsHeight      float32

	lineFuzzy fuzzyFilter

	selectedLine lineRange
	selecting    bool

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
	widthIcon  float32
	widthStats float32
	widthLine  float32

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
	win.widthIcon = imgui.CalcTextSize(fmt.Sprintf("%c", fonts.Chip), true, 0).X
	win.widthStats = imgui.CalcTextSize("00.0% ", true, 0).X
	win.widthLine = imgui.CalcTextSize("9999 ", true, 0).X
}

func (win *winCoProcSource) id() string {
	return winCoProcSourceID
}

const sourcePopupID = "sourcePopupID"

func (win *winCoProcSource) debuggerDraw() {
	if !win.img.lz.Cart.HasCoProcBus {
		return
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
		return
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
					win.selectedLine.single(focusLine.LineNumber)
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

		// change selectedFile
		if win.updateSelectedFile {
			win.selectedFile = src.FilesByShortname[win.selectedShortFileName]
			// updateSelectFile is reset to false below (because we need to check it again)
		}

		// source code view
		imgui.BeginGroup()
		win.drawSource(src)
		imgui.EndGroup()

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
			win.selectedLine.single(0)
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
	win.selectedLine.single(ln.LineNumber)
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
		imgui.TableSetupColumnV("##selection", imgui.TableColumnFlagsNone, 0.0, 0)

		// next three columns have fixed width
		imgui.TableSetupColumnV("##icon", imgui.TableColumnFlagsNone, win.widthIcon*1.5, 1)
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
					if win.selectedLine.inRange(ln.LineNumber) {
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

					// select source lines with mouse click and drag
					if imgui.IsItemHovered() {

						// add/remove lines before showing the tooltip. this
						// produces better visual results
						if imgui.IsMouseClicked(0) {
							win.selectedLine.single(ln.LineNumber)
							win.selecting = true
						}
						if imgui.IsMouseDragging(0, 0.0) && win.selecting {
							win.selectedLine.end = ln.LineNumber
							win.selectedLine.disasm.Clear()
							s, e := win.selectedLine.ordered()
							for i := s; i <= e; i++ {
								win.selectedLine.disasm.Add(win.selectedFile.Content.Lines[i-1])
							}
						}
						if imgui.IsMouseReleased(0) {
							win.selecting = false
						}

						// asm tooltip
						multiline := !win.selectedLine.isSingle() && win.selectedLine.inRange(ln.LineNumber)
						if win.showTooltip {
							// how we show the asm depends on whether there are
							// multiple lines selected and whether there is any
							// diassembly for those lines.
							//
							// if only a single line is selected then we simply
							// check that there is are asm entries for that line
							//
							// there is also condition to test whether the
							// multiline selection is in progress (win.selecting).
							// this is to prevent a frame's worth of flicker caused
							// by time difference of the mouse running out of range
							// of the multiline and the new range being setup (see
							// IsMouseDragging() below)
							if (!multiline && len(ln.Disassembly) > 0) ||
								((multiline || win.selecting) && !win.selectedLine.disasm.IsEmpty()) {

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
									if (multiline || win.selecting) && !win.selectedLine.isSingle() {
										s, e := win.selectedLine.ordered()
										win.img.drawFilenameAndLineNumber(ln.File.Filename, s, e)
									} else {
										win.img.drawFilenameAndLineNumber(ln.File.Filename, ln.LineNumber, -1)
									}

									imgui.Spacing()
									imgui.Separator()
									imgui.Spacing()

									// choose which disasm list to use
									disasm := ln.Disassembly
									if multiline || win.selecting {
										disasm = win.selectedLine.disasm.Disasm
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
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
						imgui.Text(string(fonts.Breakpoint))
						imgui.PopStyleColor()
					} else if ln.Bug {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceBug)
						imgui.Text(string(fonts.CoProcBug))
						imgui.PopStyleColor()
					} else if len(ln.Disassembly) > 0 {
						if ln.Function.IsInlined() {
							imgui.Text(string(fonts.Inlined))
						} else {
							imgui.Text(string(fonts.Chip))
						}
					}

					// performance statistics
					imgui.TableNextColumn()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
					if ln.Stats.Overall.HasExecuted() {
						if ln.Stats.Overall.OverSource.FrameValid {
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Frame))
						} else if ln.Stats.Overall.OverSource.AverageValid {
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceAvgLoad)
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Average))
							imgui.PopStyleColor()
						} else if ln.Stats.Overall.OverSource.MaxValid {
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceMaxLoad)
							imgui.Text(fmt.Sprintf("%.02f", ln.Stats.Overall.OverSource.Max))
							imgui.PopStyleColor()
						} else {
							imgui.Text(" -")
						}
					} else if len(ln.Disassembly) > 0 {
						// line has never been executed
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

					if win.syntaxHighlighting {
						win.img.drawSourceLine(ln, false)
					} else {
						imgui.Text(ln.PlainContent)
					}
				}
			}

			// scroll to correct line
			if win.updateSelectedFile {
				imgui.SetScrollY(clipper.ItemsHeight * float32(win.selectedLine.start-10))

				// we can reset updateSelectedFile here (because we don't need it again)
				win.updateSelectedFile = false
			}
		}

		imgui.EndTable()
	}

	imgui.PopStyleVarV(2)
	imgui.PopFont()

	imgui.EndChild()
}
