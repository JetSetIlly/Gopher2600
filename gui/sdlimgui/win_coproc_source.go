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

	// the first time the source window is opened
	firstOpen bool

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

	// widths of columns in the disasm table
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
		firstOpen:          true,
	}
	return win, nil
}

func (win *winCoProcSource) init() {
	win.widthIcon = imgui.CalcTextSize(fmt.Sprintf("%c ", fonts.Chip), true, 0).X
	win.widthStats = imgui.CalcTextSize("00.0% ", true, 0).X
	win.widthLine = imgui.CalcTextSize("9999 ", true, 0).X
}

func (win *winCoProcSource) id() string {
	return winCoProcSourceID
}

func (win *winCoProcSource) debuggerDraw() {
	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	// check yield state and open the window if necessary
	win.img.dbg.CoProcDev.BorrowYieldState(func(yield *developer.YieldState) {
		if yield.TimeStamp != win.yieldState.TimeStamp {
			win.yieldState = *yield

			win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
				// we need a check for src validity here. when we borrow the
				// source again later we check it for a second time
				if src == nil {
					return
				}
				win.yieldLine = src.FindSourceLine(win.yieldState.InstructionPC)
			})

			// open window and focus on yield line if the yield is a breakpoint
			if yield.Reason != mapper.YieldSyncWithVCS {
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

	var flgs imgui.WindowFlags
	if win.uncollapseNext && win.isCollapsed {
		flgs = imgui.WindowFlagsNoCollapse
	} else {
		flgs = imgui.WindowFlagsNone
	}
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

		// focuse on main function if this is the first time the window has
		// been opened.
		if win.firstOpen {
			win.focusYieldLine = true

			// indicate that we've handled firstOpen only if state is paused.
			// this is a rough-and-ready solution to the problem of breaking
			// into the debugger from playmode. without this condition, the
			// focusYieldLine is lost in the transition
			if win.img.dbg.State() == govern.Paused {
				win.firstOpen = false
			}
		}

		// focus on yield line (or main function if we don't have a yield line)
		// but only if emulation is paused
		if win.focusYieldLine {
			if win.img.dbg.State() == govern.Paused {
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
			}

			// focus has been dealt with
			win.focusYieldLine = false
		}

		// change selectedFile
		if win.updateSelectedFile {
			win.selectedFile = src.FilesByShortname[win.selectedShortFileName]
			// updateSelectFile is reset to false below (because we need to check it again)
		}

		// final check before continuing. if selectedFile is nil then exit with
		// the no source message
		if win.selectedFile == nil {
			imgui.Text("No source files available")
			return
		}

		// fuzzy file selector
		win.drawFileSelection(src)
		imgui.Separator()

		// source code view
		win.drawSource(src)

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			win.drawLineSearch()
			imgui.SameLineV(0, 10)

			if imgui.Button(fmt.Sprintf("%c Focus Yield Line", fonts.DisasmGotoCurrent)) {
				win.focusYieldLine = true
			}
			imgui.SameLineV(0, 20)
			imgui.Checkbox("Highlight Comments & String Literals", &win.syntaxHighlighting)
			imgui.SameLineV(0, 20)
			imgui.Checkbox("Show Tooltip", &win.showTooltip)
			imgui.SameLineV(0, 20)
			if imgui.Button(fmt.Sprintf("%c Save to CSV", fonts.Disk)) {
				win.saveToCSV(src)
			}
		})
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
	imgui.Text(win.selectedShortFileName)

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

func (win *winCoProcSource) drawLineSearch() {
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
			win.gotoSourceLine(win.selectedFile.Content.Lines[i])
		}

		if !win.lineFuzzy.draw("##linefuzzy", win.selectedFile.Content, lineFuzzyHook, false) {
			imgui.CloseCurrentPopup()
		}

		imgui.PopItemWidth()
		imgui.EndPopup()
	}
}

func (win *winCoProcSource) gotoSourceLine(ln *developer.SourceLine) {
	win.debuggerSetOpen(true)
	win.selectedShortFileName = ln.File.ShortFilename
	win.selectedLine.single(ln.LineNumber)
	win.uncollapseNext = true
	win.updateSelectedFile = true

	// force firstOpen to false. sometimes gotoSourceLine() is called before
	// the source window is ever opened. in these instances the firstOpen
	// procedure will run and supercede the values set above
	win.firstOpen = false
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

	const numColumns = 4
	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsNoHostExtendX
	if imgui.BeginTableV("##coprocSourceTable", numColumns, flgs, imgui.Vec2{}, 0.0) {
		// first column is a dummy column so that Selectable (span all columns) works correctly
		imgui.TableSetupColumnV("Icon", imgui.TableColumnFlagsNone, win.widthIcon, 0)
		imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNone, win.widthStats, 1)
		imgui.TableSetupColumnV("LineNumber", imgui.TableColumnFlagsNone, win.widthLine, 2)

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

				// show appropriate icon in the gutter
				if len(ln.Disassembly) > 0 {
					if src.CheckBreakpointBySourceLine(ln) {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
						imgui.SelectableV(string(fonts.Breakpoint), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
						imgui.PopStyleColor()
					} else if ln.Bug {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceBug)
						imgui.SelectableV(string(fonts.CoProcBug), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
						imgui.PopStyleColor()
					} else {
						imgui.SelectableV(string(fonts.Chip), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
					}

					// allow breakpoint toggle for lines with executable entries
					if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
						src.ToggleBreakpoint(ln)
					}
				} else {
					imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
				}
				imgui.PopStyleColorV(2)

				// select source lines with mouse click and drag
				if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenBlockedByActiveItem) {

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
							}, false)

							imgui.PushFont(win.img.glsl.fonts.code)
						}
					}

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

		imgui.EndTable()
	}

	imgui.PopStyleVarV(2)
	imgui.PopFont()

	imgui.EndChild()
}
