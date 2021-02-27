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

	// labels in the status register header are adjusted slightly so that they
	// are centered in the column
	statusLabelAdj imgui.Vec2
}

func newWinCPU(img *SdlImgui) (window, error) {
	win := &winCPU{
		img: img,
	}

	return win, nil
}

func (win *winCPU) init() {
	win.statusLabelAdj = imgui.Vec2{X: imgui.CalcTextSize("x", false, 0.0).X / 2, Y: 0.0}
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

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

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
		_ = imguiBooleanButton(win.img.cols, win.img.lz.CPU.RdyFlg, "RDY Flag", imgui.Vec2{X: -1, Y: imgui.FrameHeight()})
		imgui.TableNextColumn()
		win.drawRegister(win.img.lz.CPU.Y)

		imgui.EndTable()
	}

	imgui.Spacing()
	if imgui.BeginTableV("statusRegister", 7, imgui.TableFlagsBordersOuter, imgui.Vec2{}, 0.0) {
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

		imgui.TableNextRow()
		imgui.EndTable()
	}

	imgui.Spacing()
	res := win.img.lz.Debugger.LastResult
	if res.Address != "" {
		imgui.Text(fmt.Sprintf("[%d] %s %s %s", res.Bank.Number, res.Address, res.Operator, res.Operand))
		if !res.Result.Final {
			imgui.Text(fmt.Sprintf("%s of %s cycles", res.ActualCycles, res.DefnCycles))
		} else {
			imgui.Indent()
			imgui.Text(fmt.Sprintf("%s cycles", res.ActualCycles))
			if res.Result.PageFault {
				imgui.SameLine()
				imgui.Text("(page-fault)")
			}
		}
	} else {
		imgui.Text("no execution yet")
		imgui.Text("")
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
