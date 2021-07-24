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
	"github.com/jetsetilly/gopher2600/disassembly/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
)

const winCoProcLastExecutionID = "Last Execution"

type winCoProcLastExecution struct {
	img  *SdlImgui
	open bool

	summaryHeight     float32
	showLastExecution bool
}

func newWinCoProcLastExecution(img *SdlImgui) (window, error) {
	win := &winCoProcLastExecution{
		img:               img,
		showLastExecution: true,
	}
	return win, nil
}

func (win *winCoProcLastExecution) init() {
}

func (win *winCoProcLastExecution) id() string {
	return winCoProcLastExecutionID
}

func (win *winCoProcLastExecution) isOpen() bool {
	return win.open
}

func (win *winCoProcLastExecution) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcLastExecution) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.CoProc.HasCoProcBus || win.img.dbg.Disasm.Coprocessor == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)

	title := fmt.Sprintf("%s %s", win.img.lz.CoProc.ID, winCoProcLastExecutionID)
	imgui.BeginV(title, &win.open, 0)
	defer imgui.End()

	var itr *coprocessor.Iterate
	if win.showLastExecution {
		itr = win.img.dbg.Disasm.Coprocessor.NewIteration(coprocessor.LastExecution)
	} else {
		itr = win.img.dbg.Disasm.Coprocessor.NewIteration(coprocessor.Disassembly)
	}

	if itr.Count != 0 {
		imguiLabel("Last execution at:")
		imgui.SameLineV(0, 15)
		imguiLabel("Frame:")
		imguiLabel(fmt.Sprintf("%-4d", itr.Details.Frame))
		imgui.SameLineV(0, 15)
		imguiLabel("Scanline:")
		imguiLabel(fmt.Sprintf("%-3d", itr.Details.Scanline))
		imgui.SameLineV(0, 15)
		imguiLabel("Clock:")
		imguiLabel(fmt.Sprintf("%-3d", itr.Details.Clock))

		imgui.SameLineV(0, 15)
		if !(itr.Details.Frame == win.img.lz.TV.Frame &&
			itr.Details.Scanline == win.img.lz.TV.Scanline &&
			itr.Details.Clock == win.img.lz.TV.Clock) {
			if imgui.Button("Goto") {
				win.img.dbg.PushGotoCoords(itr.Details.Frame, itr.Details.Scanline, itr.Details.Clock)
			}
		} else {
			imgui.InvisibleButton("Goto", imgui.Vec2{X: 10, Y: imgui.FrameHeight()})
		}

		imguiSeparator()
	}

	win.drawDisasm(itr)

	win.summaryHeight = imguiMeasureHeight(func() {
		imguiSeparator()

		if itr.Count == 0 {
			imgui.Text("Coprocessor has not yet executed.")
			return
		}

		if summary, ok := itr.Details.Summary.(arm7tdmi.DisasmSummary); ok {
			if summary.ImmediateMode {
				imgui.Text("Execution ran in immediate mode. Cycle counting disabled")
				imgui.Spacing()
			} else if imgui.BeginTableV("cycles", 3, imgui.TableFlagsBordersOuter, imgui.Vec2{}, 0.0) {
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

		imgui.Spacing()

		if imgui.Button("Save CSV") {
			win.save()
		}
		tooltipHover("CSV file separated by semi-colons")

		imgui.SameLineV(0, 15)
		if win.showLastExecution {
			imgui.Checkbox("Showing last executed sequence", &win.showLastExecution)
		} else {
			imgui.Checkbox("Showing disassembled program", &win.showLastExecution)
		}
	})
}

func (win *winCoProcLastExecution) drawDisasm(itr *coprocessor.Iterate) {
	height := imguiRemainingWinHeight() - win.summaryHeight
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
			tooltipHover(tooltip)

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
			tooltipHover(tooltip)

			if entry.MergedIS {
				imgui.SameLine()
				imguiColorLabel("", win.img.cols.CoProcMergedIS)
				tooltipHover("Merged I-S cycle")
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

func (win *winCoProcLastExecution) save() {
	var itr *coprocessor.Iterate
	var fn string
	if win.showLastExecution {
		itr = win.img.dbg.Disasm.Coprocessor.NewIteration(coprocessor.LastExecution)
		fn = paths.UniqueFilename("coproc_lastexecution", "")
	} else {
		itr = win.img.dbg.Disasm.Coprocessor.NewIteration(coprocessor.Disassembly)
		fn = paths.UniqueFilename("coproc_disasm", "")
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
