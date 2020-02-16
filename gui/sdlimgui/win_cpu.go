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

const cpuTitle = "CPU"

type cpu struct {
	windowManagement
	img *SdlImgui

	pc string
	a  string
	x  string
	y  string
	sp string
}

func newCPU(img *SdlImgui) (managedWindow, error) {
	cpu := &cpu{
		img: img,
	}

	return cpu, nil
}

func (cpu *cpu) destroy() {
}

func (cpu *cpu) id() string {
	return cpuTitle
}

func (cpu *cpu) draw() {
	if !cpu.open {
		return
	}

	inputWidth := minFrameDimension("FFFF").X

	imgui.SetNextWindowPosV(imgui.Vec2{632, 46}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(cpuTitle, &cpu.open, imgui.WindowFlagsAlwaysAutoResize)

	cpu.drawRegister(cpu.img.vcs.CPU.PC, &cpu.pc, inputWidth)
	cpu.drawRegister(cpu.img.vcs.CPU.A, &cpu.a, inputWidth)
	cpu.drawRegister(cpu.img.vcs.CPU.X, &cpu.x, inputWidth)
	cpu.drawRegister(cpu.img.vcs.CPU.Y, &cpu.y, inputWidth)
	cpu.drawRegister(cpu.img.vcs.CPU.SP, &cpu.sp, inputWidth)

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	cpu.drawStatusRegister()

	imgui.End()
}

func (cpu *cpu) drawStatusRegister() {
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.Sign, "S")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.Overflow, "O")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.Break, "B")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.DecimalMode, "D")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.InterruptDisable, "I")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.Zero, "Z")
	imgui.SameLine()
	cpu.drawStatusRegisterBit(&cpu.img.vcs.CPU.Status.Carry, "C")
}

func (cpu *cpu) drawStatusRegisterBit(bit *bool, label string) {
	if *bit {
		imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{0.73, 0.49, 0.14, 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{0.79, 0.54, 0.15, 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonActive, imgui.Vec4{0.79, 0.54, 0.15, 1.0})
		label = strings.ToUpper(label)
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{0.64, 0.40, 0.09, 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{0.70, 0.45, 0.10, 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonActive, imgui.Vec4{0.70, 0.45, 0.10, 1.0})
		label = strings.ToLower(label)
	}

	if imgui.Button(label) {
		*bit = !*bit
	}

	imgui.PopStyleColorV(3)
}

func (cpu *cpu) drawRegister(reg registers.Generic, s *string, inputWidth float32) {
	imgui.AlignTextToFramePadding()
	imgui.Text(fmt.Sprintf("% 2s", reg.Label()))
	imgui.SameLine()

	if !cpu.img.paused {
		*s = reg.String()
	}

	cb := func(d imgui.InputTextCallbackData) int32 {
		return cpu.hex8Bit(reg.BitWidth()/4, d)
	}

	imgui.PushItemWidth(inputWidth)
	if imgui.InputTextV(fmt.Sprintf("##%s", reg.Label()), s,
		imgui.InputTextFlagsCharsHexadecimal|imgui.InputTextFlagsCallbackAlways, cb) {
		if v, err := strconv.ParseUint(*s, 16, reg.BitWidth()); err == nil {
			reg.LoadFromUint64(v)
		}
		*s = reg.String()
	}
	imgui.PopItemWidth()
}

func (cpu *cpu) hex8Bit(nibbles int, d imgui.InputTextCallbackData) int32 {
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
