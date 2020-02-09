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
	"github.com/inkyblackness/imgui-go/v2"
)

const cpuTitle = "CPU"

type cpu struct {
	img *SdlImgui
}

func newCPU(img *SdlImgui) (*cpu, error) {
	cpu := &cpu{
		img: img,
	}

	return cpu, nil
}

// draw is called by service loop
func (cpu *cpu) draw() {
	if cpu.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{753, 92}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.SetNextWindowSizeV(imgui.Vec2{169, 142}, imgui.ConditionFirstUseEver)
		imgui.BeginV(cpuTitle, nil, imgui.WindowFlagsNoResize)

		imgui.ColumnsV(2, "", false)

		imgui.Text("PC")
		imgui.Text("A")
		imgui.Text("X")
		imgui.Text("Y")
		imgui.Text("SP")
		imgui.Text("Status")

		imgui.NextColumn()
		imgui.SetColumnWidth(-1, 100)

		imgui.Text(cpu.img.vcs.CPU.PC.String())
		imgui.Text(cpu.img.vcs.CPU.A.String())
		imgui.Text(cpu.img.vcs.CPU.X.String())
		imgui.Text(cpu.img.vcs.CPU.Y.String())
		imgui.Text(cpu.img.vcs.CPU.SP.String())
		imgui.Text(cpu.img.vcs.CPU.Status.String())

		imgui.End()
	}
}
