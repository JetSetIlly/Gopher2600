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

package sdlimgui

import (
	"fmt"
	"strings"

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

	// depending on how big the disassembly is and where in the list the PC is
	// currently, it may take a couple of frames before the PC is aligned. the
	// hasAlignedOnPC variable keeps track of this. alignOnPC will not be
	// unset, once it has been set, so long as hasAlignedOnPC is true
	hasAlignedOnPC bool

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
	cpuStep := win.img.lz.Debugger.LastResult.Result.Final
	if cpuStep {
		pcaddr = win.img.lz.CPU.PC.Address()
	} else {
		// note that we're using LastResult straight from the CPU not the
		// copy in debugger.LastDisasmEntry. the latter gets updated too
		// late for our needs
		pcaddr = win.img.lz.Debugger.LastResult.Result.Address
		currBank = win.img.lz.Debugger.LastResult.Bank
	}

	if win.img.lz.Cart.NumBanks <= 1 {
		// for cartridges with just one bank we don't bother with a TabBar
		win.drawBank(pcaddr, 0, !currBank.NonCart, cpuStep)
	} else {
		// create a new TabBar and iterate through the cartridge banks,
		// adding a page for each one
		imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)

		citr := win.img.lz.Dbg.Disasm.NewCartIteration()
		citr.Start()
		for b, ok := citr.Start(); ok; b, ok = citr.Next() {
			// set tab flags. select the tab that represents the
			// bank currently being referenced by the VCS
			flgs := imgui.TabItemFlagsNone
			if !currBank.NonCart && win.alignOnPC && b == currBank.Number {
				flgs = imgui.TabItemFlagsSetSelected
			}

			// BeginTabItem() will return true when the item is selected.
			if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
				win.drawBank(pcaddr, b, b == currBank.Number && !currBank.NonCart, cpuStep)
				imgui.EndTabItem()
			}
		}

		imgui.EndTabBar()
	}

	// set alignOnPC flag when PC address has not changed since last (imgui) frame
	win.alignOnPC = pcaddr != win.pcaddrPrevFrame || !win.hasAlignedOnPC

	// note pc address to help set win.alignOnPC value next (imgui) frame
	win.pcaddrPrevFrame = pcaddr

	// draw options and status line. start height measurement
	optionsHeight := imgui.CursorPosY()

	// status line
	s := strings.Builder{}
	if currBank.NonCart {
		s.WriteString("execution in VCS RAM")
	} else if currBank.IsRAM {
		s.WriteString("execution in cartridge RAM")
	}
	imgui.Text(s.String())

	// options line
	if imgui.Checkbox("Show all", &win.showAllEntries) {
		win.alignOnOtherAddress = true
		win.alignAddress = win.addressTopList
		win.alignOnPC = true
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

	// unset hasAlignedOnPC if alignOnPC has been set (either by
	// pcaddrPrevFramee moving on of the "Goto PC" button being pressed)
	win.hasAlignedOnPC = !win.alignOnPC
}

// draw a bank for each tabitem in the tab bar. if there is only one bank then
// drawBank() is called once.
func (win *winDisasm) drawBank(pcaddr uint16, b int, selected bool, cpuStep bool) {
	lvl := disassembly.EntryLevelBlessed
	if win.showAllEntries {
		lvl = disassembly.EntryLevelDecoded
	}
	bitr, err := win.img.lz.Dbg.Disasm.NewBankIteration(lvl, b)

	// check that NewBankIteration has succeeded. if it hasn't it probably
	// means the cart has changed in the middle of the draw routine. but that's
	// okay, we only have to wait one frame before we draw again
	if err != nil {
		return
	}

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", b), imgui.Vec2{X: 0, Y: height}, false, 0)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(bitr.EntryCount + bitr.LabelCount)
	for clipper.Step() {
		_, _ = bitr.Start()
		_, e := bitr.SkipNext(clipper.DisplayStart, true)

		// note address of top entry in the list. we use this to help
		// list alignment
		if e == nil {
			break // clipper.Step() loop
		}

		win.addressTopList = e.Result.Address

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			// try to draw address labely. if successful then advance clipper
			// counter and check for end of display
			if win.drawLabel(e) {
				i++
				if i >= clipper.DisplayEnd {
					break // clipper.DisplayStart loop
				}
			}

			// if address value of current disasm entry and current PC value
			// match then highlight the entry
			win.drawEntry(e, pcaddr, selected, cpuStep)

			_, e = bitr.Next()
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

			// we have centred on the PC. the alignOnPC will be unset next
			// frame (so long as the the CPU hasn't moved on)
			win.hasAlignedOnPC = true
		}

		// walk through disassembly and note the count for the current entry
		hlEntry := float32(0.0)
		i := float32(0.0)
		for _, e := bitr.Start(); e != nil; _, e = bitr.Next() {
			if e.Result.Address&memorymap.CartridgeBits == addr&memorymap.CartridgeBits {
				hlEntry = i
				break // for loop
			}

			// make sure to count labels
			if e.Label.String() != "" {
				i++
			}

			i++
		}

		// calculate the pixel value of the current entry. the adjustment of 4
		// is to ensure that some preceding entries are displayed before the
		// current entry
		h := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
		h = (hlEntry - scrollMargin) * h

		// scroll to pixel value
		imgui.SetScrollY(h)
	}

	imgui.EndChild()
}

func (win *winDisasm) drawLabel(e *disassembly.Entry) bool {
	s := e.GetField(disassembly.FldLabel)
	if s == "" {
		return false
	}
	imgui.Text(s)
	return true
}

// drawEntry() is called many times from drawBank(), once for each entry in the list.
func (win *winDisasm) drawEntry(e *disassembly.Entry, pcaddr uint16, selected bool, cpuStep bool) {
	imgui.BeginGroup()
	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}

	// highlight current disassembly entry
	if win.showAllEntries && e.Level < disassembly.EntryLevelBlessed {
		adj = imgui.Vec4{0.0, 0.0, 0.0, -0.4}
	}

	// if the entry is being drawn by a selected bank then highlight the entry
	// for the current pc address
	if selected && pcaddr&memorymap.CartridgeBits == e.Result.Address&memorymap.CartridgeBits {
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
	s := e.GetField(disassembly.FldAddress)
	imgui.Text(s)

	if win.showByteCode {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode.Plus(adj))
		s := e.GetField(disassembly.FldBytecode)
		imgui.Text(s)
		imgui.PopStyleColorV(1)
	}

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmMnemonic.Plus(adj))
	s = e.GetField(disassembly.FldMnemonic)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand.Plus(adj))
	s = e.GetField(disassembly.FldOperand)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles.Plus(adj))
	s = e.GetField(disassembly.FldDefnCycles)
	imgui.Text(s)

	imgui.PopStyleColorV(4)

	if e.Level == disassembly.EntryLevelExecuted {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes.Plus(adj))
		s = e.GetField(disassembly.FldActualNotes)
		imgui.Text(s)
		imgui.PopStyleColor()
	}

	imgui.EndGroup()

	// the following Is*() conditions apply to the whole group

	// on right mouse button, set interactive to false. if emulation is not
	// running, it will be true for only one (imgui) frame but that is enough
	// to cause the scroller to centre on the current entry.
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.alignOnPC = true
	}

	// single click toggles a PC breakpoint on the entries address
	if imgui.IsItemClicked() {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.TogglePCBreak(e) })
	}
}

func (win *winDisasm) drawBreak(e *disassembly.Entry) {
	switch win.img.lz.Breakpoints.HasBreak(e) {
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
