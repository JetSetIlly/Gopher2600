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
	"gopher2600/hardware/cpu/registers"
	"strconv"
	"strings"

	"github.com/inkyblackness/imgui-go/v2"
)

const winCPUTitle = "CPU"

type winCPU struct {
	windowManagement
	img *SdlImgui

	pc       string
	a        string
	x        string
	y        string
	sp       string
	regWidth float32
}

func newWinCPU(img *SdlImgui) (managedWindow, error) {
	win := &winCPU{
		img: img,
	}

	return win, nil
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

	win.regWidth = minFrameDimension("FFFF").X

	imgui.SetNextWindowPosV(imgui.Vec2{632, 46}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCPUTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.BeginGroup()
	win.drawRegister(win.img.vcs.CPU.PC, &win.pc, win.regWidth)
	win.drawRegister(win.img.vcs.CPU.A, &win.a, win.regWidth)
	win.drawRegister(win.img.vcs.CPU.X, &win.x, win.regWidth)
	win.drawRegister(win.img.vcs.CPU.Y, &win.y, win.regWidth)
	win.drawRegister(win.img.vcs.CPU.SP, &win.sp, win.regWidth)
	imgui.EndGroup()

	imgui.SameLine()
	imgui.BeginGroup()

	win.drawLastResult()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	win.drawRDYFlag()

	imgui.EndGroup()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	win.drawStatusRegister()

	imgui.End()
}

func (win *winCPU) drawStatusRegister() {
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.Sign, "S")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.Overflow, "O")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.Break, "B")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.DecimalMode, "D")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.InterruptDisable, "I")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.Zero, "Z")
	imgui.SameLine()
	win.drawStatusRegisterBit(&win.img.vcs.CPU.Status.Carry, "C")
}

func (win *winCPU) drawStatusRegisterBit(bit *bool, label string) {
	if *bit {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPUStatusOn)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPUStatusOnHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPUStatusOnActive)
		label = strings.ToUpper(label)
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPUStatusOff)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPUStatusOffHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPUStatusOffActive)
		label = strings.ToLower(label)
	}

	if imgui.Button(label) {
		*bit = !*bit
	}

	imgui.PopStyleColorV(3)
}

func (win *winCPU) drawRegister(reg registers.Generic, s *string, regWidth float32) {
	imgui.AlignTextToFramePadding()
	imgui.Text(fmt.Sprintf("% 2s", reg.Label()))
	imgui.SameLine()

	*s = reg.String()

	cb := func(d imgui.InputTextCallbackData) int32 {
		return win.hex8Bit(reg.BitWidth()/4, d)
	}

	imgui.PushItemWidth(regWidth)
	if imgui.InputTextV(fmt.Sprintf("##%s", reg.Label()), s,
		imgui.InputTextFlagsCharsHexadecimal|imgui.InputTextFlagsCallbackAlways, cb) {
		if v, err := strconv.ParseUint(*s, 16, reg.BitWidth()); err == nil {
			reg.LoadFromUint64(v)
		}
		*s = reg.String()
	}
	imgui.PopItemWidth()
}

func (win *winCPU) drawLastResult() {
	if !win.img.vcs.CPU.HasReset() {
		e, _ := win.img.dsm.FormatResult(win.img.vcs.CPU.LastResult)
		if e.Result.Final {
			imgui.Text(fmt.Sprintf("%s", e.Bytecode))
			imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
			imgui.Text(fmt.Sprintf("%s cyc.", e.ActualCycles))
			imgui.Text("")
		} else {
			// currently, the order of how the decoding appears is not quite
			// correct. it's correct as far as the emulation is concerned but
			// it might be misleading
			//
			// !!TODO: fix cpu.go so that video callbacks are delayed until
			// after setting of LastResult fields
			imgui.Text(fmt.Sprintf("%s", e.Bytecode))
			imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
			if e.Result.Defn != nil {
				imgui.Text(fmt.Sprintf("%s cyc.", e.ActualCycles))
				imgui.Text(fmt.Sprintf("of exp. %s", e.DefnCycles))
			} else {
				imgui.Text("")
				imgui.Text("")
			}
		}
	} else {
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
		imgui.Text("")
	}
}

func (win *winCPU) drawRDYFlag() {
	imgui.AlignTextToFramePadding()
	imgui.Text("Ready (pin6)")
	imgui.SameLine()
	if win.img.vcs.CPU.RdyFlg {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPURdyFlagOn)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPURdyFlagOn)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPURdyFlagOn)
		imgui.Button(" ")
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.CPURdyFlagOff)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.CPURdyFlagOff)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.CPURdyFlagOff)
		imgui.Button(" ")
	}
	imgui.PopStyleColorV(3)
}

func (win *winCPU) hex8Bit(nibbles int, d imgui.InputTextCallbackData) int32 {
	s := string(d.Buffer())

	// restrict length of input to two characters
	// -- note that restriction to hexadecimal characters is handled by the
	// imgui.InputTextFlagsCharsHexadecimal given to InputTextV()
	if len(s) > nibbles {
		d.DeleteBytes(0, len(s))
		s = s[:nibbles]
		d.InsertBytes(0, []byte(s))
		d.MarkBufferModified()
	}

	return 0
}
