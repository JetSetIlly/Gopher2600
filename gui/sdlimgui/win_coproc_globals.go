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
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

const winCoProcGlobalsID = "Coprocessor Global Variables"
const winCoProcGlobalsMenu = "Globals"

type winCoProcGlobals struct {
	debuggerWin

	img *SdlImgui

	firstOpen bool

	selectedFileFuzzy     fuzzyFilter
	selectedShortFileName string
	selectedFile          *dwarf.SourceFile
	updateSelectedFile    bool

	optionsHeight     float32
	showAllGlobals    bool
	showLocatableOnly bool

	openNodes map[string]bool
}

func newWinCoProcGlobals(img *SdlImgui) (window, error) {
	win := &winCoProcGlobals{
		img:            img,
		firstOpen:      true,
		showAllGlobals: true,
		openNodes:      make(map[string]bool),
	}
	return win, nil
}

func (win *winCoProcGlobals) init() {
}

func (win *winCoProcGlobals) id() string {
	return winCoProcGlobalsID
}

const globalsPopupID = "globalsPopupID"

func (win *winCoProcGlobals) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{700, 1000})

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcGlobalsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcGlobals) drawFileSelection(src *dwarf.Source) {
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
			win.updateSelectedFile = true
		}

		if !win.selectedFileFuzzy.draw("##selectedFileFuzzy", src.ShortFilenames, fuzzyFileHook, true) {
			imgui.CloseCurrentPopup()
		}

		imgui.PopItemWidth()
		imgui.EndPopup()
	}
}

func (win *winCoProcGlobals) draw() {
	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.Filenames) == 0 {
			imgui.Text("No source files available")
			return
		}

		if src.SortedGlobals.Len() == 0 {
			imgui.Text("No global variable in the source")
			return
		}

		// update all global variables on every frame
		win.img.dbg.PushFunction(src.UpdateGlobalVariables)

		if win.firstOpen {
			// assume source entry point is a function called "main"
			if m, ok := src.Functions["main"]; ok {
				win.selectedFile = m.DeclLine.File
				win.selectedShortFileName = win.selectedFile.ShortFilename
			} else {
				// if main does not exists then open at the first file in the list
				for _, fn := range src.Filenames {
					if src.Files[fn].HasGlobals {
						win.selectedFile = src.Files[fn]
						win.selectedShortFileName = win.selectedFile.ShortFilename
						break // for loop
					}
				}
			}

			win.firstOpen = false
		}

		if !win.showAllGlobals {
			win.drawFileSelection(src)
			imgui.Separator()
		}

		// change selectedFile
		if win.updateSelectedFile {
			win.selectedFile = src.FilesByShortname[win.selectedShortFileName]
		}

		// global variable table for the selected file

		const numColumns = 4

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsSortable
		flgs |= imgui.TableFlagsResizable

		if imgui.BeginTableV("##globalsTable", numColumns, flgs, imgui.Vec2{Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0) {

			// setup columns. the labelling column 2 depends on whether the coprocessor
			// development instance has source available to it
			width := imgui.ContentRegionAvail().X
			imgui.TableSetupColumnV("Name", imgui.TableColumnFlagsPreferSortDescending|imgui.TableColumnFlagsDefaultSort, width*0.40, 0)
			imgui.TableSetupColumnV("Type", imgui.TableColumnFlagsNoSort, width*0.20, 1)
			imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsPreferSortDescending, width*0.15, 2)
			imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNoSort, width*0.20, 3)

			imgui.TableSetupScrollFreeze(0, 1)
			imgui.TableHeadersRow()

			for i, varb := range src.SortedGlobals.Variables {
				if win.showAllGlobals || varb.DeclLine.File.Filename == win.selectedFile.Filename {
					win.drawVariable(src, varb, 0, false, fmt.Sprint(i))
				}
			}

			sort := imgui.TableGetSortSpecs()
			if sort.SpecsDirty() {
				for _, s := range sort.Specs() {
					switch s.ColumnUserID {
					case 0:
						src.SortedGlobals.SortByName(s.SortDirection == imgui.SortDirectionAscending)
					case 2:
						src.SortedGlobals.SortByAddress(s.SortDirection == imgui.SortDirectionAscending)
					}
				}
				sort.ClearSpecsDirty()
			}

			imgui.EndTable()

			if imgui.IsMouseDown(1) && imgui.IsItemHovered() {
				imgui.OpenPopup(globalsPopupID)
			}

			win.optionsHeight = imguiMeasureHeight(func() {
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.Checkbox("List all globals (in all files)", &win.showAllGlobals)

				imgui.SameLineV(0, 15)
				imgui.Checkbox("Don't show unlocatable variables", &win.showLocatableOnly)
				win.img.imguiTooltipSimple(`A unlocatable variable is a variable has been
removed by the compiler's optimisation process`)
			})

			if imgui.BeginPopup(globalsPopupID) {
				if imgui.Selectable(fmt.Sprintf("%c Save Globals to CSV", fonts.Disk)) {
					win.saveToCSV(src)
				}
				imgui.EndPopup()
			}
		}
	})
}

