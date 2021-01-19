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

	"github.com/inkyblackness/imgui-go/v3"
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

	imgui.SetNextWindowPosV(imgui.Vec2{905, 242}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)

	title := fmt.Sprintf("%s %s", win.img.lz.CoProc.ID, winCoProcLastExecutionID)
	imgui.BeginV(title, &win.open, 0)

	itr := win.img.lz.Dbg.Disasm.Coprocessor.NewIteration()

	if itr.Count == 0 {
		imgui.Text("Coprocessor has not yet executed.")
	} else {
		imgui.Text("Frame:")
		imgui.SameLine()
		imgui.Text(fmt.Sprintf("%-4d", itr.Details.Frame))
		imgui.SameLineV(0, 15)
		imgui.Text("Scanline:")
		imgui.SameLine()
		imgui.Text(fmt.Sprintf("%-3d", itr.Details.Scanline))
		imgui.SameLineV(0, 15)
		imgui.Text("Clock:")
		imgui.SameLine()
		imgui.Text(fmt.Sprintf("%-3d", itr.Details.Clock))

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
