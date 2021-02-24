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
	"time"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

	"github.com/inkyblackness/imgui-go/v4"
)

const winDisasmID = "Disassembly"

type winDisasm struct {
	img  *SdlImgui
	open bool

	// height of options line at bottom of window. valid after first frame
	optionsHeight float32

	// options
	followCPU    bool
	showByteCode bool

	// selected bank to display
	selectedBank int

	// selectedBank can be changed by the lazy system in a different goroutine
	// so we send such changes over a channel and collect at the beginning of
	// the draw() function.
	selectedBankRefresh chan int

	// the address of the top and bottom-most visible entry. used to limit the
	// range of addresses we enquire about breakpoints for.
	addressTopList uint16
	addressBotList uint16

	// whether to focus on the PC address
	focusOnAddr bool

	// if the PC address is already visible then flash the indicator. this
	// gives the button something to do rather than giving no feedback at all.
	focusOnAddrFlash int

	// like focusOnAddrFlash but for the status string
	statusFlash int

	// because of the inherent delay of the lazy value system we need to keep
	// the focusOnAddr value at true for a short while after the gui state
	// flips to StatePaused after the gui state flips to StatePaused.
	updateOnPause int

	// whether the entry that the CPU/PC is currently "on" is visible in the
	// scroller. we use this to decide whether to show the "Goto Current"
	// button or not.
	focusAddrIsVisible bool

	// the program counter value in the previous (imgui) frame. we use this to
	// decide whether to set the focusOnAddr flag.
	focusAddrPrevFrame uint16
}

func newWinDisasm(img *SdlImgui) (window, error) {
	win := &winDisasm{
		img:                 img,
		followCPU:           true,
		selectedBankRefresh: make(chan int),
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
	// check for refreshed selected bank
	select {
	case win.selectedBank = <-win.selectedBankRefresh:
	default:
	}

	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{905, 242}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{353, 466}, imgui.ConditionFirstUseEver)
	imgui.BeginV(win.id(), &win.open, 0)
	defer imgui.End()

	// the bank that is currently selected
	bank := win.img.lz.Cart.CurrBank

	// the value of focusAddr depends on the state of the CPU. if the Final
	// state of the CPU's last execution result is true then we can be sure the
	// PC value is valid and points to a real instruction. we need this because
	// we can never be sure when we are going to draw this window
	var focusAddr uint16

	if bank.ExecutingCoprocessor {
		// if coprocessor is running then jam the focusAddr value at address the
		// CPU will resume from once the coprocessor has finished.
		focusAddr = bank.CoprocessorResumeAddr
	} else {
		if win.img.lz.Debugger.LastResult.Result.Final {
			focusAddr = win.img.lz.CPU.PC.Address()
		} else {
			focusAddr = win.img.lz.Debugger.LastResult.Result.Address
			bank = win.img.lz.Debugger.LastResult.Bank
		}
	}

	// only keep cartridge bits
	focusAddr &= memorymap.CartridgeBits

	// bank selector / information
	comboPreview := fmt.Sprintf("Viewing bank %d", win.selectedBank)
	if imgui.BeginComboV("##bankselect", comboPreview, imgui.ComboFlagsNoArrowButton) {
		for n := 0; n < win.img.lz.Cart.NumBanks; n++ {
			if imgui.Selectable(fmt.Sprintf("View bank %d", n)) {
				win.selectedBank = n
			}
		}
		imgui.EndCombo()
	}

	// show goto current button. do not not show if focusAddr is visible or if
	// debugger is currently running. the latter case isn't important except
	// when the followCPU option is false. if it is then the "Goto Current"
	// button might flash on/off, which is annoying.
	imgui.SameLine()
	if imgui.Button("Goto Current") {
		if !bank.NonCart {
			win.focusOnAddr = true
			win.selectedBank = bank.Number
			if win.focusAddrIsVisible {
				win.focusOnAddrFlash = 6
			}
		} else {
			win.statusFlash = 6
		}
	}

	// show status information. indicating that the CPU is executing
	// instructions that are not being disassembled for one reason or another.
	//
	// "Goto Current" button will cause status string to flash in the same
	// style as the focusAddr will flash.
	var status string
	if bank.ExecutingCoprocessor {
		status = "executing coprocessor instructions"
	} else if bank.NonCart {
		status = "executing non-cartridge addresses"
	} else if bank.IsSegmented {
		if bank.Name != "" {
			status = fmt.Sprintf("executing %s in segment %d", bank.Name, bank.Segment)
		} else {
			status = fmt.Sprintf("executing %d in segment %d", bank.Number, bank.Segment)
		}
	}
	if win.statusFlash > 0 {
		win.statusFlash--
	}
	if win.statusFlash == 0 || win.statusFlash%2 == 0 {
		imguiIndentText(status)
	} else {
		imguiIndentText("")
	}

	// turn off currentPCisVisible by default, we'll turn it on if required
	win.focusAddrIsVisible = false

	// draw all entries for bank. being careful with the onBank argument,
	// making sure to test bank.NonCart
	win.drawBank(win.selectedBank, focusAddr, win.selectedBank == bank.Number && !bank.NonCart)

	// draw options and status line. start height measurement
	win.optionsHeight = imguiMeasure(func() {
		imgui.Spacing()
		imgui.Checkbox("Show Bytecode", &win.showByteCode)
		imgui.SameLine()
		imgui.Checkbox("Follow CPU", &win.followCPU)
	}).Y

	// handle different gui states.
	switch win.img.state {
	case gui.StateInitialising:
		win.updateOnPause = 5
	case gui.StateRunning:
		win.focusOnAddr = win.followCPU
		if win.focusOnAddr {
			win.selectedBank = bank.Number
			win.updateOnPause = 1
		}
	case gui.StatePaused:
		win.focusOnAddr = win.updateOnPause > 0 || focusAddr != win.focusAddrPrevFrame
		if win.updateOnPause > 0 {
			wait := time.Now()
			go func() {
				t := <-win.img.lz.RefreshPulse
				for wait.After(t) {
					t = <-win.img.lz.RefreshPulse
				}
				win.selectedBankRefresh <- win.img.lz.Cart.CurrBank.Number
			}()
			win.updateOnPause--
		}
	}

	// record the focusAddr in time for the next frame
	win.focusAddrPrevFrame = focusAddr
}

