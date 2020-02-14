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
	"gopher2600/hardware/memory/memorymap"

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
			var pcAddr uint16

			// the value of pcAddr depends on the state of the CPU. if the
			// Final state of the CPU's last execution result is true then we
			// can be sure the PC value is valid and points to a real
			// instruction. we need this because we can never be sure when we
			// are going to draw this window
			if disasm.img.vcs.CPU.LastResult.Final {
				pcAddr = disasm.img.vcs.CPU.PC.Value()
			} else {
				pcAddr = disasm.img.vcs.CPU.LastResult.Address
			}

			currBank := disasm.img.vcs.Mem.Cart.GetBank(pcAddr)

			imgui.BeginTabBar("banks")
			for b := range disasm.img.dsm.Entries {

				// set tab flags. select the tab that represents the
				// bank currently being referenced by the VCS
				flgs := imgui.TabItemFlagsNone
				if disasm.followPC && b == currBank {
					flgs = imgui.TabItemFlagsSetSelected
				}

				if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
					imgui.BeginChild(fmt.Sprintf("bank %d", b))

					itr, _ := disasm.img.dsm.NewIteration(b)

					for e := itr.Start(); e != nil; e = itr.Next(disassembly.EntryTypeDecode) {

						// if address value of current disasm entry and
						// current PC value match then highlight the entry
						if e.Result.Address&memorymap.AddressMaskCart == pcAddr&memorymap.AddressMaskCart {
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
	s := disasm.img.dsm.GetField(disassembly.FldAddress, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.FldMnemonic, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.FldOperand, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.FldDefnCycles, e)
	imgui.Text(s)
	imgui.SameLine()
	s = disasm.img.dsm.GetField(disassembly.FldDefnNotes, e)
	imgui.Text(s)
}
