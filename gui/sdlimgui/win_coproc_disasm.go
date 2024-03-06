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
	"github.com/jetsetilly/gopher2600/coprocessor/disassembly"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

const winCoProcDisasmID = "Coprocessor Disassembly"
const winCoProcDisasmMenu = "Disassembly"

type winCoProcDisasm struct {
	debuggerWin

	img *SdlImgui

	optionsHeight        float32
	optionsLastExecution bool

	// scroll window if last item is visible
	lastItemVisible bool
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

func (win *winCoProcDisasm) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{551, 526}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcDisasmID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		// only support specific ARM architectures
		arch := architecture.ARMArchitecture(coproc.ProcessorID())
		if arch == architecture.ARM7TDMI || arch == architecture.ARMv7_M {
			win.draw()
		} else {
			imgui.Text(fmt.Sprintf("%s is an unsupported architecture", arch))
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcDisasm) draw() {
	height := imguiRemainingWinHeight() - win.optionsHeight
	isEnabled := win.img.dbg.CoProcDisasm.IsEnabled()

	win.img.dbg.CoProcDisasm.BorrowDisassembly(func(dsm *disassembly.DisasmEntries) {
		if imgui.BeginChildV("##coprocDisasmMain", imgui.Vec2{X: 0, Y: height}, false, imgui.WindowFlagsNone) {
			if isEnabled {
				imgui.BeginTabBar("##coprocDisasmTabBar")
				if imgui.BeginTabItem("Disassembly") {
					win.optionsLastExecution = false
					if len(dsm.Entries) > 0 {
						win.drawDisasm(dsm, false)
					} else {
						imgui.Spacing()
						imgui.Text("No disassembly available")
					}
					imgui.EndTabItem()
				}
				if imgui.BeginTabItem("Last Execution") {
					win.optionsLastExecution = true
					if len(dsm.LastExecution) > 0 {
						win.drawDisasm(dsm, true)
					} else {
						imgui.Spacing()
						imgui.Text("No recent disassembly available")
					}
					imgui.EndTabItem()
				}
				imgui.EndTabBar()
			} else {
				imgui.Text("Coprocessor disassembly is disabled")
			}

		}
		imgui.EndChild()

		// draw options and status line. start height measurement
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			// options
			if imgui.Checkbox("Disassembly Enabled", &isEnabled) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.CoProcDisasm.Enable(isEnabled)
					if win.img.dbg.State() != govern.Running {
						// rerun the last two frames in order to gather as much disasm
						// information as possible.
						win.img.dbg.RerunLastNFrames(2)
					}
				})
			}

			// total cycles including tooltip
			if isEnabled && win.optionsLastExecution {
				if summary, ok := dsm.LastExecutionSummary.(arm.DisasmSummary); ok {
					imgui.SameLineV(0, 40)
					imgui.Text(string(fonts.CoProcCycles))
					if summary.ImmediateMode {
						imgui.SameLineV(0, 10)
						imgui.Text("Executed in Immediate Mode")
					} else {
						if len(dsm.LastExecution) > 0 {
							imgui.SameLineV(0, 10)
							imgui.Text(fmt.Sprintf("Total Cycles % 8d", summary.I+summary.N+summary.S))
							win.img.imguiTooltip(func() {
								imgui.Text(fmt.Sprintf("N cycles: % 8d", summary.N))
								imgui.Text(fmt.Sprintf("I cycles: % 8d", summary.I))
								imgui.Text(fmt.Sprintf("S cycles: % 8d", summary.S))
							}, true)
						} else {
							imgui.SameLineV(0, 10)
							imgui.Text("Recent execution unavailable")
						}
					}
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

	const numColumns = 9

	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	imgui.BeginTableV("disasmTable", numColumns, flgs, imgui.Vec2{}, 0)
	defer imgui.EndTable()

	// first column is a dummy column so that Selectable (span all columns) works correctly
	width := imgui.ContentRegionAvail().X
	imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0.00, 0)
	imgui.TableSetupColumnV("MAM", imgui.TableColumnFlagsNone, width*0.025, 1)
	imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.15, 2)
	imgui.TableSetupColumnV("Operator", imgui.TableColumnFlagsNone, width*0.07, 3)
	imgui.TableSetupColumnV("Operands", imgui.TableColumnFlagsNone, width*0.23, 4)
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

	win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
		if lastExecution {
			imgui.Text("State of execution has recently changed. Last execution details currently unavailable.")
			clipper.Begin(len(dsm.LastExecution))
			for clipper.Step() {
				for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
					if i >= len(dsm.LastExecution) {
						imgui.Text("")
						break
					}
					e := dsm.LastExecution[i]
					win.drawEntry(src, e.(arm.DisasmEntry))
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
					win.drawEntry(src, e.(arm.DisasmEntry))
				}
			}

		}

		// scroll window with the last item, if the last item was visible on the
		// last frame
		if win.lastItemVisible {
			imgui.SetScrollY(imgui.ScrollMaxY())
		}
		win.lastItemVisible = clipper.DisplayEnd >= len(dsm.Entries)
	})
}

