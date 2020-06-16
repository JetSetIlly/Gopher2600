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

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

	"github.com/inkyblackness/imgui-go/v2"
)

const winDisasmTitle = "Disassembly"

type winDisasm struct {
	windowManagement
	img *SdlImgui

	// show all entries in the cartridge, even those we're not confident about
	// being instructions.
	showAllEntries bool
	showByteCode   bool

	// height of options line at bottom of window. valid after first frame
	optionsHeight float32

	// scroll the disassembly listing so that the PC address is visible.
	// generally we want this to be true when the emulation is paused but we
	// set it when we want to force the listview to realign on the PC
	alignOnPC bool

	// it's sometimes useful to align on an arbitrary address. when
	// alignOnOtherAddress is set to true, alignAddress should also be set. we
	// use this when toggling showAllEntries flag
	alignOnOtherAddress bool
	alignAddress        uint16

	// the address of the entry a the top of the list, we use to help
	// list alignment (see alignAddress above)
	addressTopList uint16

	// the program counter value in the previous (imgui) frame
	pcaddrPrevFrame uint16

	// packed colors for drawlist
	colCPUstep      imgui.PackedColor
	colVideoStep    imgui.PackedColor
	colBreakAddress imgui.PackedColor
	colBreakOther   imgui.PackedColor
}

func newWinDisasm(img *SdlImgui) (managedWindow, error) {
	win := &winDisasm{
		img:       img,
		alignOnPC: false,
	}

	return win, nil
}

func (win *winDisasm) init() {
	win.colCPUstep = imgui.PackedColorFromVec4(win.img.cols.DisasmCPUstep)
	win.colVideoStep = imgui.PackedColorFromVec4(win.img.cols.DisasmVideoStep)
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

	imgui.SetNextWindowPosV(imgui.Vec2{905, 242}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winDisasmTitle, &win.open, 0)

	imgui.Text(win.img.lz.Cart.Summary)
	imgui.Spacing()
	imgui.Spacing()

	// the bank that is currently selected
	currBank := win.img.lz.Cart.CurrBank

	// the value of pcAddr depends on the state of the CPU. if the
	// Final state of the CPU's last execution result is true then we
	// can be sure the PC value is valid and points to a real
	// instruction. we need this because we can never be sure when we
	// are going to draw this window
	var pcaddr uint16
	cpuStep := win.img.lz.CPU.LastResult.Final
	if cpuStep {
		pcaddr = win.img.lz.CPU.PCaddr
	} else {
		// note that we're using LastResult straight from the CPU not the
		// copy in debugger.LastDisasmEntry. the latter gets updated too
		// late for our needs
		pcaddr = win.img.lz.CPU.LastResult.Address
	}

	// sometimes a cartridge will try to run instructions from VCS RAM.
	// for presentation purposes this means that we show a "VCS RAM" tab
	nonCart := !memorymap.IsArea(pcaddr, memorymap.Cartridge)

	if win.img.lz.Cart.NumBanks == 1 {
		// for cartridges with just one bank we don't bother with a TabBar
		win.drawBank(pcaddr, 0, !nonCart, cpuStep)

	} else {
		// create a new TabBar and iterate through the cartridge banks,
		// adding a page for each one
		imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)

		for b := 0; b < win.img.lz.Cart.NumBanks; b++ {
			// set tab flags. select the tab that represents the
			// bank currently being referenced by the VCS
			flgs := imgui.TabItemFlagsNone
			if !nonCart && win.alignOnPC && b == currBank {
				flgs = imgui.TabItemFlagsSetSelected
			}

			// BeginTabItem() will return true when the item is selected.
			if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
				win.drawBank(pcaddr, b, b == currBank && !nonCart, cpuStep)
				imgui.EndTabItem()
			}
		}

		imgui.EndTabBar()
	}

	// set alignOnPC flag when PC address has not changed since last
	// (imgui) frame
	win.alignOnPC = pcaddr != win.pcaddrPrevFrame

	// note pc address to help set win.alignOnPC value next (imgui) frame
	win.pcaddrPrevFrame = pcaddr

	// draw options and status line. start height measurement
	optionsHeight := imgui.CursorPosY()

	// status line
	if nonCart {
		imgui.Text("executing from VCS RAM")
	} else {
		imgui.Text("")
	}

	// options line
	if imgui.Checkbox("Show all", &win.showAllEntries) {
		win.alignOnOtherAddress = true
		win.alignAddress = win.addressTopList
	}

	imgui.SameLine()
	imgui.Checkbox("Show Bytecode", &win.showByteCode)

	imgui.SameLine()
	if imgui.Button("Goto PC") {
		win.alignOnPC = true
	}

	// commit height measurement
	win.optionsHeight = imgui.CursorPosY() - optionsHeight

	imgui.End()
}

