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
	"github.com/jetsetilly/gopher2600/disassembly/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"
)

const winCoProcLastExecutionID = "Last Execution"

type winCoProcLastExecution struct {
	img  *SdlImgui
	open bool

	summaryHeight float32
}

func newWinCoProcLastExecution(img *SdlImgui) (window, error) {
	win := &winCoProcLastExecution{
		img: img,
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

	if !win.img.lz.CoProc.HasCoProcBus || win.img.lz.Dbg.Disasm.Coprocessor == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{465, 285}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)

	title := fmt.Sprintf("%s %s", win.img.lz.CoProc.ID, winCoProcLastExecutionID)
	imgui.BeginV(title, &win.open, 0)
	defer imgui.End()

	itr := win.img.lz.Dbg.Disasm.Coprocessor.NewIteration()

	if itr.Count != 0 {
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
				win.img.lz.Dbg.PushGotoCoords(itr.Details.Frame, itr.Details.Scanline, itr.Details.Clock)
			}
		} else {
			imgui.InvisibleButtonV("Goto", imgui.Vec2{1, 1}, imgui.ButtonFlagsNone)
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

		if programCycles, ok := itr.Details.Summary.(arm7tdmi.Cycles); ok {
			iTotal := programCycles.I + programCycles.Imerged
			nTotal := programCycles.Npc + programCycles.Ndata
			sTotal := programCycles.Spc + programCycles.Sdata + programCycles.Smerged

			if imgui.BeginTableV("cycles", 4, imgui.TableFlagsBordersOuter, imgui.Vec2{}, 0.0) {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("I: %-6.0f", iTotal))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("C: %-6.0f", programCycles.C))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("N: %-6.0f", nTotal))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("S: %-6.0f", sTotal))
				imgui.EndTable()
			}

			imgui.Spacing()

			nsTotal := nTotal + sTotal
			fp := 100.0 * programCycles.FlashAccess / nsTotal
			sp := 100.0 * programCycles.SRAMAccess / nsTotal

			if imgui.BeginTableV("ratios", 2, imgui.TableFlagsBordersOuter, imgui.Vec2{}, 0.0) {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("Flash: %3.1f%%", fp))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("SRAM/MAM: %3.1f%%", sp))
				imgui.EndTable()
			}

			if imgui.IsItemHovered() {
				imgui.BeginTooltip()
				imgui.Text("The fraction of time spent by N and S cycles accessing flash or SRAM/MAM")
				imgui.EndTooltip()
			}
		} else {
			imgui.Text("")
		}
	})
}

func (win *winCoProcLastExecution) drawDisasm(itr *coprocessor.Iterate) {
	height := imguiRemainingWinHeight() - win.summaryHeight
	imgui.BeginChildV("lastexecution", imgui.Vec2{X: 0, Y: height}, false, 0)
	defer imgui.EndChild()

	numColumns := 5
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	if !imgui.BeginTableV("lastexecution", numColumns, flgs, imgui.Vec2{}, 0) {
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

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			imgui.TableNextRow()

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(e.Address)

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(e.Operator)

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(e.Operand)

			imgui.TableNextColumn()
			if e.Cycles > 0 {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(fmt.Sprintf("%.0f ", e.Cycles))

				// show cycle details as a tooltip
				if e.CycleDetails.String() != "" && imgui.IsItemHovered() {
					imgui.BeginTooltip()
					imgui.Text(e.CycleDetails.String())
					if cycles, ok := e.CycleDetails.(arm7tdmi.Cycles); ok {
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						if cycles.MAMenabled {
							imguiColorLabel("MAM", win.img.cols.True)
						} else {
							imguiColorLabel("MAM", win.img.cols.False)
						}
					}
					imgui.EndTooltip()
				}

				imgui.PopStyleColorV(1)
			} else {
				imgui.Text(" ")
			}

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			imgui.Text(e.ExecutionNotes)

			imgui.PopStyleColorV(4)

			e, ok = itr.Next()
			if !ok {
				break // clipper.DisplayStart loop
			}
		}
	}
}
