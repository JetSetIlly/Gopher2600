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

	// can tab pages be selected and scrolled? generally we want this to be
	// true when the emulation is paused because and false when the running.
	interactive bool

	// the program counter value in the previous (imgui) frame
	pcaddrPrevFrame uint16

	// packed colors for drawlist
	colCurrentEntryBG imgui.PackedColor
	colBreakAddress   imgui.PackedColor
	colBreakOther     imgui.PackedColor
}

func newWinDisasm(img *SdlImgui) (managedWindow, error) {
	win := &winDisasm{
		img:         img,
		interactive: false,
	}

	return win, nil
}

func (win *winDisasm) init() {
	win.colCurrentEntryBG = imgui.PackedColorFromVec4(win.img.cols.DisasmCurrEntryBG)
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

	imgui.Text(win.img.lazy.Cart.String)
	imgui.Spacing()
	imgui.Spacing()

	if win.img.lazy.Dsm != nil {
		// the value of pcaddr depends on the state of the CPU. if the
		// Final state of the CPU's last execution result is true then we
		// can be sure the PC value is valid and points to a real
		// instruction. we need this because we can never be sure when we
		// are going to draw this window
		var pcaddr uint16
		if win.img.lazy.Disasm.LastDisasmEntry == nil || win.img.lazy.Disasm.LastDisasmEntry.Result.Final {
			pcaddr = win.img.lazy.CPU.PCaddr
		} else {
			pcaddr = win.img.lazy.Disasm.LastDisasmEntry.Result.Address
		}

		// the bank that is currently selected
		currBank := win.img.lazy.Cart.CurrBank

		// sometimes a cartridge will try to run instructions from VCS RAM.
		// for presentation purposes this means that we show a "VCS RAM" tab
		nonCart := !memorymap.IsArea(pcaddr, memorymap.Cartridge)

		if win.img.lazy.Cart.NumBanks == 1 {
			// for cartridges with just one bank we don't bother with a TabBar
			win.drawBank(pcaddr, 0, true && !nonCart)
		} else {
			// create a new TabBar and iterate through the cartridge banks,
			// adding a page for each one
			imgui.BeginTabBar("")
			for b := 0; b < win.img.lazy.Cart.NumBanks; b++ {
				// set tab flags. select the tab that represents the
				// bank currently being referenced by the VCS
				flgs := imgui.TabItemFlagsNone
				if !nonCart && !win.interactive && b == currBank {
					flgs = imgui.TabItemFlagsSetSelected
				}

				// BeginTabItem() will return true when the item is selected.
				if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
					win.drawBank(pcaddr, b, b == currBank && !nonCart)
					imgui.EndTabItem()
				}
			}

			imgui.EndTabBar()
		}

		// set interactive flag when emulation is paused and when PC address
		// has not changed since last (imgui) frame
		win.interactive = win.img.paused && pcaddr == win.pcaddrPrevFrame

		// note pc address to help set win.interactive value next (imgui) frame
		win.pcaddrPrevFrame = pcaddr

		// draw options and status line
		h := imgui.CursorPosY()

		if nonCart {
			imgui.Text("executing from VCS RAM")
		} else {
			imgui.Text("")
		}

		imgui.Checkbox("Show all", &win.showAllEntries)
		imgui.SameLine()
		imgui.Checkbox("Show Bytecode", &win.showByteCode)

		win.optionsHeight = imgui.CursorPosY() - h
	}

	imgui.End()
}