// draw a bank for each tabitem in the tab bar. if there is only one bank then
// drawBank() is called once
func (win *winDisasm) drawBank(pcaddr uint16, b int, selected bool, cpuStep bool) {
	lvl := disassembly.EntryLevelBlessed
	if win.showAllEntries {
		lvl = disassembly.EntryLevelDecoded
	}
	itr, count, err := win.img.lz.Dbg.Disasm.NewIteration(lvl, b)

	// check that NewIteration has succeeded. if it hasn't it probably means
	// the cart has changed in the middle of the draw routine. but that's okay,
	// we only have to wait one frame before we draw again
	if err != nil {
		return
	}

	height := imgui.WindowHeight() - imgui.CursorPosY() - win.optionsHeight - imgui.CurrentStyle().FramePadding().Y*2 - imgui.CurrentStyle().ItemInnerSpacing().Y
	imgui.BeginChildV(fmt.Sprintf("bank %d", b), imgui.Vec2{X: 0, Y: height}, false, 0)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(count)
	for clipper.Step() {
		e := itr.Start()
		e = itr.SkipNext(clipper.DisplayStart)

		// note address of top entry in the list. we use this to help
		// list alignment
		if e == nil {
			break // clipper.Step() loop
		}

		win.addressTopList = e.Result.Address

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			// if address value of current disasm entry and current PC value
			// match then highlight the entry
			win.drawEntry(e, pcaddr, selected, cpuStep)

			e = itr.Next()
			if e == nil {
				break // clipper.DisplayStart loop
			}
		}
	}

	// align on a specific entry if a alignOnPC or alignOnOtherAddress is
	// set. for clarity we're doing this outside of the clipper loop above
	//
	// note that alignOnPC has an additional condition and will only be
	// honoured if the selected flag is set. this is to prevent alignment
	// attempts when the PC is executing in VCS RAM
	if (win.alignOnPC && selected) || win.alignOnOtherAddress {
		var addr uint16
		var scrollMargin float32

		// figure out what kind of alignment to perform. aligning on non-PC
		// address takes priority
		if win.alignOnOtherAddress {
			addr = win.alignAddress
			scrollMargin = 0

			// reset alignOnOtherAddress flag immediately after use
			win.alignOnOtherAddress = false
		} else {
			addr = pcaddr
			scrollMargin = 4
		}

		// walk through disassembly and note the count for the current entry
		hlEntry := float32(0.0)
		i := float32(0.0)
		for e := itr.Start(); e != nil; e = itr.Next() {
			if e.Result.Address&memorymap.AddressMaskCart == addr&memorymap.AddressMaskCart {
				hlEntry = i
				break // for loop
			}
			i++
		}

		// calculate the pixel value of the current entry. the adjustment of 4
		// is to ensure that some preceeding entries are displayed before the
		// current entry
		h := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
		h = (hlEntry - scrollMargin) * h

		// scroll to pixel value
		imgui.SetScrollY(h)
	}

	imgui.EndChild()
}

// drawEntry() is called many times from drawBank(), once for each entry in the list
func (win *winDisasm) drawEntry(e *disassembly.Entry, pcaddr uint16, selected bool, cpuStep bool) {
	imgui.BeginGroup()
	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}

	// highlight current disassembly entry
	if win.showAllEntries && e.Level < disassembly.EntryLevelBlessed {
		adj = imgui.Vec4{0.0, 0.0, 0.0, -0.4}
	}

	// if the entry is being drawn by a selected bank then highlight the entry
	// for the current pc address
	if selected && pcaddr&memorymap.AddressMaskCart == e.Result.Address&memorymap.AddressMaskCart {
		p1 := imgui.CursorScreenPos()
		p2 := p1
		p2.X += imgui.WindowWidth()
		p2.Y += imgui.FontSize() * 1.1
		dl := imgui.WindowDrawList()

		if cpuStep {
			dl.AddRectFilled(p1, p2, win.colCPUstep)
		} else {
			dl.AddRectFilled(p1, p2, win.colVideoStep)
		}

		// make entry a bit brighter
		adj = imgui.Vec4{0.1, 0.1, 0.1, 0.0}
	}

	// add some space for the gutter. has to be something tangible so that the
	// IsItemVisible() check below has something to grab onto
	imgui.Text(" ")

	win.drawBreak(e)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress.Plus(adj))
	s := win.img.lz.Dbg.Disasm.GetField(disassembly.FldAddress, e)
	imgui.Text(s)

	if win.showByteCode {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode.Plus(adj))
		s := win.img.lz.Dbg.Disasm.GetField(disassembly.FldBytecode, e)
		imgui.Text(s)
		imgui.PopStyleColorV(1)
	}

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmMnemonic.Plus(adj))
	s = win.img.lz.Dbg.Disasm.GetField(disassembly.FldMnemonic, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand.Plus(adj))
	s = win.img.lz.Dbg.Disasm.GetField(disassembly.FldOperand, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles.Plus(adj))
	s = win.img.lz.Dbg.Disasm.GetField(disassembly.FldDefnCycles, e)
	imgui.Text(s)

	imgui.PopStyleColorV(4)

	if e.Level == disassembly.EntryLevelExecuted {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes.Plus(adj))
		s = win.img.lz.Dbg.Disasm.GetField(disassembly.FldActualNotes, e)
		imgui.Text(s)
		imgui.PopStyleColor()
	}

	imgui.EndGroup()

	// the following Is*() conditions apply to the whole group

	// on right mouse button, set interactive to false. if emulation is not
	// running, it will be true for only one (imgui) frame but that is enough
	// to cause the scroller to center on the current entry.
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.alignOnPC = true
	}

	// single click toggles a PC breakpoint on the entries address
	//if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseClicked(0) {
	if imgui.IsItemClicked() {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.TogglePCBreak(e) })
	}
}

func (win *winDisasm) drawBreak(e *disassembly.Entry) {
	switch win.img.lz.HasBreak(e) {
	case debugger.BrkPCAddress:
		win.drawGutter(gutterSolid, win.colBreakAddress)
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
	// !!TODO: popup menu on right mouse click over disasm entry
}
