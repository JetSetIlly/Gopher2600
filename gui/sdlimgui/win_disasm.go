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

	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"
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

	// flag stating whether bank combo is open
	selectedBankComboOpen bool

	// whether to focus on the PC address
	focusOnAddr bool

	// if the PC address is already visible then flash the indicator. this
	// gives the button something to do rather than giving no feedback at all.
	focusAddrFlash int

	// like focusAddrFlash but for the status string
	statusFlash int

	// whether the entry that the CPU/PC is currently "on" is visible in the
	// scroller. we use this to decide whether to show the "Goto Current"
	// button or not.
	focusAddrIsVisible bool

	// label editing. labelEditTag identifies the tag being edited and
	// labelEdit is the edited version of the label which will be used to
	// update the label symbols table.
	labelEditTag string
	labelEdit    string

	// the length of time to wait before accepting the new emulation state.
	// this is to iron out the inherent delay in the lazy value system.
	syncDelay int

	contextMenu string
}

func newWinDisasm(img *SdlImgui) (window, error) {
	win := &winDisasm{
		img:       img,
		followCPU: true,
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

// the length of time to flash for (ee focusAddrFlash and statusFlash). the
// unit of time is the number of times the GUI goroutine is serviced. therfore
// if that time changes the flashing effect should change also
const focusFlashTime = 10
const focusFlashPeriod = 2

func (win *winDisasm) draw() {
	if !win.open {
		return
	}

	win.checkEmulationState()

	imgui.SetNextWindowPosV(imgui.Vec2{1021, 34}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{500, 552}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{500, 1000})

	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNone)
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
		// focus address (and bank) depends on if we're in the middle of an
		// CPU instruction or not. special condition for freshly reset CPUs
		if win.img.lz.Debugger.LastResult.Result.Final || win.img.lz.CPU.HasReset {
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

			// set scroll on first frame combo is open
			if !win.selectedBankComboOpen && n == win.selectedBank {
				imgui.SetScrollHereY(0.0)
			}
		}
		imgui.EndCombo()

		// note that combo is open *after* it has been drawn
		win.selectedBankComboOpen = true
	} else {
		win.selectedBankComboOpen = false
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
				win.focusAddrFlash = focusFlashTime
			}
		} else {
			win.statusFlash = focusFlashTime
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
	if win.statusFlash == 0 || win.statusFlash%focusFlashPeriod == 0 {
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
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Checkbox("Show Bytecode", &win.showByteCode)
		imgui.SameLine()
		imgui.Checkbox("Follow CPU", &win.followCPU)
	})
}

// switching to a new emulationState needs to be handled with care,
// particularly because of the skew with the update of the lazy value system.
// the checkEmulationState() function handles the change of state as best we
// can. it works well.
//
// note that the issue only really arises when we move *to* the Paused state.
// in other states the gui is either moving too brief or too fast to see
// anything meaningful.
func (win *winDisasm) checkEmulationState() {
	// the number of gui updates (calls to draw()) to wait before accepting the
	// number emulation state. these values is arbitrary
	const (
		syncDelayInitialising = 100
		syncDelayDefault      = 30
	)

	switch win.img.emulation.State() {
	case emulation.EmulatorStart:
		fallthrough
	case emulation.Initialising:
		win.focusOnAddr = true
		win.syncDelay = syncDelayInitialising
	default:
		win.focusOnAddr = win.followCPU
		if win.focusOnAddr {
			win.selectedBank = win.img.lz.Cart.CurrBank.Number
			win.syncDelay = syncDelayDefault
		}
	case emulation.Paused:
		win.focusOnAddr = win.syncDelay > 0 && win.followCPU
		if win.syncDelay > 0 {
			win.selectedBank = win.img.lz.Cart.CurrBank.Number
			win.syncDelay--
		}
	}
}

// drawBank specified by bank argument.
func (win *winDisasm) drawBank(bank int, focusAddr uint16, onBank bool) {
	win.img.dbg.Disasm.BorrowDisasm(func(dsmEntries *disassembly.DisasmEntries) {
		height := imguiRemainingWinHeight() - win.optionsHeight
		imgui.BeginChildV(fmt.Sprintf("bank %d", bank), imgui.Vec2{X: 0, Y: height}, false, 0)
		defer imgui.EndChild()

		numColumns := 7
		flgs := imgui.TableFlagsNone
		flgs |= imgui.TableFlagsSizingFixedFit
		if !imgui.BeginTableV("bank", numColumns, flgs, imgui.Vec2{}, 0) {
			return
		}

		defer imgui.EndTable()

		// set neutral colors for table rows by default. we'll change it to
		// something more meaningful as appropriate (eg. entry at PC address)
		imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
		imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)
		defer imgui.PopStyleColorV(2)

		// number of blessed entries in disasm. being careful to include labels
		// in the count
		ct := 0
		focusAddrCt := -1
		for _, e := range dsmEntries.Entries[bank] {
			if e.Level >= disassembly.EntryLevelBlessed {
				ct++
				if e.Label.String() != "" {
					ct++
				}

				if onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					focusAddrCt = ct - 4
				}
			}
		}

		var clipper imgui.ListClipper
		clipper.Begin(ct)
		for clipper.Step() {
			// skip entries that aren't to be displayed
			n := 0
			e := dsmEntries.Entries[bank][n]
			skip := clipper.DisplayStart
			for skip > 0 {
				if e == nil {
					return
				}
				skip--
				if e.Label.String() != "" {
					skip--
				}

				// skip non-blessed entries
				n++
				if n >= len(dsmEntries.Entries[bank]) {
					return
				}
				e = dsmEntries.Entries[bank][n]
				for e.Level < disassembly.EntryLevelBlessed {
					n++
					if n >= len(dsmEntries.Entries[bank]) {
						return
					}
					e = dsmEntries.Entries[bank][n]
				}
			}

			// draw labels and entries that are to be displayed
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				win.drawLabel(e, bank)
				win.drawEntry(e, focusAddr, onBank, bank)

				if onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					win.focusAddrIsVisible = win.focusAddrIsVisible || imgui.IsItemVisible()
				}

				// skip non-blessed entries
				n++
				if n >= len(dsmEntries.Entries[bank]) {
					return
				}
				e = dsmEntries.Entries[bank][n]
				for e.Level < disassembly.EntryLevelBlessed {
					n++
					if n >= len(dsmEntries.Entries[bank]) {
						return
					}
					e = dsmEntries.Entries[bank][n]
				}
			}

			// scroll to correct entry
			//
			// note that this is inside the clipper.Step() loop. I have no idea
			// why is necessary but it we SetScrollY() outside the loop then
			// success is inconsistent
			if onBank && win.focusOnAddr && focusAddrCt != -1 {
				y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y
				y = float32(focusAddrCt) * y

				// scroll to pixel value
				imgui.SetScrollY(y)
			}
		}
	})
}

