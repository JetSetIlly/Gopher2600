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
	"gopher2600/disassembly"

	"github.com/inkyblackness/imgui-go/v2"
)

const disasmTitle = "Disassembly"

type disasm struct {
	img      *SdlImgui
	followPC bool
}

func newDisasm(img *SdlImgui) (*disasm, error) {
	disasm := &disasm{
		img:      img,
		followPC: true,
	}

	return disasm, nil
}

// draw is called by service loop
func (disasm *disasm) draw() {
	if disasm.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{174, 204}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.SetNextWindowSizeV(imgui.Vec2{354, 387}, imgui.ConditionFirstUseEver)
		imgui.BeginV(disasmTitle, nil, 0)

		imgui.Text(disasm.img.vcs.Mem.Cart.String())
		imgui.Spacing()
		imgui.Spacing()

		if disasm.img.dsm != nil {
			// we reference the PC value often
			pcAddr := disasm.img.vcs.CPU.PC.Value()
			currBank := disasm.img.vcs.Mem.Cart.GetBank(pcAddr)

			// ee, _ := disasm.img.disasm.Get(currBank, pcAddr)
			// if ee != nil && !ee.Flow {
			// 	fmt.Printf("%d %#04x\n", currBank, pcAddr)
			// }

			imgui.BeginTabBar("banks")
			for b := range disasm.img.dsm.Entries {

				// set tab flags. select the tab thar represents the
				// bank currently being referenced by the VCS
				flgs := imgui.TabItemFlagsNone
				if disasm.followPC && b == currBank {
					flgs = imgui.TabItemFlagsSetSelected
				}

				if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
					imgui.BeginChild(fmt.Sprintf("bank %d", b))

					for a := range disasm.img.dsm.Entries[b] {
						e := disasm.img.dsm.Entries[b][a]

						if e.ReferenceResult.Address == pcAddr {
							imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.9, 0.7, 0.3, 1.0})

							imgui.Text(">")
							imgui.SameLine()
							disasm.drawEntry(e)

							imgui.PopStyleColor()

							// if emulation is running then centre on the current
							// program counter
							if !disasm.img.paused || disasm.followPC {
								imgui.SetScrollHereY(0.5)
							}
						} else {
							imgui.Text(" ")
							imgui.SameLine()
							disasm.drawEntry(e)
						}

					}

					imgui.EndChild()
					imgui.EndTabItem()
				}
			}

			imgui.EndTabBar()
		}

		imgui.End()
	}

	disasm.followPC = !disasm.img.paused
}

func (disasm *disasm) drawEntry(e *disassembly.Entry) {
	s := disasm.img.dsm.GetField(disassembly.Address, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.Mnemonic, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.Operand, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.Cycles, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.Notes, e)
	imgui.Text(s)
}
