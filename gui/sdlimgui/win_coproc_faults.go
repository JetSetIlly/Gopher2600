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
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

const winCoProcFaultsID = "Memory Faults"
const winCoProcFaultsMenu = "Memory Faults"

type winCoProcFaults struct {
	debuggerWin

	img *SdlImgui

	showSrcInTooltip bool
	optionsHeight    float32
}

func newWinCoProcFaults(img *SdlImgui) (window, error) {
	win := &winCoProcFaults{
		img:              img,
		showSrcInTooltip: true,
	}
	return win, nil
}

func (win *winCoProcFaults) init() {
}

func (win *winCoProcFaults) id() string {
	return winCoProcFaultsID
}

func (win *winCoProcFaults) debuggerDraw() bool {
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
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcFaultsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.img.dbg.CoProcDev.BorrowFaults(func(flt faults.Faults) {
			win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
				win.draw(flt, src)
			})
		})
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcFaults) draw(flt faults.Faults, src *dwarf.Source) {
	// hasStackCollision to decide whether to issue warning in footer
	hasStackCollision := false

	if len(flt.Log) == 0 {
		imgui.Text("No memory faults")
		return
	}

	// note HasStackCollision for later comparison
	hasStackCollision = flt.HasStackCollision

	const numColumns = 4

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingStretchSame
	flgs |= imgui.TableFlagsResizable

	imgui.BeginTableV("##coprocFaultsTable", numColumns, flgs, imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0)

	// setup columns. the labelling column 2 depends on whether the coprocessor
	// development instance has source available to it
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("Category", imgui.TableColumnFlagsNone, width*0.25, 0)
	imgui.TableSetupColumnV("Event", imgui.TableColumnFlagsNone, width*0.25, 1)
	imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.15, 2)
	if src == nil {
		imgui.TableSetupColumnV("Instruction Address", imgui.TableColumnFlagsNone, width*0.35, 3)
	} else {
		imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNone, width*0.35, 3)
	}

	imgui.Spacing()
	imgui.TableHeadersRow()

	for i := 0; i < len(flt.Log); i++ {
		e := flt.Log[i]

		var ln *dwarf.SourceLine
		if src != nil {
			ln = src.SourceLineByAddr(e.InstructionAddr)
		}

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHoverLine)
		imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHoverLine)
		imgui.SelectableV(string(e.Category), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColorV(2)

		// source on tooltip
		win.img.imguiTooltip(func() {
			imgui.Text("Instruction Address:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcFaultsAddress)
			imgui.Text(fmt.Sprintf("%08x", e.InstructionAddr))
			imgui.PopStyleColor()

			if e.Category == faults.NullDereference {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcFaultsNotes)
				imgui.Text("likely null pointer dereference")
				imgui.PopStyleColor()
			}

			imgui.Text("Frequency:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcFaultsFrequency)
			imgui.Text(fmt.Sprintf("%d", e.Count))
			imgui.PopStyleColor()

			if win.showSrcInTooltip && ln != nil {
				if !ln.IsStub() {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()

					win.img.drawFilenameAndLineNumber(ln.File.Filename, ln.LineNumber, -1)

					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					win.img.drawSourceLine(ln, true)
					if len(ln.Instruction) > 0 {
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						win.img.drawDisasmForCoProc(ln.Instruction, ln, false, false, 0, shortDisasmWindow)
					}
				} else {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					imgui.Text("No source for this instruction")
				}
			}
		}, true)

		// open source window on click
		if imgui.IsItemClicked() && ln != nil {
			if !ln.IsStub() {
				srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
				srcWin.gotoSourceLine(ln)
			}
		}

		imgui.TableNextColumn()
		imgui.Text(e.Event)

		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcFaultsAddress)
		imgui.Text(fmt.Sprintf("%08x", e.AccessAddr))
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if src == nil {
			imgui.Text(fmt.Sprintf("%#08x", e.InstructionAddr))
		} else {
			if ln != nil {
				imgui.Text(ln.Function.Name)
			} else {
				imgui.Text("-")
			}
		}
	}

	imgui.EndTable()

	// options toolbar at foot of window
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Separator()
		imgui.Spacing()

		if src != nil {
			imgui.Checkbox("Show Source in Tooltip", &win.showSrcInTooltip)
			imgui.SameLineV(0, 20)
		}

		if hasStackCollision {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf(" %c", fonts.Warning))
			imgui.PopStyleColor()
			imgui.SameLine()
			imgui.AlignTextToFramePadding()
			imgui.Text("Stack collision detected")
			win.img.imguiTooltip(func() {
				imgui.Text("Results of memory access is unreliable after a stack collision")
				imgui.Text("and so memory faults are no longer being logged.")
			}, true)
		} else {
			// empty call to imgui.Text to consume an earlier call to imgui.SameLineV()
			imgui.Text("")
		}
	})
}
