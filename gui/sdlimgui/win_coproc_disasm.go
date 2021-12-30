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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
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
			win.img.dbg.CoProcDisasm.Enable(true)
			if win.img.emulation.State() != emulation.Running {
				// rerun the last two frames in order to gather as much disasm
				// information as possible.
				win.img.dbg.RerunLastNFrames(2)
			}
		}
	} else {
		if imgui.Button("Disable disassembly") {
			win.img.dbg.CoProcDisasm.Enable(false)
		}
	}

	imguiSeparator()

	imgui.BeginTabBar("")
	if imgui.BeginTabItem("Disassembly") {
		itr := win.img.dbg.CoProcDisasm.NewIteration(coprocessor.IterateComplete)
		win.drawDisasm(itr)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Last Execution") {
		itr := win.img.dbg.CoProcDisasm.NewIteration(coprocessor.IterateLast)
		win.drawLastExecution(itr)
		imgui.EndTabItem()
	}
	imgui.EndTabBar()
}

func (win *winCoProcDisasm) drawDisasm(itr *coprocessor.Iterate) {
	if itr.Count == 0 {
		imgui.Spacing()
		imgui.Text("Coprocessor has not yet executed.")
		imgui.Spacing()
		return
	}

	if !win.img.dbg.CoProcDisasm.IsEnabled() {
		imgui.Spacing()
		imgui.Text("Execution disassembly is disabled. Disassembly below may be")
		imgui.Text("incomplete. Last disassembly was at:")
		imgui.Text(itr.LastStart.String())
		imgui.Spacing()
	}

	win.drawIteration(itr, false)
}

func (win *winCoProcDisasm) drawLastExecution(itr *coprocessor.Iterate) {
	if itr.Count == 0 {
		imgui.Spacing()
		imgui.Text("Execution disassembly is disabled")
		imgui.Spacing()
		return
	}

	imgui.Spacing()
	imgui.Text("Started at:")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Frame: %-4d", itr.LastStart.Frame))
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Scanline: %-3d", itr.LastStart.Scanline))
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("Clock: %-3d", itr.LastStart.Clock))
	imgui.Spacing()

	win.drawIteration(itr, true)

	win.summaryHeight = imguiMeasureHeight(func() {
		imguiSeparator()

		if summary, ok := itr.Summary.(arm7tdmi.DisasmSummary); ok {
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

func (win *winCoProcDisasm) drawIteration(itr *coprocessor.Iterate, adjustHeightForSummary bool) {
	height := imguiRemainingWinHeight()

	if adjustHeightForSummary {
		height -= win.summaryHeight
	}

	imgui.BeginChildV("lastexecution", imgui.Vec2{X: 0, Y: height}, false, 0)
	defer imgui.EndChild()

	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	if !imgui.BeginTableV("lastexecution", 8, flgs, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	// set neutral colors for table rows by default. we'll change it to
	// something more meaningful as appropriate (eg. entry at PC address)
	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)
	defer imgui.PopStyleColorV(2)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(itr.Count)
	for clipper.Step() {
		_, _ = itr.Start()
		e, ok := itr.SkipNext(clipper.DisplayStart)
		if !ok {
			break // clipper.Step() loop
		}

		// several columns use a tooltip
		tooltip := ""

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			entry := (*e).(arm7tdmi.DisasmEntry)

			imgui.TableNextRow()

			imgui.TableNextColumn()
			switch entry.MAMCR {
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
			imgui.Text(entry.Address)
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(entry.Operator)
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(entry.Operand)
			imgui.PopStyleColor()

			// branch trail and merged IS indicator
			imgui.TableNextColumn()
			tooltip = ""
			switch entry.BranchTrail {
			case arm7tdmi.BranchTrailUsed:
				tooltip = "Branch trail was used"
				imguiColorLabel("", win.img.cols.CoProcBranchTrailUsed)
			case arm7tdmi.BranchTrailFlushed:
				tooltip = "Branch trail was flushed causing a pipeline stall"
				imguiColorLabel("", win.img.cols.CoProcBranchTrailFlushed)
			}
			imguiTooltipSimple(tooltip)

			if entry.MergedIS {
				imgui.SameLine()
				imguiColorLabel("", win.img.cols.CoProcMergedIS)
				imguiTooltipSimple("Merged I-S cycle")
			}

			// cycle sequence
			imgui.TableNextColumn()
			imgui.Text(entry.CyclesSequence)

			// cycles total
			imgui.TableNextColumn()
			if entry.Cycles > 0 {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(fmt.Sprintf("%d", entry.Cycles))
				imgui.PopStyleColor()
			} else {
				imgui.Text("??")
			}

			// execution notes
			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			imgui.Text(entry.ExecutionNotes)
			imgui.PopStyleColor()

			e, ok = itr.Next()
			if !ok {
				break // clipper.DisplayStart loop
			}
		}
	}
}

func (win *winCoProcDisasm) save() {
	var itr *coprocessor.Iterate
	var fn string
	if win.showLastExecution {
		itr = win.img.dbg.CoProcDisasm.NewIteration(coprocessor.IterateLast)
		fn = unique.Filename("coproc_lastexecution", "")
	} else {
		itr = win.img.dbg.CoProcDisasm.NewIteration(coprocessor.IterateComplete)
		fn = unique.Filename("coproc_disasm", "")
	}

	f, err := os.Create(fmt.Sprintf("%s.csv", fn))
	if err != nil {
		logger.Logf("sdlimgui", "error saving last coproc execution: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf("sdlimgui", "error saving last coproc execution: %v", err)
		}
	}()

	e, _ := itr.Start()
	for e != nil {
		f.Write([]byte((*e).String()))
		f.Write([]byte("\n"))
		e, _ = itr.Next()
	}
}
