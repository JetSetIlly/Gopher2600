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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"

	"github.com/jetsetilly/imgui-go/v5"
)

const winCPUID = "CPU"

type winCPU struct {
	debuggerWin

	img *SdlImgui

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

func (win *winCPU) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 836, Y: 315}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSize(imgui.Vec2{X: imguiTextWidth(25), Y: -1})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCPU) draw() {
	fillWidth := imgui.Vec2{X: -1, Y: imgui.FrameHeight()}

	if imgui.BeginTable("cpuLayout", 2) {
		imgui.TableSetupColumnV("registers0", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 0)
		imgui.TableSetupColumnV("registers1", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)

		imgui.TableNextRow()
		imgui.TableNextColumn()
		win.drawRegister(win.img.cache.VCS.CPU.PC)
		imgui.TableNextColumn()
		imgui.PushStyleVarFloat(imgui.StyleVarFrameRounding, readOnlyButtonRounding)
		if win.img.cache.VCS.CPU.Killed {
			_ = imguiColourButton(win.img.cols.CPUKIL, fmt.Sprintf("%c Killed", fonts.CPUKilled), fillWidth)
		} else {
			_ = imguiBooleanButton(win.img.cols.CPURDY, win.img.cols.CPUNotRDY, win.img.cache.VCS.CPU.RdyFlg, "RDY", fillWidth)
		}
		imgui.PopStyleVar()

		imgui.TableNextRow()
		imgui.TableNextRow()
		imgui.TableNextColumn()

		win.drawRegister(win.img.cache.VCS.CPU.A)
		imgui.TableNextColumn()
		win.drawRegister(win.img.cache.VCS.CPU.SP)

		imgui.TableNextRow()
		imgui.TableNextColumn()

		win.drawRegister(win.img.cache.VCS.CPU.X)
		imgui.TableNextColumn()
		win.drawRegister(win.img.cache.VCS.CPU.Y)

		imgui.EndTable()
	}

	imgui.Spacing()
	if imgui.CollapsingHeaderV("Status Register", imgui.TreeNodeFlagsDefaultOpen) {
		if imgui.BeginTable("statusRegister", statusRegisterNumColumns) {
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("S")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("O")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("B")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("D")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("I")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("Z")
			imgui.TableNextColumn()
			imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.statusLabelAdj))
			imgui.Text("C")

			sr := win.img.cache.VCS.CPU.Status
			fg := win.img.cols.Text
			bg := win.img.cols.TitleBgActive

			imgui.TableNextRow()
			imgui.TableNextColumn()
			if imguiToggleButton("s", &sr.Sign, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE S")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("o", &sr.Overflow, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE O")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("b", &sr.Break, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE B")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("d", &sr.DecimalMode, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE D")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("i", &sr.InterruptDisable, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE I")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("z", &sr.Zero, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE Z")
			}
			imgui.TableNextColumn()
			if imguiToggleButton("c", &sr.Carry, fg, bg, true, 0.75) {
				win.img.term.pushCommand("CPU STATUS TOGGLE C")
			}

			imgui.EndTable()
		}
	}

	imgui.Spacing()

	if imgui.CollapsingHeaderV("Last Cycle", imgui.TreeNodeFlagsDefaultOpen) {
		res := win.img.cache.Dbg.LiveDisasmEntry
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
			imgui.Text(res.Operand.Resolve())

			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
			imgui.Text(fmt.Sprintf("%s cycles", res.Cycles()))

			if !win.img.cache.Dbg.LiveDisasmEntry.Result.Final {
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)

				switch win.img.cache.VCS.TIA.ClocksSinceCycle {
				case 1:
					imgui.Text(fmt.Sprintf("%c", fonts.Paw))
				case 2:
					imgui.Text(fmt.Sprintf("%c", fonts.Paw))
					imgui.SameLineV(0, 4)
					imgui.Text(fmt.Sprintf("%c", fonts.Paw))
				case 3:
					// only show paws for value 3 if we're in QuantumClock mode
					if win.img.dbg.Quantum() == govern.QuantumClock {
						imgui.Text(fmt.Sprintf("%c", fonts.Paw))
						imgui.SameLineV(0, 4)
						imgui.Text(fmt.Sprintf("%c", fonts.Paw))
						imgui.SameLineV(0, 4)
						imgui.Text(fmt.Sprintf("%c", fonts.Paw))
					}
				}
				imgui.PopStyleColor()
			}

			imgui.PopStyleColorV(5)
		} else {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			imgui.Text("no execution yet")
			imgui.Text("")
			imgui.Text("")
			imgui.PopStyleColor()
		}
	}
}

type register interface {
	Label() string
	String() string
	BitWidth() int
}

func (win *winCPU) drawRegister(reg register) {
	if reg == nil {
		return
	}

	label := reg.Label()

	imguiLabel(fmt.Sprintf("% 2s", label))
	imgui.SameLine()

	content := reg.String()
	bitwidth := reg.BitWidth()

	if imguiHexInput(fmt.Sprintf("##%s", label), bitwidth/4, &content) {
		win.img.term.pushCommand(fmt.Sprintf("CPU SET %s 0x%s", reg.Label(), content))
	}
}
