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
)

const winCoProcLastExecutionID = "Last Execution"

type winCoProcLastExecution struct {
	img  *SdlImgui
	open bool
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

	itr := win.img.lz.Dbg.Disasm.Coprocessor.NewIteration()

	if itr.Count == 0 {
		imgui.Text("Coprocessor has not yet executed.")
	} else {
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

	imgui.BeginChildV("scrollable", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)

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
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(e.Address)

			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(e.Operator)

			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(e.Operand)

			imgui.SameLine()
			if e.Cycles > 0 {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(fmt.Sprintf("%.0f ", e.Cycles))
				imgui.PopStyleColorV(1)
			} else {
				imgui.Text(" ")
			}

			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			imgui.Text(e.ExecutionNotes)

			imgui.PopStyleColorV(4)

			e, ok = itr.Next()
			if !ok {
				break // clipper.DisplayStart loop
			}
		}
	}

	imgui.EndChild()

	imgui.End()
}
