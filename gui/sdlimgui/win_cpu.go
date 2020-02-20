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
		b := string(d.Buffer())
		nibbles := reg.BitWidth() / 4

		// restrict length of input to two characters. note that restriction to
		// hexadecimal characters is handled by imgui's CharsHexadecimal flag
		// given to InputTextV()
		if len(b) > nibbles {
			d.DeleteBytes(0, len(b))
			b = b[:nibbles]
			d.InsertBytes(0, []byte(b))
			d.MarkBufferModified()
		}

		return 0
	}

	// flags used with InputTextV()
	flags := imgui.InputTextFlagsCharsHexadecimal |
		imgui.InputTextFlagsCallbackAlways |
		imgui.InputTextFlagsAutoSelectAll

	// if emulator is not paused, the values entered in the TextInput box will
	// be loaded into the register immediately and not just when the enter
	// key is pressed.
	if !win.img.paused {
		flags |= imgui.InputTextFlagsEnterReturnsTrue
	}

	imgui.PushItemWidth(regWidth)
	if imgui.InputTextV(fmt.Sprintf("##%s", reg.Label()), s, flags, cb) {
		if v, err := strconv.ParseUint(*s, 16, reg.BitWidth()); err == nil {
			reg.LoadFromUint64(v)
		}
		*s = reg.String()
	}
	imgui.PopItemWidth()
}

func (win *winCPU) drawLastResult() {
	if !win.img.vcs.CPU.HasReset() {
		// there's a danger of race-condition failure here. when in full flow
		// it might be possible for LastResult to be in a state which will
		// trigger a panic in FormatResult.
		//
		// this is because the GUI can draw at anytime and not just at the
		// CPU's endCycle() points. solutions:
		//
		// 1) semaphore locks (heavy-handed)
		// 2) as above but only when a debugger is attached (maybe. requires an
		//          extra setting on the cpu. additional checks will be ugly).
		// 3) the debugger formats the result when it's safe and we display
		//          that here, rather than calling FormatResult ourselves
		//          (slow. FormatResult() will be called far more frequently than
		//          necessary)
		// 4) build FormatResult() so that the race condition is impossible.
		//          this will requires more careful setting of LastResult
		//          fields in the CPU too.
		// 5) flag in FormatResult() that says it's safe to process. almost
		//          like a semaphore but not as heavy handed. additional code
		//          in CPU.
		//
		// option 4 is the best but the trickiest to get right. the best way to
		// get it right it is to err on the side of caution. any false
		// negatives will be fleeting and will resolve itself when (a) the CPU
		// moves on to the next phase; or (b) when the emulation pauses, even
		// mid CPU-instruction.
		res := win.img.vcs.CPU.LastResult
		e, _ := win.img.dsm.FormatResult(res)
		if e.Result.Final {
			imgui.Text(fmt.Sprintf("%s", e.Bytecode))
			imgui.Text(fmt.Sprintf("%s %s", e.Mnemonic, e.Operand))
			imgui.Text(fmt.Sprintf("%s cyc.", e.ActualCycles))
			imgui.Text("")
		} else {
			// if there's a problem with the accuracy of what is being
			// displayed, the problem probably isn't here and it probably isn't
			// a problem with the actual CPU emulation. the problem is probably
			// with how and when the CPU is populating the LastResult value.
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
