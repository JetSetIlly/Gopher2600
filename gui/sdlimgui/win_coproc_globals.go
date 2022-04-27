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

const winCoProcGlobalsID = "Coprocessor Global Variables"
const winCoProcGlobalsMenu = "Globals"

type winCoProcGlobals struct {
	debuggerWin

	img *SdlImgui

	firstOpen bool

	selectedFile          *developer.SourceFile
	selectedFileComboOpen bool

	optionsHeight  float32
	showAllGlobals bool

	openNodes map[string]bool
}

func newWinCoProcGlobals(img *SdlImgui) (window, error) {
	win := &winCoProcGlobals{
		img:       img,
		firstOpen: true,
		openNodes: make(map[string]bool),
	}
	return win, nil
}

func (win *winCoProcGlobals) init() {
}

func (win *winCoProcGlobals) id() string {
	return winCoProcGlobalsID
}

func (win *winCoProcGlobals) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{700, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcGlobalsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	imgui.End()
}

func (win *winCoProcGlobals) draw() {
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.Filenames) == 0 {
			imgui.Text("No source files available")
			return
		}

		thereAreGlobals := false
		for _, fn := range src.Filenames {
			if len(src.Files[fn].GlobalNames) > 0 {
				thereAreGlobals = true
			}
		}
		if !thereAreGlobals {
			imgui.Text("No global variable in the source")
			return
		}

		if win.firstOpen {
			// assume source entry point is a function called "main"
			if m, ok := src.Functions["main"]; ok {
				win.selectedFile = m.DeclLine.File
			} else {
				imgui.Text("Can't find main() function")
				return
			}

			win.firstOpen = false
		}

		if !win.showAllGlobals {
			imgui.AlignTextToFramePadding()
			imgui.Text("Filename")
			imgui.SameLine()
			imgui.PushItemWidth(imgui.ContentRegionAvail().X)
			if imgui.BeginComboV("##selectedFile", win.selectedFile.ShortFilename, imgui.ComboFlagsHeightRegular) {
				for _, fn := range src.Filenames {
					// skip files that have no global variables
					if len(src.Files[fn].GlobalNames) == 0 {
						continue
					}

					if imgui.Selectable(src.Files[fn].ShortFilename) {
						win.selectedFile = src.Files[fn]
					}

					// set scroll on the first frame that the combo is open
					if !win.selectedFileComboOpen && fn == win.selectedFile.Filename {
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
		}

		// global variable table for the selected file

		const numColumns = 4

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsNoHostExtendX
		flgs |= imgui.TableFlagsResizable

		imgui.BeginTableV("##globalsTable", numColumns, flgs, imgui.Vec2{Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0)

		// setup columns. the labelling column 2 depends on whether the coprocessor
		// development instance has source available to it
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsNone, width*0.40, 0)
		imgui.TableSetupColumnV("Type", imgui.TableColumnFlagsNone, width*0.20, 1)
		imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.15, 2)
		imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNone, width*0.20, 3)

		imgui.TableSetupScrollFreeze(0, 1)
		imgui.TableHeadersRow()

		// the global list depends on the state fo the showAllGlobals state
		var globalList []string
		if win.showAllGlobals {
			globalList = src.GlobalNames
		} else {
			globalList = win.selectedFile.GlobalNames
		}

		for _, name := range globalList {
			// get variable from the correct globals list
			var varb *developer.SourceVariable
			if win.showAllGlobals {
				varb = src.Globals[name]
			} else {
				varb = win.selectedFile.Globals[name]
			}

			win.drawVariable(src, varb, 0, 0)
		}

		imgui.EndTable()

		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.Checkbox("List all globals (in all files)", &win.showAllGlobals)
		})
	})
}

func (win *winCoProcGlobals) drawVariable(src *developer.Source,
	varb *developer.SourceVariable, baseAddress uint64,
	indentLevel int) {

	address := varb.Address
	if varb.AddressIsOffset() {
		// address of variable is an offset of parent address
		address += baseAddress
	}

	const IndentDepth = 2
	name := fmt.Sprintf("%s%s", strings.Repeat(" ", IndentDepth*indentLevel), varb.Name)

	if varb.IsComposite() || varb.IsArray() {
		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV(name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		imguiTooltip(func() {
			imgui.Text(varb.Name)
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
			imgui.Text(varb.Type.Name)
			imgui.PopStyleColor()

			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesTypeSize)
			imgui.Text(fmt.Sprintf("%d bytes", varb.Type.Size))
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			imgui.Text(varb.DeclLine.File.ShortFilename)
			imgui.Text(fmt.Sprintf("Line: %d", varb.DeclLine.LineNumber))
		}, true)

		nodeID := fmt.Sprintf("%s_%x", varb.Name, baseAddress+varb.Address)

		if imgui.IsItemClicked() {
			win.openNodes[nodeID] = !win.openNodes[nodeID]
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
		imgui.Text(fmt.Sprintf("%08x", address))
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if win.openNodes[nodeID] {
			imgui.Text(string(fonts.TreeOpen))
		} else {
			imgui.Text(string(fonts.TreeClosed))
		}

		if win.openNodes[nodeID] {
			if varb.IsComposite() {
				for _, memb := range varb.Type.Members {
					win.drawVariable(src, memb, address, indentLevel+1)
				}
			} else if varb.IsArray() {
				for i := 0; i < varb.Type.ElementCount; i++ {
					elem := &developer.SourceVariable{
						Name:     fmt.Sprintf("%s[%d]", varb.Name, i),
						Type:     varb.Type.BaseType,
						DeclLine: varb.DeclLine,
						Address:  address + uint64(i*varb.Type.BaseType.Size),
					}
					win.drawVariable(src, elem, elem.Address, indentLevel+1)
				}
			}
		}

	} else {
		value, valueOk := win.readMemory(address)
		value &= varb.Type.Mask()

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
		imgui.SelectableV(name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		if valueOk {
			imguiTooltip(func() {
				imgui.Text(varb.Name)
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
				imgui.Text(varb.Type.Name)
				imgui.PopStyleColor()

				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesTypeSize)
				imgui.Text(fmt.Sprintf("%d bytes", varb.Type.Size))
				imgui.PopStyleColor()

				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

				imgui.Text("Hex: ")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesNotes)
				imgui.Text(fmt.Sprintf(varb.Type.Hex(), value))
				imgui.PopStyleColor()

				imgui.Text("Dec: ")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesNotes)
				imgui.Text(fmt.Sprintf("%d", value))
				imgui.PopStyleColor()

				imgui.Text("Bin: ")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesNotes)
				imgui.Text(fmt.Sprintf(varb.Type.Bin(), value))
				imgui.PopStyleColor()

				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

				imgui.Text(varb.DeclLine.File.ShortFilename)
				imgui.Text(fmt.Sprintf("Line: %d", varb.DeclLine.LineNumber))
			}, true)
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
		imgui.Text(fmt.Sprintf("%08x", address))
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if valueOk {
			imgui.Text(fmt.Sprintf(varb.Type.Hex(), value))
		} else {
			imgui.Text("-")
		}
	}
}

func (win *winCoProcGlobals) readMemory(address uint64) (uint32, bool) {
	if !win.img.lz.Cart.HasStaticBus {
		return 0, false
	}
	return win.img.lz.Cart.Static.Read32bit(uint32(address))
}