func drawVariableTooltipShort(varb *dwarf.SourceVariable, cols *imguiColors) {
	imgui.Text(varb.DeclLine.File.ShortFilename)
	imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcSourceLineNumber)
	imgui.Text(fmt.Sprintf("Line: %d", varb.DeclLine.LineNumber))
	imgui.PopStyleColor()

	if varb.Error != nil {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(string(fonts.CoProcBug))
		imgui.SameLine()
		imgui.Text("Error on Resolve")
		for _, l := range strings.Split(varb.Error.Error(), ":") {
			imgui.Text("·")
			imgui.SameLine()
			imgui.Text(strings.TrimSpace(l))
		}
	}
}

func drawVariableTooltip(varb *dwarf.SourceVariable, value uint32, cols *imguiColors) {
	if a, ok := varb.Address(); ok {
		imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesAddress)
		imgui.Text(fmt.Sprintf("%08x", a))
		imgui.PopStyleColor()
	}

	imgui.Text(varb.Name)
	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesType)
	imgui.Text(varb.Type.Name)
	imgui.PopStyleColor()

	imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesTypeSize)
	imgui.Text(fmt.Sprintf("%d bytes", varb.Type.Size))
	imgui.PopStyleColor()

	if varb.Type.IsArray() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(fmt.Sprintf("is an array of %d elements", varb.Type.ElementCount))
	} else if varb.Type.IsComposite() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(fmt.Sprintf("is a struct of %d members", len(varb.Type.Members)))
	} else {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		if imgui.BeginTableV("##variablevalues", 2, imgui.TableFlagsNone, imgui.Vec2{}, 0.0) {
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.Text("Dec: ")
			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesNotes)
			imgui.Text(fmt.Sprintf("%d", value))
			imgui.PopStyleColor()

			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.Text("Hex: ")
			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesNotes)
			hex := fmt.Sprintf(varb.Type.Hex(), value)
			imgui.Text(hex[:2])

			for i := 1; i < len(hex)/2; i++ {
				imgui.SameLine()
				s := i * 2
				imgui.Text(fmt.Sprintf("%s", hex[s:s+2]))
			}

			imgui.PopStyleColor()

			// binary information is a little more complex to draw. we split
			// the binary value into bytes and display vertically
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.Text("Bin: ")

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesNotes)
			bin := fmt.Sprintf(varb.Type.Bin(), value)
			imgui.Text(bin[:8])

			for i := 1; i < len(bin)/8; i++ {
				s := i * 8
				imgui.Text(bin[s : s+8])
			}

			imgui.PopStyleColor()
		}
		imgui.EndTable()

		if varb.Type.Conversion != nil {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.Text("Converted Value: ")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcVariablesNotes)
			imgui.Text(fmt.Sprintf(varb.Type.Conversion(varb.Value())))
			imgui.PopStyleColor()
		}

	}

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	imgui.Text(varb.DeclLine.File.ShortFilename)
	imgui.PushStyleColor(imgui.StyleColorText, cols.CoProcSourceLineNumber)
	imgui.Text(fmt.Sprintf("Line: %d", varb.DeclLine.LineNumber))
	imgui.PopStyleColor()

	if varb.Error != nil {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(string(fonts.CoProcBug))
		imgui.SameLine()
		imgui.Text(varb.Error.Error())
	}
}

