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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"

	"github.com/inkyblackness/imgui-go/v2"
)

const winCPUTitle = "CPU"

type winCPU struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// ready flag colors
	colFlgReadyOn  imgui.PackedColor
	colFlgReadyOff imgui.PackedColor
}

func newWinCPU(img *SdlImgui) (managedWindow, error) {
	win := &winCPU{
		img: img,
	}

	return win, nil
}

func (win *winCPU) init() {
	win.widgetDimensions.init()
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
	win.drawRegister(win.img.lz.Dbg.VCS.CPU.PC)
	win.drawRegister(win.img.lz.Dbg.VCS.CPU.A)
	win.drawRegister(win.img.lz.Dbg.VCS.CPU.X)
	win.drawRegister(win.img.lz.Dbg.VCS.CPU.Y)
	win.drawRegister(win.img.lz.Dbg.VCS.CPU.SP)
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
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.Sign = !sr.Sign })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Overflow, "O") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.Overflow = !sr.Overflow })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Break, "B") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.Break = !sr.Break })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.DecimalMode, "D") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.DecimalMode = !sr.DecimalMode })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.InterruptDisable, "I") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.InterruptDisable = !sr.InterruptDisable })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Zero, "Z") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.Zero = !sr.Zero })
	}
	imgui.SameLine()
	if win.drawStatusRegisterBit(sr.Carry, "C") {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.CPU.Status.Carry = !sr.Carry })
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
	regLabel := win.img.lz.CPU.RegLabel(reg)

	imguiText(fmt.Sprintf("% 2s", regLabel))
	imgui.SameLine()

	label := fmt.Sprintf("##%s", regLabel)
	content := win.img.lz.CPU.RegValue(reg)
	bitwidth := win.img.lz.CPU.RegBitwidth(reg)

	imgui.PushItemWidth(win.fourDigitDim.X)
	if imguiHexInput(label, !win.img.paused, bitwidth/4, &content) {
		if v, err := strconv.ParseUint(content, 16, bitwidth); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() { reg.LoadFromUint64(v) })
		}
	}
	imgui.PopItemWidth()
}

// draw most recent instruction in the CPU or as much as can be interpreted
// currently
func (win *winCPU) drawLastResult() {
	if win.img.lz.CPU.HasReset {
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
		return
	}

	e := win.img.lz.Debugger.LastResult

	if e.Result.Final {
		imgui.Text(fmt.Sprintf("%s", e.Bytecode))
		imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
		imgui.Text(fmt.Sprintf("%s cyc", e.ActualCycles))
		if win.img.lz.Cart.NumBanks == 1 {
			imgui.Text(fmt.Sprintf("(%s)", e.Address))
		} else {
			imgui.Text(fmt.Sprintf("(%s) [%s]", e.Address, e.Bank))
		}
		return
	}

	// this is not a completed CPU instruction, we're in the middle of one, so
	// we need to format the result for the partially completed instruction

	imgui.Text(fmt.Sprintf("%s", e.Bytecode))
	imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
	if e.Result.Defn != nil {
		imgui.Text(fmt.Sprintf("%s of %s cyc", e.ActualCycles, e.DefnCycles))
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
