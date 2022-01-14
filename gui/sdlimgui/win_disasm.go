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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

	"github.com/inkyblackness/imgui-go/v4"
)

const winDisasmID = "Disassembly"

type winDisasm struct {
	img  *SdlImgui
	open bool

	// more recently seen emulation state
	lastSeenState emulation.State
	lastSeenPC    uint16

	// height of options line at bottom of window. valid after first frame
	optionsHeight float32

	// options
	showDetails bool
	followCPU   bool

	// selected bank to display
	selectedBank int

	// flag stating whether bank combo is open
	selectedBankComboOpen bool

	// whether to focus on the PC address
	focusOnAddr bool

	// widths of columns in the disasm table
	//
	// widthOperands is implied and is the width of the window minus widthSum
	widthBreak    float32
	widthAddr     float32
	widthOperator float32
	widthCycles   float32
	widthNotes    float32
	widthSum      float32

	// widths of goto column in control bar. bank selector column takes the
	// remainder of the space
	widthGoto float32
}

func newWinDisasm(img *SdlImgui) (window, error) {
	win := &winDisasm{
		img:         img,
		showDetails: true,
		followCPU:   true,
	}
	return win, nil
}

func (win *winDisasm) init() {
	win.widthBreak = imgui.CalcTextSize("! ", true, 0).X
	win.widthAddr = imgui.CalcTextSize("$FFFF", true, 0).X
	win.widthOperator = imgui.CalcTextSize("AND ", true, 0).X
	win.widthCycles = imgui.CalcTextSize("2/3 ", true, 0).X
	win.widthNotes = imgui.CalcTextSize(string(fonts.ExecutionNotes), true, 0).X
	win.widthSum = win.widthBreak + win.widthAddr + win.widthOperator + win.widthCycles + win.widthNotes

	win.widthGoto = imgui.CalcTextSize(string(fonts.DisasmGotoCurrent), true, 0).X
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

	imgui.SetNextWindowPosV(imgui.Vec2{1021, 34}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{500, 552}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{500, 1000})

	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	if imgui.IsWindowCollapsed() {
		return
	}

	// the bank that is currently selected
	bank := win.img.lz.Cart.CurrBank

	// focus on address if the state has changed to the paused state; or the PC
	// has changed and the followCPU option is set
	//
	// using the lazy emulation.State value rather than the live state - the
	// live state can cause synchronisation problems meaning focus is lost
	if (win.img.lz.Debugger.State == emulation.Paused && win.lastSeenState != emulation.Paused) ||
		(win.followCPU && win.img.lz.CPU.PC.Address() != win.lastSeenPC) {

		win.focusOnAddr = true
		win.selectedBank = bank.Number
	}
	win.lastSeenPC = win.img.lz.CPU.PC.Address()
	win.lastSeenState = win.img.lz.Debugger.State

	// the value of focusAddr depends on the state of the CPU. if the Final
	// state of the CPU's last execution result is true then we can be sure the
	// PC value is valid and points to a real instruction. we need this because
	// we can never be sure when we are going to draw this window
	var focusAddr uint16

	if bank.ExecutingCoprocessor {
		// if coprocessor is running then jam the focusAddr value at address the
		// CPU will resume from once the coprocessor has finished.
		focusAddr = bank.CoprocessorResumeAddr & memorymap.CartridgeBits
	} else {
		// focus address (and bank) depends on if we're in the middle of an
		// CPU instruction or not. special condition for freshly reset CPUs
		if win.img.lz.Debugger.LastResult.Result.Final || win.img.lz.CPU.HasReset {
			focusAddr = win.img.lz.CPU.PC.Address() & memorymap.CartridgeBits
		} else {
			focusAddr = win.img.lz.Debugger.LastResult.Result.Address & memorymap.CartridgeBits
			bank = win.img.lz.Debugger.LastResult.Bank
		}
	}

	win.drawControlBar(bank)

	// draw all entries for bank. being careful with the onBank argument,
	// making sure to test bank.NonCart
	win.drawBank(win.selectedBank, focusAddr, win.selectedBank == bank.Number && !bank.NonCart)

	win.drawOptions(bank)
}

