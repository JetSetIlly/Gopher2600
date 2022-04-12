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
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcGlobalsID = "Coprocessor Global Variables"
const winCoProcGlobalsMenu = "Globals"

type winCoProcGlobals struct {
	img  *SdlImgui
	open bool

	firstOpen bool

	selectedFile          *developer.SourceFile
	selectedFileComboOpen bool
}

func newWinCoProcGlobals(img *SdlImgui) (window, error) {
	win := &winCoProcGlobals{
		img:       img,
		firstOpen: true,
	}
	return win, nil
}

func (win *winCoProcGlobals) init() {
}

func (win *winCoProcGlobals) id() string {
	return winCoProcGlobalsID
}

func (win *winCoProcGlobals) isOpen() bool {
	return win.open
}

func (win *winCoProcGlobals) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcGlobals) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{551, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcGlobalsID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.Filenames) == 0 {
			imgui.Text("No source files available")
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

		imgui.AlignTextToFramePadding()
		imgui.Text("Filename")
		imgui.SameLine()
		imgui.PushItemWidth(imgui.ContentRegionAvail().X)
		if imgui.BeginComboV("##selectedFile", win.selectedFile.ShortFilename, imgui.ComboFlagsHeightRegular) {
			for _, fn := range src.Filenames {
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

		// global variable table for the selected file

		const numColumns = 4

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsNoHostExtendX
		flgs |= imgui.TableFlagsResizable

		imgui.BeginTableV("##globalsTable", numColumns, flgs, imgui.Vec2{}, 0.0)

		// setup columns. the labelling column 2 depends on whether the coprocessor
		// development instance has source available to it
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsNone, width*0.40, 0)
		imgui.TableSetupColumnV("Type", imgui.TableColumnFlagsNone, width*0.20, 1)
		imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.15, 2)
		imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNone, width*0.20, 3)

		imgui.Spacing()
		imgui.TableHeadersRow()

		for _, name := range win.selectedFile.GlobalNames {
			varb := win.selectedFile.Globals[name]

			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesName)
			imgui.Text(varb.Name)
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
			imgui.Text(varb.Type)
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
			imgui.Text(fmt.Sprintf("%08x", varb.Address))
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.Text(win.readMemory(varb.Address))
		}

		imgui.EndTable()
	})
}

func (win *winCoProcGlobals) readMemory(address uint64) string {
	if !win.img.lz.Cart.HasStaticBus {
		return "-"
	}

	v, ok := win.img.lz.Cart.Static.Read32bit(uint32(address))
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%08x", v)
}
