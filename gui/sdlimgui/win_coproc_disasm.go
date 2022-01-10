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

	summaryHeight     float32
	showLastExecution bool
}

func newWinCoProcDisasm(img *SdlImgui) (window, error) {
	win := &winCoProcDisasm{
		img:               img,
		showLastExecution: true,
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

	// show enable button only if coprocessor disassembly is disabled
	if !win.img.dbg.CoProcDisasm.IsEnabled() {
		if imgui.Button("Enable disassembly") {
			win.img.dbg.PushRawEvent(func() {
				win.img.dbg.CoProcDisasm.Enable(true)
				if win.img.emulation.State() != emulation.Running {
					// rerun the last two frames in order to gather as much disasm
					// information as possible.
					win.img.dbg.RerunLastNFrames(2)
				}
			})
		}
	} else {
		if imgui.Button("Disable disassembly") {
			win.img.dbg.CoProcDisasm.Enable(false)
		}
	}

	imguiSeparator()

	imgui.BeginTabBar("##coprocDisasmTabBar")
	if imgui.BeginTabItem("Disassembly") {
		win.img.dbg.CoProcDisasm.BorrowDisassembly(win.drawDisasm)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Last Execution") {
		win.img.dbg.CoProcDisasm.BorrowDisassembly(win.drawLastExecution)
		imgui.EndTabItem()
	}
	imgui.EndTabBar()
}

func (win *winCoProcDisasm) drawDisasm(dsm *disassembly.DisasmEntries) {
	if !dsm.Enabled {
		imgui.Spacing()
		imgui.Text("Execution disassembly is disabled")
		return
	}

	height := imguiRemainingWinHeight()
	imgui.BeginChildV("disasm", imgui.Vec2{X: 0, Y: height}, false, 0)

	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	imgui.BeginTableV("disasmTable", 9, flgs, imgui.Vec2{}, 0)

	// set neutral colors for table rows by default. we'll change it to
	// something more meaningful as appropriate (eg. entry at PC address)
	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
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

	imgui.PopStyleColorV(2)
	imgui.EndTable()
	imgui.EndChild()
}

func (win *winCoProcDisasm) drawLastExecution(dsm *disassembly.DisasmEntries) {
	if !dsm.Enabled {
		imgui.Spacing()
		imgui.Text("Execution disassembly is disabled")
		return
	}

	imgui.Spacing()
	imgui.Text("Started at:")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Frame: %-4d", dsm.LastStart.Frame))
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Scanline: %-3d", dsm.LastStart.Scanline))
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Clock: %-3d", dsm.LastStart.Clock))
	imgui.Spacing()

	height := imguiRemainingWinHeight() - win.summaryHeight
	imgui.BeginChildV("disasm", imgui.Vec2{X: 0, Y: height}, false, 0)

	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	imgui.BeginTableV("disasmTable", 9, flgs, imgui.Vec2{}, 0)

	// set neutral colors for table rows by default. we'll change it to
	// something more meaningful as appropriate (eg. entry at PC address)
	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(len(dsm.Entries))
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

	imgui.PopStyleColorV(2)
	imgui.EndTable()
	imgui.EndChild()

	win.summaryHeight = imguiMeasureHeight(func() {
		imguiSeparator()

		if summary, ok := dsm.LastExecutionSummary.(arm7tdmi.DisasmSummary); ok {
			if summary.ImmediateMode {
				imgui.Text("Execution ran in immediate mode. Cycle counting disabled")
				imgui.Spacing()
			} else if imgui.BeginTableV("cycles", 3, imgui.TableFlagsNone, imgui.Vec2{}, 0.0) {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("N: %d", summary.N))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("I: %d", summary.I))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("S: %d", summary.S))
				imgui.EndTable()
			}
		} else {
			imgui.Text("cannot find a summary of execution")
		}
	})
}

func (win *winCoProcDisasm) drawEntry(e arm7tdmi.DisasmEntry) {
	// several columns use a tooltip
	tooltip := ""

	imgui.TableNextRow()

	// highlight line mouse is over
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	imgui.TableNextColumn()
	switch e.MAMCR {
	case 0:
		imguiColorLabel("", win.img.cols.CoProcMAM0)
		tooltip = "MAM-0"
	case 1:
		imguiColorLabel("", win.img.cols.CoProcMAM1)
		tooltip = "MAM-1"
	case 2:
		imguiColorLabel("", win.img.cols.CoProcMAM2)
		tooltip = "MAM-2"
	}
	imguiTooltipSimple(tooltip)

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
	tooltip = ""
	switch e.BranchTrail {
	case arm7tdmi.BranchTrailUsed:
		tooltip = "Branch trail was used"
		imguiColorLabel("", win.img.cols.CoProcBranchTrailUsed)
	case arm7tdmi.BranchTrailFlushed:
		tooltip = "Branch trail was flushed causing a pipeline stall"
		imguiColorLabel("", win.img.cols.CoProcBranchTrailFlushed)
	}
	imguiTooltipSimple(tooltip)

	if e.MergedIS {
		imgui.SameLine()
		imguiColorLabel("", win.img.cols.CoProcMergedIS)
		imguiTooltipSimple("Merged I-S cycle")
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

	// execution notes
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
	imgui.Text(e.ExecutionNotes)
	imgui.PopStyleColor()
}
