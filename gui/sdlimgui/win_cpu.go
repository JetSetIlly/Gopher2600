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
	"strings"

	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"

	"github.com/inkyblackness/imgui-go/v2"
)

const winCPUTitle = "CPU"

type winCPU struct {
	windowManagement
	img *SdlImgui
}

func newWinCPU(img *SdlImgui) (managedWindow, error) {
	win := &winCPU{
		img: img,
	}

	return win, nil
}

func (win *winCPU) init() {
}

func (win *winCPU) destroy() {
}

func (win *winCPU) id() string {
	return winCPUTitle
}

func (win *winCPU) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCPUTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	win.drawRegister(win.img.lz.CPU.PC)
	win.drawRegister(win.img.lz.CPU.A)
	win.drawRegister(win.img.lz.CPU.X)
	win.drawRegister(win.img.lz.CPU.Y)
	win.drawRegister(win.img.lz.CPU.SP)
	imgui.EndGroup()

	imgui.SameLine()
	imgui.BeginGroup()

	win.drawLastResult()

	imgui.EndGroup()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	win.drawStatusRegister()

	imgui.End()
}

func (win *winCPU) drawStatusRegister() {
	sr := win.img.lz.CPU.StatusReg

	if win.drawStatusRegisterBit(sr.Sign, "S") {
		win.img.term.pushCommand("CPU STATUS TOGGLE S")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Overflow, "O") {
		win.img.term.pushCommand("CPU STATUS TOGGLE O")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Break, "B") {
		win.img.term.pushCommand("CPU STATUS TOGGLE B")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.DecimalMode, "D") {
		win.img.term.pushCommand("CPU STATUS TOGGLE D")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.InterruptDisable, "I") {
		win.img.term.pushCommand("CPU STATUS TOGGLE I")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Zero, "Z") {
		win.img.term.pushCommand("CPU STATUS TOGGLE Z")
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Carry, "C") {
		win.img.term.pushCommand("CPU STATUS TOGGLE C")
	}

	imgui.SameLine()
	_ = imguiBooleanButton(win.img.cols, win.img.lz.CPU.RdyFlg, "RDY")
}

func (win *winCPU) drawStatusRegisterBit(bit bool, label string) bool {
	if bit {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPUStatusOn)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPUStatusOn)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPUStatusOn)
		label = strings.ToUpper(label)
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPUStatusOff)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPUStatusOff)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPUStatusOff)
		label = strings.ToLower(label)
	}

	defer imgui.PopStyleColorV(3)

	return imgui.Button(label)
}

func (win *winCPU) drawRegister(reg registers.Generic) {
	if reg == nil {
		return
	}

	label := reg.Label()

	imguiText(fmt.Sprintf("% 2s", label))
	imgui.SameLine()

	content := reg.String()
	bitwidth := reg.BitWidth()

	if imguiHexInput(fmt.Sprintf("##%s", label), !win.img.paused, bitwidth/4, &content) {
		win.img.term.pushCommand(fmt.Sprintf("CPU SET %s %s", reg.Label(), content))
	}
}

// draw most recent instruction in the CPU or as much as can be interpreted
// currently.
func (win *winCPU) drawLastResult() {
	e := win.img.lz.Debugger.LastResult

	if e.Level == disassembly.EntryLevelUnmappable {
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
		return
	}

	if e.Result.Final {
		imgui.Text(e.Bytecode)
		imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
		imgui.Text(fmt.Sprintf("%s cyc", e.Cycles))
		if win.img.lz.Cart.NumBanks == 1 {
			imgui.Text(fmt.Sprintf("(%s)", e.Address))
		} else {
			imgui.Text(fmt.Sprintf("(%s) [%s]", e.Address, e.Bank))
		}
		return
	}

	// this is not a completed CPU instruction, we're in the middle of one, so
	// we need to format the result for the partially completed instruction

	imgui.Text(e.Bytecode)
	imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
	if e.Result.Defn != nil {
		imgui.Text(fmt.Sprintf("%s of %s cyc", e.Cycles, e.DefnCycles))
		if win.img.lz.Cart.NumBanks == 1 {
			imgui.Text(fmt.Sprintf("(%s)", e.Address))
		} else {
			imgui.Text(fmt.Sprintf("(%s) [%s]", e.Address, e.Bank))
		}
	} else {
		imgui.Text("")
		imgui.Text("")
	}
}
