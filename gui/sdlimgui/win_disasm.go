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
	"gopher2600/disassembly"

	"github.com/inkyblackness/imgui-go/v2"
)

const disasmTitle = "Disassembly"

type disasm struct {
	img *SdlImgui
}

func newDisasm(img *SdlImgui) (*disasm, error) {
	disasm := &disasm{
		img: img,
	}

	return disasm, nil
}

// draw is called by service loop
func (disasm *disasm) draw() {
	if disasm.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{174, 204}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.SetNextWindowSizeV(imgui.Vec2{354, 387}, imgui.ConditionFirstUseEver)
		imgui.BeginV(disasmTitle, nil, 0)

		activeLine := false
		if disasm.img.disasm != nil {
			for b := range disasm.img.disasm.Entries {
				for a := range disasm.img.disasm.Entries[b] {
					e := disasm.img.disasm.Entries[b][a]
					if e != nil && e.Flow {
						s := disasm.img.disasm.GetField(disassembly.Address, e)

						if e.ReferenceResult.Address == disasm.img.vcs.CPU.PC.Value() {
							imgui.SetScrollHereY(0.5)
							imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.9, 0.7, 0.3, 1.0})
							activeLine = true
						}

						imgui.Text(s)
						imgui.SameLine()
						s = disasm.img.disasm.GetField(disassembly.Mnemonic, e)
						imgui.Text(s)
						imgui.SameLine()
						s = disasm.img.disasm.GetField(disassembly.Operand, e)
						imgui.Text(s)
						imgui.SameLine()
						s = disasm.img.disasm.GetField(disassembly.Cycles, e)
						imgui.Text(s)
						imgui.SameLine()
						s = disasm.img.disasm.GetField(disassembly.Notes, e)
						imgui.Text(s)

						// if this is the active line then we have pushed a
						// color style
						if activeLine {
							imgui.PopStyleColor()
							activeLine = false
						}
					}
				}
			}
		}

		imgui.End()
	}
}