func (win *winDisasm) drawControlBar(bank mapper.BankInfo) {
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	numColumns := 2
	imgui.BeginTableV("##controlBar", numColumns, flgs, imgui.Vec2{}, 0)

	bankWidth := imgui.ContentRegionAvail().X - imgui.CurrentStyle().ItemSpacing().X*float32(numColumns)
	bankWidth -= win.widthGoto
	imgui.TableSetupColumnV("goto", imgui.TableColumnFlagsNone, win.widthGoto, 0)
	imgui.TableSetupColumnV("bank", imgui.TableColumnFlagsNone, bankWidth, 1)

	imgui.TableNextRow()

	// goto current PC
	imgui.TableNextColumn()
	imgui.AlignTextToFramePadding()
	imgui.Text(string(fonts.DisasmGotoCurrent))
	if imgui.IsItemHovered() {
		if imgui.IsItemClicked() {
			win.focusOnAddr = true
			win.selectedBank = bank.Number
		} else {
			imguiTooltip(func() {
				imgui.Text("Focus on PC address")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
				imgui.Text(fmt.Sprintf("$%04x", win.img.lz.CPU.PC.Address()))
				imgui.PopStyleColor()

				if win.img.lz.Cart.NumBanks > 1 {
					imgui.SameLine()
					imgui.Text("bank")
					imgui.SameLine()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBank)
					imgui.Text(bank.String())
					imgui.PopStyleColor()
				}
			}, true)
		}
	}

	// bank selector / information
	imgui.TableNextColumn()
	comboPreview := fmt.Sprintf("Viewing bank %d", win.selectedBank)
	imgui.PushItemWidth(imgui.ContentRegionAvail().X)
	if imgui.BeginComboV("##bankselect", comboPreview, imgui.ComboFlagsNone) {
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
	imgui.PopItemWidth()

	imgui.EndTable()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()
}

func (win *winDisasm) drawOptions(bank mapper.BankInfo) {
	// draw options and status line. start height measurement
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Checkbox("Show Details in Tooltip", &win.showDetails)
		imgui.SameLineV(0, 15)
		imgui.Checkbox("Follow CPU", &win.followCPU)

		// special execution icon
		if bank.ExecutingCoprocessor {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.CoProcExecution))
			imguiTooltipSimple("CoProcessor is executing")
		} else if bank.NonCart {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.NonCartExecution))
			imguiTooltipSimple("Executing a non-cartridge address!")
		}
	})
}

func (win *winDisasm) startTable() {
	numColumns := 6
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	if !imgui.BeginTableV("bank", numColumns, flgs, imgui.Vec2{}, 0) {
		return
	}

	operandWidth := imgui.ContentRegionAvail().X - imgui.CurrentStyle().ItemSpacing().X*float32(numColumns)
	operandWidth -= win.widthSum

	imgui.TableSetupColumnV("break", imgui.TableColumnFlagsNone, win.widthBreak, 0)
	imgui.TableSetupColumnV("address", imgui.TableColumnFlagsNone, win.widthAddr, 1)
	imgui.TableSetupColumnV("operator", imgui.TableColumnFlagsNone, win.widthOperator, 2)
	imgui.TableSetupColumnV("operand", imgui.TableColumnFlagsNone, operandWidth, 3)
	imgui.TableSetupColumnV("cycles", imgui.TableColumnFlagsNone, win.widthCycles, 4)
	imgui.TableSetupColumnV("notes", imgui.TableColumnFlagsNone, win.widthNotes, 5)
}

// drawBank specified by bank argument.
func (win *winDisasm) drawBank(selectedBank int, focusAddr uint16, onBank bool) {
	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", selectedBank), imgui.Vec2{X: 0, Y: height}, false, imgui.WindowFlagsAlwaysVerticalScrollbar)

	win.img.dbg.Disasm.BorrowDisasm(func(dsmEntries *disassembly.DisasmEntries) {
		if dsmEntries == nil {
			imgui.Text("No disassembly available")
			return
		}

		win.startTable()

		// set neutral colors for table rows by default. we'll change it to
		// something more meaningful as appropriate (eg. entry at PC address)
		imgui.PushStyleColor(imgui.StyleColorTableRowBg, win.img.cols.WindowBg)
		imgui.PushStyleColor(imgui.StyleColorTableRowBgAlt, win.img.cols.WindowBg)

		// number of blessed entries in disasm. being careful to include labels in the count
		ct := 0
		focusCt := 0
		focusCtApply := false
		for _, e := range dsmEntries.Entries[selectedBank] {
			if e.Level >= disassembly.EntryLevelBlessed {
				ct++
				if e.Label.String() != "" {
					ct++
				}

				if onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					focusCt = ct
					focusCtApply = true
				}
			}
		}

		func() {
			var clipper imgui.ListClipper
			clipper.Begin(ct)
			for clipper.Step() {
				// skip entries that aren't to be displayed
				n := 0
				e := dsmEntries.Entries[selectedBank][n]
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
					if n >= len(dsmEntries.Entries[selectedBank]) {
						return
					}
					e = dsmEntries.Entries[selectedBank][n]
					for e.Level < disassembly.EntryLevelBlessed {
						n++
						if n >= len(dsmEntries.Entries[selectedBank]) {
							return
						}
						e = dsmEntries.Entries[selectedBank][n]
					}
				}

				// draw labels and entries that are to be displayed
				for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
					win.drawLabel(e, selectedBank)
					win.drawEntry(e, focusAddr, onBank, selectedBank)

					// skip non-blessed entries
					n++
					if n >= len(dsmEntries.Entries[selectedBank]) {
						return
					}
					e = dsmEntries.Entries[selectedBank][n]
					for e.Level < disassembly.EntryLevelBlessed {
						n++
						if n >= len(dsmEntries.Entries[selectedBank]) {
							return
						}
						e = dsmEntries.Entries[selectedBank][n]
					}
				}
			}
		}()

		// scroll to correct entry. this is in addition to the automated
		// scrolling we do in drawEntry(). both are required as a consequence
		// of how ListClipper works
		if onBank && win.focusOnAddr && focusCtApply {
			y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y

			// leave a small gap between the top of the scroll window and
			// the focused entry
			const focusGap = 7

			y = float32(focusCt-focusGap) * y
			imgui.SetScrollY(y)
		}

		imgui.PopStyleColorV(2)
		imgui.EndTable()
	})

	imgui.EndChild()
}