func (win *winDisasm) drawBank(pcaddr uint16, b int, selected bool) {
	height := imgui.WindowHeight() - imgui.CursorPosY() - win.optionsHeight - 8
	imgui.BeginChildV(fmt.Sprintf("bank %d", b), imgui.Vec2{X: 0, Y: height}, false, 0)

	// Bless entry to make sure we can see it in the disassembly window. see
	// commenatry in debuuger/inputloop.go about why we're doing this here.
	win.img.lazy.Dsm.BlessEntry(b, pcaddr)

	var itr *disassembly.Iterate
	var count int
	if win.showAllEntries {
		itr, count, _ = win.img.lazy.Dsm.NewIteration(disassembly.EntryLevelDecoded, b)
	} else {
		itr, count, _ = win.img.lazy.Dsm.NewIteration(disassembly.EntryLevelBlessed, b)
	}

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(count)
	for clipper.Step() {
		e := itr.Start()
		e = itr.SkipNext(clipper.DisplayStart)

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			// if address value of current disasm entry and current PC value
			// match then highlight the entry
			win.drawEntry(e, selected && e.Result.Address&memorymap.AddressMaskCart == pcaddr&memorymap.AddressMaskCart)

			e = itr.Next()
			if e == nil {
				break // clipper for loop
			}
		}
	}

	// if emulation is running then centre on the current program counter. this
	// takes a bit of effort because we're using the ListClipper system. if we
	// weren't we could just use SetScrollY() at the appropriate point.
	//
	// we might be able to fold this into the loop above but this is clearer
	// and has little impact on performance (the performance issue solved by
	// ListClipper is due to invisible calls to imgui.Text() etc)
	if !win.interactive {

		// walk through disassembly and note the count for the current entry
		hlEntry := float32(0.0)
		i := float32(0.0)
		for e := itr.Start(); e != nil; e = itr.Next() {
			if e.Result.Address&memorymap.AddressMaskCart == pcaddr&memorymap.AddressMaskCart {
				hlEntry = i
				break // for loop
			}
			i++
		}

		// calculate the pixel value of the current entry. the adjustment of 4
		// is to ensure that some preceeding entries are displayed before the
		// current entry
		h := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
		h = (hlEntry - 4) * h

		// scroll to pixel value
		imgui.SetScrollY(h)
	}

	imgui.EndChild()
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, selected bool) {
	imgui.BeginGroup()
	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}

	// highlight current disassembly entry
	if win.showAllEntries && e.Level < disassembly.EntryLevelBlessed {
		adj = imgui.Vec4{0.0, 0.0, 0.0, -0.4}
	}

	if selected {
		p1 := imgui.CursorScreenPos()
		p2 := p1
		p2.X += imgui.WindowWidth()
		p2.Y += imgui.FontSize() * 1.1
		dl := imgui.WindowDrawList()
		dl.AddRectFilled(p1, p2, win.colCurrentEntryBG)

		// make entry a bit brighter
		adj = imgui.Vec4{0.1, 0.1, 0.1, 0.0}
	}

	// add some space for the gutter. has to be something tangible so that the
	// IsItemVisible() check below has something to grab onto
	imgui.Text(" ")

	win.drawBreak(e)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress.Plus(adj))
	s := win.img.lazy.Dsm.GetField(disassembly.FldAddress, e)
	imgui.Text(s)

	if win.showByteCode {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode.Plus(adj))
		s := win.img.lazy.Dsm.GetField(disassembly.FldBytecode, e)
		imgui.Text(s)
		imgui.PopStyleColorV(1)
	}

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmMnemonic.Plus(adj))
	s = win.img.lazy.Dsm.GetField(disassembly.FldMnemonic, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand.Plus(adj))
	s = win.img.lazy.Dsm.GetField(disassembly.FldOperand, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles.Plus(adj))
	s = win.img.lazy.Dsm.GetField(disassembly.FldDefnCycles, e)
	imgui.Text(s)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes.Plus(adj))
	s = win.img.lazy.Dsm.GetField(disassembly.FldDefnNotes, e)
	imgui.Text(s)

	imgui.PopStyleColorV(5)

	imgui.EndGroup()

	// the following Is*() conditions apply to the whole group

	// on right mouse button, set interactive to false. if emulation is not
	// running, it will be true for only one (imgui) frame but that is enough
	// to cause the scroller to center on the current entry.
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.interactive = false
	}

	// double click toggles a PC breakpoint on the entries address
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDoubleClicked(0) {
		win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.Dbg.TogglePCBreak(e) })
	}
}

func (win *winDisasm) drawBreak(e *disassembly.Entry) {
	switch win.img.lazy.HasBreak(e) {
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