func (win *winCoProcGlobals) drawVariable(src *dwarf.Source, varb *dwarf.SourceVariable,
	indentLevel int, unnamed bool, nodeID string) {

	const IndentDepth = 2

	var name string
	if unnamed {
		name = fmt.Sprintf("%s%s", strings.Repeat(" ", IndentDepth*indentLevel), string(fonts.Pointer))
	} else {
		name = fmt.Sprintf("%s%s", strings.Repeat(" ", IndentDepth*indentLevel), varb.Name)
	}

	if !varb.IsValid() {
		if win.showLocatableOnly {
			return
		}
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesNotVisible)
		defer imgui.PopStyleColor()
	}

	imgui.TableNextRow()
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
	imgui.SelectableV(name, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	if !varb.IsValid() {
		imgui.TableNextColumn()
		imgui.Text(varb.Type.Name)
		imgui.TableNextColumn()
		imgui.Text("not locatable")
		return
	}

	if varb.NumChildren() > 0 {
		// we could show a tooltip for variables with children but this needs
		// work. for instance, how do we illustrate a composite type or an
		// array?
		win.img.imguiTooltip(func() {
			drawVariableTooltipShort(varb, win.img.cols)
		}, true)

		if imgui.IsItemClicked() {
			win.openNodes[nodeID] = !win.openNodes[nodeID]
		}

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
		if a, ok := varb.Address(); ok {
			imgui.Text(fmt.Sprintf("%08x", a))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		// value column shows tree open/close icon unless there was an error
		// during variable resolution
		imgui.TableNextColumn()
		if varb.Error != nil {
			imgui.Text(string(fonts.CoProcBug))
		} else {
			if win.openNodes[nodeID] {
				imgui.Text(string(fonts.TreeOpen))
			} else {
				imgui.Text(string(fonts.TreeClosed))
			}

			if win.openNodes[nodeID] {
				for i := 0; i < varb.NumChildren(); i++ {
					win.drawVariable(src, varb.Child(i), indentLevel+1, false, fmt.Sprint(nodeID, i))
				}
			}
		}
	} else {
		win.img.imguiTooltip(func() {
			if varb.Error != nil {
				drawVariableTooltipShort(varb, win.img.cols)
			} else {
				drawVariableTooltip(varb, varb.Value(), win.img.cols)
			}
		}, true)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesType)
		imgui.Text(varb.Type.Name)
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
		if a, ok := varb.Address(); ok {
			imgui.Text(fmt.Sprintf("%08x", a))
		} else {
			imgui.Text("-")
		}
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if varb.Error != nil {
			imgui.Text(string(fonts.CoProcBug))
		} else {
			imgui.Text(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
		}
	}
}

// save all variables in the curent view to a CSV file in the working
// directory. filename will be of the form:
//
// globals_<cart name>_<timestamp>.csv
//
// all entries in the current view are saved, including closed nodes.
func (win *winCoProcGlobals) saveToCSV(src *dwarf.Source) {
	// open unique file
	fn := unique.Filename("globals", win.img.cache.VCS.Mem.Cart.ShortName)
	fn = fmt.Sprintf("%s.csv", fn)
	f, err := os.Create(fn)
	if err != nil {
		logger.Logf("sdlimgui", "could not save globals CSV: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf("sdlimgui", "error saving globals CSV: %v", err)
		}
	}()

	// write variable to file
	writeVarb := func(varb *dwarf.SourceVariable) {
		f.WriteString(fmt.Sprintf("%s,", varb.Name))
		f.WriteString(fmt.Sprintf("%s,", varb.Type.Name))
		if a, ok := varb.Address(); ok {
			f.WriteString(fmt.Sprintf("%08x,", a))
		} else {
			f.WriteString(",")
		}

		f.WriteString(fmt.Sprintf(varb.Type.Hex(), varb.Value()))
		f.WriteString("\n")
	}

	// the builEntry function is recursive and will is very similar in
	// structure to the drawVariable() function above
	var buildEntry func(*dwarf.SourceVariable, string)
	buildEntry = func(varb *dwarf.SourceVariable, parent string) {
		f.WriteString(fmt.Sprintf("%s,", parent))

		// how we write the line differs depending on whether the variable has
		// children or not
		if varb.NumChildren() > 0 {
			if parent != "" {
				parent = fmt.Sprintf("%s->%s", parent, varb.Name)
			} else {
				parent = varb.Name
				writeVarb(varb)
			}

			for i := 0; i < varb.NumChildren(); i++ {
				buildEntry(varb.Child(i), parent)
			}
		} else {
			writeVarb(varb)
		}
	}

	// write header to CSV file
	f.WriteString("Parent, Name, Type, Address, Value\n")

	// process every variable in the current view
	for _, varb := range src.SortedGlobals.Variables {
		buildEntry(varb, "")
	}
}
