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

	scrollTo     int
	scrollToFile string
	selectedLine int

	selectedFile          *developer.SourceFile
	selectedFileComboOpen bool
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

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcSourceID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	// safely iterate over source code
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if win.scrollTo > 0 && (win.selectedFile == nil || win.scrollToFile != win.selectedFile.Filename) {
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
		lineSpacing := float32(win.img.prefs.codeFontLineSpacing.Get().(int))
		imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, imgui.Vec2{X: 4, Y: lineSpacing})

		imgui.BeginChildV("##coprocSource", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)
		imgui.BeginTableV("##coprocSourceTable", 5, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0.0)

		// first column is a dummy column so that Selectable (span all columns) works correctly
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 0)
		imgui.TableSetupColumnV("Icon", imgui.TableColumnFlagsNone, width*0.04, 1)
		imgui.TableSetupColumnV("Load", imgui.TableColumnFlagsNone, width*0.07, 2)
		imgui.TableSetupColumnV("Number", imgui.TableColumnFlagsNone, width*0.04, 3)
		imgui.TableSetupColumnV("Source", imgui.TableColumnFlagsNone, width*0.85, 4)

		var clipper imgui.ListClipper
		clipper.Begin(len(win.selectedFile.Lines))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				if i >= len(win.selectedFile.Lines) {
					break
				}

				ln := win.selectedFile.Lines[i]
				imgui.TableNextRow()

				// scroll to correct line
				if win.scrollTo > 0 {
					y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
					y = float32(win.selectedLine-10) * y
					imgui.SetScrollY(y)
				}

				// highlight selected line
				if ln.LineNumber == win.selectedLine {
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.CoProcSourceSelected)
				}

				// highlight line mouse is over
				imgui.TableNextColumn()
				imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
				imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
				imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
				imgui.PopStyleColorV(2)

				// show chip icon and also tooltip if mouse is hovered
				// on selectable
				imgui.TableNextColumn()
				if len(ln.Disassembly) > 0 {
					if win.showAsm {
						imguiTooltip(func() {
							imgui.Text(ln.File.ShortFilename)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
							imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
							imgui.PopStyleColor()
							imgui.Spacing()
							imgui.Separator()
							imgui.Spacing()
							for _, asm := range ln.Disassembly {
								imgui.Text(asm.String())
							}
						}, true)
					}

					if ln.IllegalCount > 0 {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceBug)
						imgui.Text(string(fonts.CoProcBug))
						imgui.PopStyleColor()
					} else {
						imgui.Text(string(fonts.Chip))
					}
				}

				// percentage of time taken by this line
				imgui.TableNextColumn()
				if ln.FrameCycles > 0 {
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLoad)
					imgui.Text(fmt.Sprintf("%0.2f%%", ln.FrameCycles/src.FrameCycles*100.0))
					imgui.PopStyleColor()
				}

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

		imgui.EndTable()
		imgui.EndChild()

		imgui.PopStyleVar()
		imgui.PopFont()

		imgui.EndChild()

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()
			imgui.Checkbox("Show ASM in Tooltip", &win.showAsm)
		})
	})

	if win.scrollTo > 0 {
		win.scrollTo--
	}
}

func (win *winCoProcSource) gotoSource(ln *developer.SourceLine) {
	win.setOpen(true)
	win.scrollTo = 10
	win.scrollToFile = ln.File.Filename
	win.selectedLine = ln.LineNumber
}
