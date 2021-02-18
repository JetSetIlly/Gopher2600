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

	"github.com/inkyblackness/imgui-go/v3"
)

const winDisasmID = "Disassembly"

type winDisasm struct {
	img  *SdlImgui
	open bool

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

	// the address of the top-most visible entry. we use to help list alignment
	// (see alignAddress above)
	addressTopList uint16

	// address of the bottom-most visible entry.
	addressBotList uint16

	// the program counter value in the previous (imgui) frame
	focusAddrPrevFrame uint16
}

func newWinDisasm(img *SdlImgui) (window, error) {
	win := &winDisasm{
		img:       img,
		alignOnPC: false,
	}

	return win, nil
}

func (win *winDisasm) init() {
}

func (win *winDisasm) id() string {
	return winDisasmID
}

func (win *winDisasm) isOpen() bool {
	return win.open
}

func (win *winDisasm) setOpen(open bool) {
	win.open = open
}

func (win *winDisasm) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{905, 242}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)
	imgui.BeginV(win.id(), &win.open, 0)

	imgui.Text(win.img.lz.Cart.Summary)
	imgui.Spacing()
	imgui.Spacing()

	// the bank that is currently selected
	currBank := win.img.lz.Cart.CurrBank

	// whether we're at CPU step boundary
	cpuStep := win.img.lz.Debugger.LastResult.Result.Final

	// the value of focusAddr depends on the state of the CPU. if the
	// Final state of the CPU's last execution result is true then we
	// can be sure the PC value is valid and points to a real
	// instruction. we need this because we can never be sure when we
	// are going to draw this window
	var focusAddr uint16

	if currBank.ExecutingCoprocessor {
		// if coprocessor is running then jam the focusAddr value at address the
		// CPU will resume from once the coprocessor has finished.
		focusAddr = currBank.CoprocessorResumeAddr
	} else {
		if cpuStep {
			focusAddr = win.img.lz.CPU.PC.Address()
		} else {
			focusAddr = win.img.lz.Debugger.LastResult.Result.Address
			currBank = win.img.lz.Debugger.LastResult.Bank
		}
	}

	if win.img.lz.Cart.NumBanks <= 1 {
		// for cartridges with just one bank we don't bother with a TabBar
		win.drawBank(focusAddr, 0, !currBank.NonCart, cpuStep)
	} else {
		// create a new TabBar and iterate through the cartridge banks,
		// adding a page for each one
		imgui.BeginTabBarV("", imgui.TabBarFlagsFittingPolicyScroll)

		bitr := win.img.lz.Dbg.Disasm.NewBanksIteration()
		bitr.Start()
		for b, ok := bitr.Start(); ok; b, ok = bitr.Next() {
			// set tab flags. select the tab that represents the
			// bank currently being referenced by the VCS
			flgs := imgui.TabItemFlagsNone
			if !currBank.NonCart && win.alignOnPC && b == currBank.Number {
				flgs = imgui.TabItemFlagsSetSelected
			}

			// BeginTabItem() will return true when the item is selected.
			if imgui.BeginTabItemV(fmt.Sprintf("%d", b), nil, flgs) {
				win.drawBank(focusAddr, b, b == currBank.Number && !currBank.NonCart, cpuStep)
				imgui.EndTabItem()
			}
		}

		imgui.EndTabBar()
	}

	// set alignOnPC flag when PC address has not changed since last (imgui) frame
	win.alignOnPC = focusAddr != win.focusAddrPrevFrame || !win.hasAlignedOnPC

	// note pc address to help set win.alignOnPC value next (imgui) frame
	win.focusAddrPrevFrame = focusAddr

	// draw options and status line. start height measurement
	win.optionsHeight = measureHeight(func() {
		// status line
		s := strings.Builder{}
		if currBank.NonCart {
			s.WriteString("executing non-cartridge addresses")
		} else if currBank.IsRAM {
			s.WriteString("executing cartridge RAM")
		} else if currBank.ExecutingCoprocessor {
			s.WriteString("executing coprocessor instructions")
		}
		imgui.Spacing()
		imgui.Text(s.String())
		imgui.Spacing()

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
	})

	imgui.End()

	// unset hasAlignedOnPC if alignOnPC has been set (either by
	// focusAddrPrevFramee moving on of the "Goto PC" button being pressed)
	win.hasAlignedOnPC = !win.alignOnPC
}

