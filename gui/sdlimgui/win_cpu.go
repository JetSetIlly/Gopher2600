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

	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"

	"github.com/inkyblackness/imgui-go/v4"
)

const winCPUID = "CPU"

type winCPU struct {
	img  *SdlImgui
	open bool

	// width of status register. we use this to set the width of the window.
	statusWidth float32

	// labels in the status register header are adjusted slightly so that they
	// are centred in the column
	statusLabelAdj imgui.Vec2
}

func newWinCPU(img *SdlImgui) (window, error) {
	win := &winCPU{
		img: img,
	}

	return win, nil
}

const statusRegisterNumColumns = 7

func (win *winCPU) init() {
	x := imgui.CalcTextSize("x", false, 0.0).X
	win.statusLabelAdj = imgui.Vec2{X: x / 2, Y: 0.0}

	// using imguiMeasureWidth() has side effects when used to measure tables.
	// fortunately, we can manually figure out the width of the status register
	// table quite easily.
	sty := imgui.CurrentStyle()
	win.statusWidth = statusRegisterNumColumns * (x + sty.ItemInnerSpacing().X + sty.ItemSpacing().X)
	win.statusWidth += ((statusRegisterNumColumns - 2) * sty.ItemSpacing().X)
}

func (win *winCPU) id() string {
	return winCPUID
}

func (win *winCPU) isOpen() bool {
	return win.open
}

func (win *winCPU) setOpen(open bool) {
	win.open = open
}

func (win *winCPU) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{836, 315}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{win.statusWidth, -1}, imgui.ConditionNone)
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNone)

	fillWidth := imgui.Vec2{X: -1, Y: imgui.FrameHeight()}

	if imgui.BeginTable("cpuLayout", 2) {
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, 75, 1)

		imgui.TableNextRow()
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.PC)
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.A)

		imgui.TableNextRow()
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.SP)
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.X)

		imgui.TableNextRow()
		imgui.TableNextColumn()
		_ = imguiBooleanButton(win.img.cols, win.img.lz.CPU.RdyFlg, "RDY Flag", fillWidth)
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.Y)

		imgui.EndTable()
	}

	imgui.Spacing()
	if imgui.BeginTable("statusRegister", statusRegisterNumColumns) {
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("s")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("o")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("b")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("d")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("i")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("z")
		imgui.TableNextColumn()
		imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
		imgui.Text("c")

		sr := win.img.lz.CPU.StatusReg
		col := win.img.cols.TitleBgActive

		imgui.TableNextRow()
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("s", sr.Sign, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE S")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("o", sr.Overflow, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE O")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("b", sr.Break, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE B")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("d", sr.DecimalMode, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE D")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("i", sr.InterruptDisable, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE I")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("z", sr.Zero, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE Z")
		}
		imgui.TableNextColumn()
		if imguiToggleButtonVertical("c", sr.Carry, col) {
			win.img.term.pushCommand("CPU STATUS TOGGLE C")
		}

		imgui.EndTable()
	}

	imgui.Spacing()

	res := win.img.lz.Debugger.LiveDisasmEntry
	if res.Address != "" {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
		imgui.Text(res.Address)

		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBank)
		imgui.Text(fmt.Sprintf("[bank %d]", res.Bank))

		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
		imgui.Text(res.Operator)

		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
		imgui.Text(res.Operand.String())

		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
		imgui.Text(fmt.Sprintf("%s cycles", res.Cycles()))
		if res.Result.PageFault {
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			imgui.Text("(page-fault)")
			imgui.PopStyleColor()
		}

		imgui.PopStyleColorV(5)
	} else {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
		imgui.Text("")
		imgui.Text("no execution yet")
		imgui.Text("")
		imgui.PopStyleColor()
	}

	imgui.End()
}

func (win *winCPU) drawRegister(reg registers.Generic) {
	if reg == nil {
		return
	}

	label := reg.Label()

	imguiLabel(fmt.Sprintf("% 2s", label))
	imgui.SameLine()

	content := reg.String()
	bitwidth := reg.BitWidth()

	if imguiHexInput(fmt.Sprintf("##%s", label), bitwidth/4, &content) {
		win.img.term.pushCommand(fmt.Sprintf("CPU SET %s %s", reg.Label(), content))
	}
}