func (win *winDisasm) drawLabel(e *disassembly.Entry, bank int) {
	// try to draw address label
	s := e.Label.String()
	if len(s) > 0 {
		imgui.TableNextRow()

		// put in address column (second column)
		imgui.TableNextColumn()
		imgui.TableNextColumn()

		// address label (and label editing)
		labelEditTag := fmt.Sprintf("%s%s", e.Bank.Name, e.Address)
		if win.labelEditTag == labelEditTag {
			imgui.PushItemWidth(imguiTextWidth(len(win.labelEdit)))

			flgs := imgui.InputTextFlagsEnterReturnsTrue | imgui.InputTextFlagsCharsNoBlank
			if imgui.InputTextV("##labeledit", &win.labelEdit, flgs, nil) {
				win.img.dbg.Disasm.Sym.UpdateLabel(symbols.SourceCustom, bank, e.Result.Address, s, win.labelEdit)
				win.labelEditTag = ""
			}

			if imgui.IsAnyMouseDown() && !imgui.IsItemHovered() {
				win.labelEditTag = ""
			} else {
				imgui.SetKeyboardFocusHere()
			}

			imgui.PopItemWidth()
		} else {
			imgui.Text(e.Label.String())
			if imgui.IsItemClicked() && imgui.IsMouseDoubleClicked(0) {
				win.labelEditTag = labelEditTag
				win.labelEdit = s
			}
		}
	}
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, focusAddr uint16, onBank bool, bank int) {
	imgui.TableNextRow()

	// prepare execution notes and handle focus-flash
	executionNotes := e.LastExecutionNotes
	if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		// execution notes
		if !win.img.lz.Debugger.LastResult.Result.Final {
			executionNotes = fmt.Sprintf("executing (%s cycles)", win.img.lz.Debugger.LastResult.Cycles())
		}

		// draw attention to current entry. flash if necessary
		if win.focusAddrFlash > 0 {
			win.focusAddrFlash--
		}
		if win.focusAddrFlash == 0 || win.focusAddrFlash%focusFlashPeriod == 0 {
			imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.DisasmStep)
		}
	}

	// does this entry/address have a PC break applied to it
	var hasPCbreak bool
	if win.img.lz.Debugger.Breakpoints != nil {
		hasPCbreak, _ = win.img.lz.Debugger.Breakpoints.HasPCBreak(e.Result.Address, bank)
	}

	// first column is a selectable that spans all lines and the breakpoint indicator
	//
	// the selectable isn't visible but it's something against which we can
	// IsItemClick() and IsItemHovered(). this is so we can toggle a PC break
	// and open a context menu
	imgui.TableNextColumn()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	defer imgui.PopStyleColorV(2)

	imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	imgui.SameLine()

	// single click on the address entry toggles a PC breakpoint
	if imgui.IsItemClicked() {
		win.toggleBreak(e)
	}

	// context menu on right mouse button
	if imgui.IsItemHovered() && imgui.IsMouseDown(1) {
		imgui.OpenPopup(disasmBreakMenuID)
		win.contextMenu = e.Address
	}
	if e.Address == win.contextMenu {
		win.drawDisasmBreakMenu(e, hasPCbreak)
	}

	// breakpoint indicator column. using the same column as the selectable
	// above (which spans all columns)
	if hasPCbreak {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
		imgui.Text(fmt.Sprintf("%c", fonts.Breakpoint))
		imgui.PopStyleColor()
	} else {
		imgui.Text(" ")
	}

	// address column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
	imgui.Text(e.Address)
	imgui.PopStyleColor()

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
	imgui.Text(e.DefnCycles)
	imgui.PopStyleColor()

	// execution notes column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
	imgui.Text(executionNotes)
	imgui.PopStyleColor()
}

const disasmBreakMenuID = "disasmBreakMenu"

func (win *winDisasm) drawDisasmBreakMenu(e *disassembly.Entry, hasPCbreak bool) {
	if imgui.BeginPopup(disasmBreakMenuID) {
		imgui.Text("Break on PC Address")
		imguiSeparator()

		var s string
		if hasPCbreak {
			s = fmt.Sprintf("Clear %s", e.Address)
		} else {
			s = fmt.Sprintf("Set %s", e.Address)
		}

		if imgui.Selectable(s) {
			win.toggleBreak(e)
		}
		imgui.EndPopup()
	}
}

func (win *winDisasm) toggleBreak(e *disassembly.Entry) {
	f := e // copy of pushed disasm entry
	win.img.dbg.PushRawEvent(func() { win.img.dbg.TogglePCBreak(f) })
}
