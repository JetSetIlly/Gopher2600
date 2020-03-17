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

	// packed colors for drawlist
	colCurrentEntryBg imgui.PackedColor
	colBreakAddress   imgui.PackedColor
	colBreakOther     imgui.PackedColor
}

func newWinDisasm(img *SdlImgui) (managedWindow, error) {
	win := &winDisasm{
		img:      img,
		followPC: true,
	}

	return win, nil
}

func (win *winDisasm) init() {
	win.colCurrentEntryBg = imgui.PackedColorFromVec4(win.img.cols.DisasmCurrHighlight)
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

	imgui.Text(win.img.lazy.Cart.String)
	imgui.Spacing()
	imgui.Spacing()

	if win.img.dsm != nil {
		// the value of pcAddr depends on the state of the CPU. if the
		// Final state of the CPU's last execution result is true then we
		// can be sure the PC value is valid and points to a real
		// instruction. we need this because we can never be sure when we
		// are going to draw this window
		var pcAddr uint16
		if win.img.lazy.CPU.LastResult.Final {
			pcAddr = win.img.lazy.CPU.PCaddr
		} else {
			pcAddr = win.img.lazy.CPU.LastResult.Address
		}

		currBank := win.img.lazy.Cart.CurrBank

		if win.img.lazy.Cart.NumBanks == 1 {
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

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(win.img.dsm.Counts[b][disassembly.EntryTypeDecode])
	for clipper.Step() {
		e := itr.Start()
		e = itr.SkipNext(clipper.DisplayStart)

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			// if address value of current disasm entry and current PC value
			// match then highlight the entry
			win.drawEntry(e, selected &&
				e.Result.Address&memorymap.AddressMaskCart == pcAddr&memorymap.AddressMaskCart)

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
	if win.followPC {

		// walk through disassembly and note the count for the current entry
		hlEntry := float32(0.0)
		i := float32(0.0)
		for e := itr.Start(); e != nil; e = itr.Next() {
			if e.Result.Address&memorymap.AddressMaskCart == pcAddr&memorymap.AddressMaskCart {
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

	// highlight current disassembly entry
	adj := imgui.Vec4{0.0, 0.0, 0.0, 0.0}
	if selected {
		p1 := imgui.CursorScreenPos()
		p2 := p1
		p2.X += imgui.WindowWidth()
		p2.Y += imgui.FontSize() * 1.1
		dl := imgui.WindowDrawList()
		dl.AddRectFilled(p1, p2, win.colCurrentEntryBg)

		// make entry a bit brighter
		adj = imgui.Vec4{0.1, 0.1, 0.1, 0.0}
	}

	// add some space for the gutter. has to be something tangible so that the
	// IsItemVisible() check below has something to grab onto
	imgui.Text(" ")

	win.drawBreak(e)

	imgui.SameLine()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress.Plus(adj))
	s := win.img.dsm.GetField(disassembly.FldAddress, e)
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

	// the following Is*() conditions apply to the whole group

	// on right mouse button, set followPC to true. if emulation is not
	// running, it will be true for only one frame but that is enough to cause
	// the scroller to center on the current entry.
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.followPC = true
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
