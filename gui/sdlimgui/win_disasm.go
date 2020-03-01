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
	img *SdlImgui

	// should tab pages be selected and scrolled. generally we want this to be
	// false when the emulation is paused because we want the user to be able
	// to scroll around and explore the window
	followPC bool

	// the selected cartridge bank in the previous frame. this is used to help
	// decide what the value of followPC should be
	bankPrevFrame int
	pcPrevFrame   uint16

	// gutter colors
	colCurrentPC    imgui.PackedColor
	colBreakAddress imgui.PackedColor
	colBreakOther   imgui.PackedColor
}

func newWinDisasm(img *SdlImgui) (managedWindow, error) {
	win := &winDisasm{
		img:      img,
		followPC: true,
	}

	return win, nil
}

func (win *winDisasm) init() {
	win.colCurrentPC = imgui.PackedColorFromVec4(win.img.cols.DisasmCurrentPC)
	win.colBreakAddress = imgui.PackedColorFromVec4(win.img.cols.DisasmBreakAddress)
	win.colBreakOther = imgui.PackedColorFromVec4(win.img.cols.DisasmBreakOther)
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
		// the value of pcAddr depends on the state of the CPU. if the
		// Final state of the CPU's last execution result is true then we
		// can be sure the PC value is valid and points to a real
		// instruction. we need this because we can never be sure when we
		// are going to draw this window
		var pcAddr uint16
		if win.img.vcs.CPU.LastResult.Final {
			pcAddr = win.img.vcs.CPU.PC.Value()
		} else {
			pcAddr = win.img.vcs.CPU.LastResult.Address
		}

		currBank := win.img.vcs.Mem.Cart.GetBank(pcAddr)

		if win.img.vcs.Mem.Cart.NumBanks() == 1 {
			// for cartridges with just one bank we don't bother with a TabBar
			win.drawBank(pcAddr, 0, true)
		} else {
			// create a new TabBar and iterate through the cartridge banks,
			// adding a page for each one
			imgui.BeginTabBar("")
			for b := range win.img.dsm.Entries {
				// set tab flags. select the tab that represents the
				// bank currently being referenced by the VCS
				flgs := imgui.TabItemFlagsNone
				if win.followPC && b == currBank {
					flgs = imgui.TabItemFlagsSetSelected
				}

				// BeginTabItem() will return true when the item is selected.
				// When the SetSelected flag is specified, it does not take
				// effect until the end of the frame and so BeginTabItem() will
				// return true *next* frame. see the setting of win.followPC
				// below for more.
				if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
					win.drawBank(pcAddr, b, b == currBank)
					imgui.EndTabItem()
				}
			}
			imgui.EndTabBar()
		}

		// if the current bank has only been selected this frame then we need
		// an extra frame to draw the tab page with drawBank() and for the page
		// to scroll to the correct position. the second part of the condition
		// below sustains followPC for the additional frame
		win.followPC = !win.img.paused || currBank != win.bankPrevFrame || pcAddr != win.pcPrevFrame

		// update bank information to help with followPC next frame
		win.bankPrevFrame = currBank
		win.pcPrevFrame = pcAddr
	}

	imgui.End()
}

func (win *winDisasm) drawBank(pcAddr uint16, b int, selected bool) {
	imgui.BeginChild(fmt.Sprintf("bank %d", b))

	itr, _ := win.img.dsm.NewIteration(disassembly.EntryTypeDecode, b)

	for e := itr.Start(); e != nil; e = itr.Next() {

		// if address value of current disasm entry and
		// current PC value match then highlight the entry
		if e.Result.Address&memorymap.AddressMaskCart == pcAddr&memorymap.AddressMaskCart {
			win.drawEntry(e, selected)

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
	imgui.BeginGroup()

	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}
	if selected {
		adj = win.img.cols.DisasmSelectedAdj
	}

	// add some space for the gutter. has to be something tangible so that the
	// IsItemVisible() check below has something to grab onto
	imgui.Text(" ")

	// draw pointer, breakpoint or nothing at all in gutter
	if selected {
		win.drawPointer()
	} else if imgui.IsItemVisible() {
		win.drawBreak(e)
	}

	imgui.SameLine()
	s := win.img.dsm.GetField(disassembly.FldAddress, e)
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress.Plus(adj))
	imgui.Text(s)

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

	imgui.EndGroup()

	// these Is*() conditions apply to the whole group
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		fmt.Println("TODO: context menu")
	}

	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDoubleClicked(0) {
		win.img.dbg.TogglePCBreak(e)
	}
}

func (win *winDisasm) drawPointer() {
	win.drawGutter(gutterSolid, win.colCurrentPC)
}

func (win *winDisasm) drawBreak(e *disassembly.Entry) {
	switch win.img.dbg.HasBreak(e) {
	case debugger.BrkPCAddress:
		win.drawGutter(gutterDotted, win.colBreakAddress)
	case debugger.BrkOther:
		win.drawGutter(gutterOutline, win.colBreakOther)
	}
}

type gutterType int

const (
	gutterOutline gutterType = iota
	gutterDotted
	gutterSolid
)

func (win *winDisasm) drawGutter(fill gutterType, col imgui.PackedColor) {
	r := imgui.FrameHeight() / 4
	p := imgui.CursorScreenPos()
	p.Y -= r * 2
	dl := imgui.WindowDrawList()
	switch fill {
	case gutterOutline:
		dl.AddCircle(p, r, col)
	case gutterDotted:
		dl.AddCircle(p, r, col)
		dl.AddCircle(p, r/2, col)
	case gutterSolid:
		dl.AddCircleFilled(p, r, col)
	}
}

func (win *winDisasm) drawPopupMenu(e *disassembly.Entry) {
}
