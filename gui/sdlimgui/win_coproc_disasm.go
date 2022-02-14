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
	"github.com/jetsetilly/gopher2600/coprocessor/disassembly"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcDisasmID = "Coprocessor Disassembly"
const winCoProcDisasmMenu = "Disassembly"

type winCoProcDisasm struct {
	img  *SdlImgui
	open bool

	summaryHeight float32

	optionsHeight        float32
	optionsLastExecution bool
}

func newWinCoProcDisasm(img *SdlImgui) (window, error) {
	win := &winCoProcDisasm{
		img: img,
	}
	return win, nil
}

func (win *winCoProcDisasm) init() {
}

func (win *winCoProcDisasm) id() string {
	return winCoProcDisasmID
}

func (win *winCoProcDisasm) isOpen() bool {
	return win.open
}

func (win *winCoProcDisasm) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcDisasm) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDisasm == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{800, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcDisasmID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	height := imguiRemainingWinHeight() - win.optionsHeight
	isEnabled := win.img.dbg.CoProcDisasm.IsEnabled()

	win.img.dbg.CoProcDisasm.BorrowDisassembly(func(dsm *disassembly.DisasmEntries) {
		if imgui.BeginChildV("##coprocDisasmMain", imgui.Vec2{X: 0, Y: height}, false, imgui.WindowFlagsNone) {
			if isEnabled {
				imgui.BeginTabBar("##coprocDisasmTabBar")
				if imgui.BeginTabItem("Disassembly") {
					win.optionsLastExecution = false
					win.drawDisasm(dsm, false)
					imgui.EndTabItem()
				}
				if imgui.BeginTabItem("Last Execution") {
					win.optionsLastExecution = true
					win.drawDisasm(dsm, true)
					imgui.EndTabItem()
				}
				imgui.EndTabBar()
			} else {
				imgui.Text("Coprocessor disassembly is disabled")
			}

			imgui.EndChild()
		}

		// draw options and status line. start height measurement
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			// options
			if imgui.Checkbox("Disassembly Enabled", &isEnabled) {
				win.img.dbg.PushRawEvent(func() {
					win.img.dbg.CoProcDisasm.Enable(isEnabled)
					if win.img.emulation.State() != emulation.Running {
						// rerun the last two frames in order to gather as much disasm
						// information as possible.
						win.img.dbg.RerunLastNFrames(2)
					}
				})
			}

			// total cycles including tooltip
			if isEnabled && win.optionsLastExecution {
				if summary, ok := dsm.LastExecutionSummary.(arm7tdmi.DisasmSummary); ok {
					imgui.SameLineV(0, 40)
					imgui.Text(fmt.Sprintf("%c Total Cycles % 8d", fonts.CoProcCycles, summary.I+summary.N+summary.S))
					imguiTooltip(func() {
						imgui.Text(fmt.Sprintf("N cycles: % 8d", summary.N))
						imgui.Text(fmt.Sprintf("I cycles: % 8d", summary.I))
						imgui.Text(fmt.Sprintf("S cycles: % 8d", summary.S))
					}, true)
				}
			}
		})
	})
}

func (win *winCoProcDisasm) drawDisasm(dsm *disassembly.DisasmEntries, lastExecution bool) {
	height := imguiRemainingWinHeight()
	imgui.BeginChildV("disasm", imgui.Vec2{X: 0, Y: height}, false, 0)
	defer imgui.EndChild()

	if dsm == nil {
		return
	}

	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	imgui.BeginTableV("disasmTable", 9, flgs, imgui.Vec2{}, 0)
	defer imgui.EndTable()

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0.00, 0)
	imgui.TableSetupColumnV("MAM", imgui.TableColumnFlagsNone, width*0.025, 1)
	imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.15, 2)
	imgui.TableSetupColumnV("Opertor", imgui.TableColumnFlagsNone, width*0.05, 3)
	imgui.TableSetupColumnV("Operands", imgui.TableColumnFlagsNone, width*0.25, 4)
	imgui.TableSetupColumnV("Branch Trail", imgui.TableColumnFlagsNone, width*0.025, 5)
	imgui.TableSetupColumnV("MergedIS", imgui.TableColumnFlagsNone, width*0.025, 5)
	imgui.TableSetupColumnV("Cycle Profile", imgui.TableColumnFlagsNone, width*0.30, 6)
	imgui.TableSetupColumnV("Cycle Count", imgui.TableColumnFlagsNone, width*0.025, 7)

	// set neutral colors for table rows by default. we'll change it to
	// something more meaningful as appropriate (eg. entry at PC address)
	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)
	defer imgui.PopStyleColorV(2)

	var clipper imgui.ListClipper

	if lastExecution {
		clipper.Begin(len(dsm.LastExecution))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				if i >= len(dsm.LastExecution) {
					imgui.Text("")
					break
				}
				e := dsm.LastExecution[i]
				win.drawEntry(e.(arm7tdmi.DisasmEntry))
			}
		}
	} else {
		clipper.Begin(len(dsm.Entries))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				if i >= len(dsm.Keys) {
					imgui.Text("")
					break
				}
				k := dsm.Keys[i]
				e := dsm.Entries[k]
				win.drawEntry(e.(arm7tdmi.DisasmEntry))
			}
		}
	}

}