// draw a bank for each tabitem in the tab bar. if there is only one bank then
// drawBank() is called once.
func (win *winDisasm) drawBank(focusAddr uint16, b int, selected bool, cpuStep bool) {

	lvl := disassembly.EntryLevelBlessed
	if win.showAllEntries {
		lvl = disassembly.EntryLevelDecoded
	}
	eitr, err := win.img.lz.Dbg.Disasm.NewEntriesIteration(lvl, b, focusAddr)

	// check that NewBankIteration has succeeded. if it hasn't it probably
	// means the cart has changed in the middle of the draw routine. but that's
	// okay, we only have to wait one frame before we draw again
	if err != nil {
		return
	}

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", b), imgui.Vec2{X: 0, Y: height}, false, 0)
	defer imgui.EndChild()

	numColumns := 7
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	if !imgui.BeginTableV("bank", numColumns, flgs, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)
	defer imgui.PopStyleColorV(2)

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(eitr.EntryCount + eitr.LabelCount)
	for clipper.Step() {
		_, _ = eitr.Start()
		_, e := eitr.SkipNext(clipper.DisplayStart, true)

		// note address of top entry in the list. we use this to help
		// list alignment
		if e == nil {
			break // clipper.Step() loop
		}

		// note address of top-most visible entry
		win.addressTopList = e.Result.Address

		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			// try to draw address label
			s := e.Label.String()
			if len(s) > 0 {
				imgui.TableNextRow()
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.TableNextColumn()
				imgui.Text(s)

				// advance clipper counter and check for end of display
				i++
				if i >= clipper.DisplayEnd {
					break // clipper.DisplayStart loop
				}
			}

			win.drawEntry(cpuStep, selected, focusAddr, e)

			// advance clipper
			_, e = eitr.Next()
			if e == nil {
				break // clipper.DisplayStart loop
			}

			// note address of bottom-most visible entry
			win.addressBotList = e.Result.Address
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
			addr = focusAddr
			scrollMargin = 4

			// we have centred on the PC. the alignOnPC will be unset next
			// frame (so long as the the CPU hasn't moved on)
			win.hasAlignedOnPC = true
		}

		// walk through disassembly and note the count for the current entry
		hlEntry := float32(0.0)
		i := float32(0.0)
		for _, e := eitr.Start(); e != nil; _, e = eitr.Next() {
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

	// set lazy update list
	win.img.lz.Breakpoints.SetUpdateList(b, win.addressTopList, win.addressBotList)
}

func (win *winDisasm) drawEntry(cpuStep bool, selected bool, focusAddr uint16, e *disassembly.Entry) {
	imgui.TableNextRow()
	if selected && focusAddr&memorymap.CartridgeBits == e.Result.Address&memorymap.CartridgeBits {
		hi := win.img.cols.DisasmCPUstep
		if !cpuStep {
			hi = win.img.cols.DisasmVideoStep
		}
		imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, hi)
	}

	// breakpoint indicator column
	imgui.TableNextColumn()
	switch win.img.lz.Breakpoints.HasBreak(e.Result.Address) {
	case debugger.BrkPCAddress:
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
		imgui.Text("*")
		imgui.PopStyleColor()
	case debugger.BrkOther:
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakOther)
		imgui.Text("#")
		imgui.PopStyleColor()
	default:
		imgui.Text(" ")
	}

	// address column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
	imgui.Text(e.Address)
	imgui.PopStyleColor()

	// single click on the address entry toggles a PC breakpoint
	if imgui.IsItemClicked() {
		f := e // copy of pushed disasm entry
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.TogglePCBreak(f) })
	}

	// optional bytecode column
	if win.showByteCode {
		imgui.TableNextColumn()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode)
		imgui.Text(e.Bytecode)
		imgui.PopStyleColor()
	}

	// operator column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
	imgui.Text(e.Operator)
	imgui.PopStyleColor()

	// operand column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
	imgui.Text(e.Operand.String())
	imgui.PopStyleColor()

	// cycles column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
	imgui.Text(e.Cycles())
	imgui.PopStyleColor()

	// execution notes column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
	imgui.Text(e.ExecutionNotes)
	imgui.PopStyleColor()
}