func (win *winDisasm) drawLabel(e *disassembly.Entry, bank int) {
	// no label to draw
	label := e.Label.String()
	if len(label) == 0 {
		return
	}

	// end existing disasm table before drawing label. (re)start table before
	// the end of the function
	imgui.EndTable()
	defer win.startTable()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	defer imgui.PopStyleColorV(2)
	imgui.SelectableV("", false, imgui.SelectableFlagsNone, imgui.Vec2{0, 0})
	imgui.SameLine()

	imgui.Text(e.Label.String())
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, focusAddr uint16, onBank bool, bank int) {
	imgui.TableNextRow()

	// highligh current PC entry
	if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.DisasmStep)

		// scroll to this entry if required. this is in addition to the
		// calculated scrolling we do in the drawBank() loop. both are required
		// as a consequence of how ListClipper works
		if win.focusOnAddr {
			imgui.SetScrollHereY(0.20)
			win.focusOnAddr = false
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

	// single click on the address entry toggles a PC breakpoint
	if imgui.IsItemClicked() {
		win.toggleBreak(e)
	}

	// tooltip on hover and context menu on right mouse button
	if imgui.IsItemHovered() {
		imguiTooltip(func() {
			imgui.Text("Address:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(e.Address)
			imgui.PopStyleColor()

			imgui.Text("Bytecode:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmByteCode)
			imgui.Text(e.Bytecode)
			imgui.PopStyleColor()

			imgui.Text("Instruction:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
			imgui.Text(e.Operator)
			imgui.PopStyleColor()
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(e.Operand.String())
			imgui.PopStyleColor()

			// treat an instruction that is "cycling" differently
			if !win.img.lz.Debugger.LastResult.Result.Final {
				if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
					imgui.Text(fmt.Sprintf("%c cycling instruction (%s)", fonts.CyclingInstruction, win.img.lz.Debugger.LastResult.Cycles()))
					imgui.PopStyleColor()
				}
			} else {
				if e.Level == disassembly.EntryLevelExecuted {
					imgui.Text("Cycles:")
					imgui.SameLine()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
					imgui.Text(e.Cycles())
					imgui.PopStyleColor()

					if e.LastExecutionNotes != "" {
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						imgui.Text(fmt.Sprintf("%c %s", fonts.ExecutionNotes, e.LastExecutionNotes))
					}
				} else {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					imgui.Text("Never executed")
				}
			}
		}, true)
	}

	// breakpoint indicator column. using the same column as the selectable
	// above (which spans all columns)
	imgui.SameLine()
	if hasPCbreak {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
		imgui.Text(fmt.Sprintf("%c", fonts.Breakpoint))
		imgui.PopStyleColor()
	}

	// address column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
	imgui.Text(e.Address)
	imgui.PopStyleColor()

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
	if !win.img.lz.Debugger.LastResult.Result.Final && onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		imgui.Text(string(fonts.CyclingInstruction))
	} else {
		imgui.Text(e.DefnCycles)
	}
	imgui.PopStyleColor()

	// execution notes column
	imgui.TableNextColumn()
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
	if e.LastExecutionNotes != "" {
		imgui.Text(string(fonts.ExecutionNotes))
	}
	imgui.PopStyleColor()
}

func (win *winDisasm) toggleBreak(e *disassembly.Entry) {
	f := e // copy of pushed disasm entry
	win.img.dbg.PushRawEvent(func() { win.img.dbg.TogglePCBreak(f) })
}