func (win *winCoProcDisasm) drawEntry(src *dwarf.Source, e arm.DisasmEntry) {
	var ln *dwarf.SourceLine
	if src != nil {
		ln = src.FindSourceLine(e.Addr)
	}

	imgui.TableNextRow()

	// highlight line mouse is over
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.PopStyleColorV(2)

	// open source window if there is underlying source for this instruction
	if imgui.IsItemClicked() && ln != nil {
		srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
		srcWin.gotoSourceLine(ln)
	}

	if imgui.IsItemHovered() {
		win.drawEntryTooltip(e, ln)
	}

	imgui.TableNextColumn()
	switch e.MAMCR {
	case 0:
		imguiColorLabelSimple("", win.img.cols.CoProcMAM0)
	case 1:
		imguiColorLabelSimple("", win.img.cols.CoProcMAM1)
	case 2:
		imguiColorLabelSimple("", win.img.cols.CoProcMAM2)
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
	case arm.BranchTrailUsed:
		imguiColorLabelSimple("", win.img.cols.CoProcBranchTrailUsed)
	case arm.BranchTrailFlushed:
		imguiColorLabelSimple("", win.img.cols.CoProcBranchTrailFlushed)
	}

	imgui.TableNextColumn()
	if e.MergedIS {
		imguiColorLabelSimple("", win.img.cols.CoProcMergedIS)
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
		imgui.Text("")
	}
}

func (win *winCoProcDisasm) drawEntryTooltip(e arm.DisasmEntry, ln *dwarf.SourceLine) {
	win.img.imguiTooltip(func() {
		// if ln is nil then that means there is no source code available for the disassembly
		if ln != nil && ln.Function != nil {
			imgui.Text("Function:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmLocation)
			imgui.Text(ln.Function.Name)
			imgui.PopStyleColor()
		}

		imgui.Text("Address:")
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
		imgui.Text(e.Address)
		imgui.PopStyleColor()

		imgui.Text("Opcode:")
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode)
		if e.Is32bit {
			imgui.Text(fmt.Sprintf("%04x", e.OpcodeHi))
			imgui.SameLine()
		}
		imgui.Text(fmt.Sprintf("%04x", e.Opcode))
		imgui.PopStyleColor()

		imgui.Text("Instruction:")
		imgui.SameLine()
		if e.Operator != "" {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(e.Operator)
			imgui.PopStyleColor()
			imgui.SameLine()
		}
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
		imgui.Text(e.Operand)
		imgui.PopStyleColor()

		imgui.Text("Cycles:")
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
		if e.ImmediateMode {
			imgui.Text("Immediate Mode")
		} else {
			imgui.Text(fmt.Sprintf("%d", e.Cycles))
		}
		imgui.PopStyleColor()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		switch e.MAMCR {
		case 0:
			imguiColorLabelSimple("MAM-0", win.img.cols.CoProcMAM0)
		case 1:
			imguiColorLabelSimple("MAM-1", win.img.cols.CoProcMAM1)
		case 2:
			imguiColorLabelSimple("MAM-2", win.img.cols.CoProcMAM2)
		}

		if e.MergedIS {
			imguiColorLabelSimple("Merged I/S Cycle", win.img.cols.CoProcMergedIS)
		}

		switch e.BranchTrail {
		case arm.BranchTrailUsed:
			imguiColorLabelSimple("Branch Trail Used", win.img.cols.CoProcBranchTrailUsed)
		case arm.BranchTrailFlushed:
			imguiColorLabelSimple("Branch Trail Flushed", win.img.cols.CoProcBranchTrailFlushed)
		}

		if ln != nil {
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()

			if ln.IsStub() {
				imgui.Text("No sourcecode available")
			} else {
				imgui.Text(ln.File.ShortFilename)
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
				imgui.Text(fmt.Sprintf("Line: %d", ln.LineNumber))
				imgui.PopStyleColor()
			}
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Indent()
		if imgui.BeginTable("coprocDisasmTooltipRegisters", 2) {
			for r, v := range e.Registers {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("R%d", r))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%08x", v))
			}
			imgui.EndTable()
		}
	}, false)
}