func (win *winCoProcDisasm) drawEntry(e arm7tdmi.DisasmEntry) {
	imgui.TableNextRow()

	// highlight line mouse is over
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	if imgui.IsItemHovered() && e.Operator != "" {
		imguiTooltip(func() {
			imgui.Text("Address:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(e.Address)
			imgui.PopStyleColor()

			imgui.Text("Instruction:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(e.Operator)
			imgui.PopStyleColor()
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(e.Operand)
			imgui.PopStyleColor()

			imgui.Text("Cycles:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
			imgui.Text(fmt.Sprintf("%d", e.Cycles))
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			switch e.MAMCR {
			case 0:
				imguiColorLabel("MAM-0", win.img.cols.CoProcMAM0)
			case 1:
				imguiColorLabel("MAM-1", win.img.cols.CoProcMAM1)
			case 2:
				imguiColorLabel("MAM-2", win.img.cols.CoProcMAM2)
			}

			switch e.BranchTrail {
			case arm7tdmi.BranchTrailUsed:
				imguiColorLabel("Branch Trail Used", win.img.cols.CoProcBranchTrailUsed)
			case arm7tdmi.BranchTrailFlushed:
				imguiColorLabel("Branch Trail Flushed", win.img.cols.CoProcBranchTrailFlushed)
			}

			if e.MergedIS {
				imguiColorLabel("Merged I/S Cycle", win.img.cols.CoProcMergedIS)
			}

			if e.ExecutionNotes != "" {
				imgui.SameLineV(0, 20)
				imgui.Text(fmt.Sprintf("%c %s", fonts.ExecutionNotes, e.ExecutionNotes))
			}
		}, false)
	}

	imgui.TableNextColumn()
	switch e.MAMCR {
	case 0:
		imguiColorLabel("", win.img.cols.CoProcMAM0)
	case 1:
		imguiColorLabel("", win.img.cols.CoProcMAM1)
	case 2:
		imguiColorLabel("", win.img.cols.CoProcMAM2)
	}

	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
	imgui.Text(e.Address)
	imgui.PopStyleColor()

	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
	imgui.Text(e.Operator)
	imgui.PopStyleColor()

	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
	imgui.Text(e.Operand)
	imgui.PopStyleColor()

	// branch trail and merged IS indicator
	imgui.TableNextColumn()
	switch e.BranchTrail {
	case arm7tdmi.BranchTrailUsed:
		imguiColorLabel("", win.img.cols.CoProcBranchTrailUsed)
	case arm7tdmi.BranchTrailFlushed:
		imguiColorLabel("", win.img.cols.CoProcBranchTrailFlushed)
	}

	imgui.TableNextColumn()
	if e.MergedIS {
		imguiColorLabel("", win.img.cols.CoProcMergedIS)
	}

	// cycle sequence
	imgui.TableNextColumn()
	imgui.Text(e.CyclesSequence)

	// cycles total
	imgui.TableNextColumn()
	if e.Cycles > 0 {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
		imgui.Text(fmt.Sprintf("%d", e.Cycles))
		imgui.PopStyleColor()
	} else {
		imgui.Text("??")
	}
}
