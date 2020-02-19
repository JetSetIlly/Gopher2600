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
	"gopher2600/debugger"
	"gopher2600/disassembly"
	"gopher2600/hardware/memory/memorymap"

	"github.com/inkyblackness/imgui-go/v2"
)

const winDisasmTitle = "Disassembly"

type winDisasm struct {
	windowManagement
	img      *SdlImgui
	followPC bool
}

func newWinDisasm(img *SdlImgui) (managedWindow, error) {
	win := &winDisasm{
		img:      img,
		followPC: true,
	}

	return win, nil
}

func (win *winDisasm) destroy() {
}

func (win *winDisasm) id() string {
	return winDisasmTitle
}

func (win *winDisasm) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{915, 214}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{355, 495}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winDisasmTitle, &win.open, 0)

	imgui.Text(win.img.vcs.Mem.Cart.String())
	imgui.Spacing()
	imgui.Spacing()

	if win.img.dsm != nil {
		var pcAddr uint16

		// the value of pcAddr depends on the state of the CPU. if the
		// Final state of the CPU's last execution result is true then we
		// can be sure the PC value is valid and points to a real
		// instruction. we need this because we can never be sure when we
		// are going to draw this window
		if win.img.vcs.CPU.LastResult.Final {
			pcAddr = win.img.vcs.CPU.PC.Value()
		} else {
			pcAddr = win.img.vcs.CPU.LastResult.Address
		}

		if win.img.vcs.Mem.Cart.NumBanks() == 1 {
			// for cartridges with just one bank we don't bother with a TabBar
			win.drawBank(pcAddr, 0)
		} else {
			// create a new TabBar and iterate throuhg the cartridge banks,
			// adding a new TabPage for each
			currBank := win.img.vcs.Mem.Cart.GetBank(pcAddr)
			imgui.BeginTabBar("banks")
			for b := range win.img.dsm.Entries {

				// set tab flags. select the tab that represents the
				// bank currently being referenced by the VCS
				flgs := imgui.TabItemFlagsNone
				if win.followPC && b == currBank {
					flgs = imgui.TabItemFlagsSetSelected
				}

				if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
					win.drawBank(pcAddr, b)
					imgui.EndTabItem()
				}
			}
			imgui.EndTabBar()
		}
	}

	imgui.End()

	win.followPC = !win.img.paused
}

func (win *winDisasm) drawBank(pcAddr uint16, b int) {
	imgui.BeginChild(fmt.Sprintf("bank %d", b))

	itr, _ := win.img.dsm.NewIteration(b)

	for e := itr.Start(); e != nil; e = itr.Next(disassembly.EntryTypeDecode) {

		// if address value of current disasm entry and
		// current PC value match then highlight the entry
		if e.Result.Address&memorymap.AddressMaskCart == pcAddr&memorymap.AddressMaskCart {
			win.drawEntry(e, true)

			// if emulation is running then centre on the current
			// program counter
			if win.followPC {
				imgui.SetScrollHereY(0.5)
			}
		} else {
			win.drawEntry(e, false)
		}
	}

	imgui.EndChild()
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, selected bool) {
	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}
	if selected {
		adj = win.img.cols.DisasmSelectedAdj
	}

	s := win.img.dsm.GetField(disassembly.FldAddress, e)
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress.Plus(adj))
	imgui.Text(s)

	// if item is visible then check for breakpoint
	if imgui.IsItemVisible() {
		switch win.img.dbg.HasPcBreak(e) {
		case debugger.PcBreakAnyBank:
			imgui.SameLine()
			badgeBreakpointAnyBank(win.img.cols)
		case debugger.PcBreakThisBank:
			imgui.SameLine()
			badgeBreakpointThisBank(win.img.cols)
		}
	}

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmMnemonic.Plus(adj))
	s = win.img.dsm.GetField(disassembly.FldMnemonic, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand.Plus(adj))
	s = win.img.dsm.GetField(disassembly.FldOperand, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles.Plus(adj))
	s = win.img.dsm.GetField(disassembly.FldDefnCycles, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes.Plus(adj))
	s = win.img.dsm.GetField(disassembly.FldDefnNotes, e)
	imgui.Text(s)

	imgui.PopStyleColorV(5)
}