// drawBank specified by bank argument.
func (win *winDisasm) drawBank(bank int, focusAddr uint16, onBank bool) {
	var err error
	var eitr *disassembly.IterateEntries

	if onBank {
		eitr, err = win.img.lz.Dbg.Disasm.NewEntriesIteration(disassembly.EntryLevelBlessed, bank, focusAddr)
	} else {
		eitr, err = win.img.lz.Dbg.Disasm.NewEntriesIteration(disassembly.EntryLevelBlessed, bank)
	}

	// check that NewBankIteration has succeeded. if it hasn't it probably
	// means the cart has changed in the middle of the draw routine. but that's
	// okay, we only have to wait one frame before we draw again
	if err != nil {
		return
	}

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", bank), imgui.Vec2{X: 0, Y: height}, false, 0)
	defer imgui.EndChild()

	numColumns := 7
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsRowBg
	if !imgui.BeginTableV("bank", numColumns, flgs, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	// set neutral colors for table rows by default. we'll change it to
	// something more meaningful as appropriate (eg. entry at PC address)
	imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
	imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)
	defer imgui.PopStyleColorV(2)

	var clipper imgui.ListClipper
	clipper.Begin(eitr.EntryCount + eitr.LabelCount)
	for clipper.Step() {
		_, _ = eitr.Start()
		_, e := eitr.SkipNext(clipper.DisplayStart, true)
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

			win.drawEntry(e, focusAddr, onBank)

			// if the current CPU entry is visible then raise the currentPCisVisble flag
			if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
				win.focusAddrIsVisible = win.focusAddrIsVisible || imgui.IsItemVisible()
			}

			// advance clipper
			_, e = eitr.Next()
			if e == nil {
				break // clipper.DisplayStart loop
			}

			// note address of bottom-most visible entry
			win.addressBotList = e.Result.Address
		}
	}

	// scroll to correct entry
	if onBank && win.focusOnAddr {
		// calculate the pixel value of the current entry. the adjustment of 4
		// is to ensure that some preceding entries are displayed before the
		// current entry
		y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
		y = float32(eitr.FocusAddrCt-4) * y

		// scroll to pixel value
		imgui.SetScrollY(y)
	}

	// set lazy update list
	win.img.lz.Breakpoints.SetUpdateList(bank, win.addressTopList, win.addressBotList)
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, focusAddr uint16, onBank bool) {
	imgui.TableNextRow()

	// draw attention to current disasm of current PC address. flash if necessary
	if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		if win.focusOnAddrFlash > 0 {
			win.focusOnAddrFlash--
		}
		if win.focusOnAddrFlash == 0 || win.focusOnAddrFlash%2 == 0 {
			hi := win.img.cols.DisasmCPUstep
			if !win.img.lz.Debugger.LastResult.Result.Final {
				hi = win.img.cols.DisasmVideoStep
			}
			imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, hi)
		}
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
