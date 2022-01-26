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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

	"github.com/inkyblackness/imgui-go/v4"
)

const winDisasmID = "Disassembly"

type disasmFilter int

const (
	filterBank disasmFilter = iota
	filterCPUBug
	filterPageFault
)

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
	filter       disasmFilter
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
	win.widthNotes = imgui.CalcTextSize(string(fonts.CPUBug), true, 0).X
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

	// the currBank that is currently selected
	currBank := win.img.lz.Cart.CurrBank

	// focus on address if the state has changed to the paused state and
	// followCPU is set; or the PC has changed (this is because the state
	// change might be missed)
	//
	// using the lazy emulation.State value rather than the live state - the
	// live state can cause synchronisation problems meaning focus is lost
	if win.followCPU {
		if (win.img.lz.Debugger.State == emulation.Paused && win.lastSeenState != emulation.Paused) ||
			win.img.lz.CPU.PC.Address() != win.lastSeenPC {

			win.focusOnAddr = true
			win.selectedBank = currBank.Number
		}
	}
	win.lastSeenPC = win.img.lz.CPU.PC.Address()
	win.lastSeenState = win.img.lz.Debugger.State

	// the value of focusAddr depends on the state of the CPU. if the Final
	// state of the CPU's last execution result is true then we can be sure the
	// PC value is valid and points to a real instruction. we need this because
	// we can never be sure when we are going to draw this window
	var focusAddr uint16

	if currBank.ExecutingCoprocessor {
		// if coprocessor is running then jam the focusAddr value at address the
		// CPU will resume from once the coprocessor has finished.
		focusAddr = currBank.CoprocessorResumeAddr & memorymap.CartridgeBits
	} else {
		// focus address depends on if we're in the middle of an CPU
		// instruction or not. special condition for freshly reset CPUs
		if win.img.lz.Debugger.LiveDisasmEntry.Result.Final || win.img.lz.CPU.HasReset {
			focusAddr = win.img.lz.CPU.PC.Address() & memorymap.CartridgeBits
		} else {
			focusAddr = win.img.lz.Debugger.LiveDisasmEntry.Result.Address & memorymap.CartridgeBits
		}
	}

	win.drawControlBar()
	win.drawBank(focusAddr)
	win.drawOptions()
}

func (win *winDisasm) drawControlBar() {
	currBank := win.img.lz.Cart.CurrBank

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
			win.selectedBank = currBank.Number
			win.filter = filterBank
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
					imgui.Text(currBank.String())
					imgui.PopStyleColor()
				}
			}, true)
		}
	}

	// bank selector / information
	imgui.TableNextColumn()
	comboPreview := ""
	switch win.filter {
	case filterBank:
		comboPreview = fmt.Sprintf("Viewing bank %d", win.selectedBank)
	case filterCPUBug:
		comboPreview = fmt.Sprintf("Viewing %c CPU Bugs", fonts.CPUBug)
	case filterPageFault:
		comboPreview = fmt.Sprintf("Viewing %c Page Faults", fonts.PageFault)
	}

	imgui.PushItemWidth(imgui.ContentRegionAvail().X)
	if imgui.BeginComboV("##filter", comboPreview, imgui.ComboFlagsHeightLargest) {
		for n := 0; n < win.img.lz.Cart.NumBanks; n++ {
			if imgui.Selectable(fmt.Sprintf("View bank %d", n)) {
				win.filter = filterBank
				win.selectedBank = n
			}

			// set scroll on the first frame that the combo is open
			if !win.selectedBankComboOpen && n == win.selectedBank {
				imgui.SetScrollHereY(0.0)
			}
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		if imgui.Selectable(fmt.Sprintf("View %c CPU Bugs", fonts.CPUBug)) {
			win.filter = filterCPUBug
		}

		imgui.Spacing()

		if imgui.Selectable(fmt.Sprintf("View %c Page Faults", fonts.PageFault)) {
			win.filter = filterPageFault
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

func (win *winDisasm) drawOptions() {
	currBank := win.img.lz.Cart.CurrBank

	// draw options and status line. start height measurement
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Checkbox("Show Details in Tooltip", &win.showDetails)
		imgui.SameLineV(0, 15)
		if imgui.Checkbox("Follow CPU", &win.followCPU) {
			// goto current PC on option being set to true
			if win.followCPU {
				win.focusOnAddr = true
				win.selectedBank = currBank.Number
			}
		}

		// special execution icon
		if currBank.ExecutingCoprocessor {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.CoProcExecution))
			win.drawCoProcTooltip()
		} else if currBank.NonCart {
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
func (win *winDisasm) drawBank(focusAddr uint16) {
	currBank := win.img.lz.Cart.CurrBank
	onBank := win.selectedBank == currBank.Number && !currBank.NonCart

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("bank %d", win.selectedBank), imgui.Vec2{X: 0, Y: height}, false, imgui.WindowFlagsAlwaysVerticalScrollbar)

	win.img.dbg.Disasm.BorrowDisasm(func(dsmEntries *disassembly.DisasmEntries) {
		// borrow disasm callback can be called with a nil pointer
		if dsmEntries == nil {
			return
		}

		// very important that we check to see if selectedBank is not too high
		// for the number of entries disasm entries. the disassembly might be
		// in the process of changing while we're drawing the current frame
		if win.selectedBank >= len(dsmEntries.Entries) {
			return
		}

		// the method of iteration is different depending on the selected filter
		iterateBank := 0
		iterateIdx := 0
		headerRequired := false

		// reset should be called at outset of a new iteration
		iterateReset := func() {
			iterateBank = 0
			iterateIdx = 0
			headerRequired = true
		}

		// iterateNext moves iterateIdx and iterateBank. returns false when
		// iteration has ended
		iterateNext := func() bool {
			iterateIdx++
			if iterateIdx >= len(dsmEntries.Entries[iterateBank]) {
				iterateIdx = 0
				iterateBank++
				headerRequired = true
				if iterateBank >= len(dsmEntries.Entries) {
					return false
				}
			}
			return true
		}

		// iterateFilter returns true if entry satisifies the filter conditions
		iterateFilter := func(_ *disassembly.Entry) bool {
			return true
		}

		// iterateDraw presents the entry according to the current rules
		iterateDraw := func(e *disassembly.Entry) {
			if headerRequired {
				imgui.EndTable()
				imgui.Text(fmt.Sprintf("Bank %d", iterateBank))
				win.startTable()
				headerRequired = false
			}
			win.drawEntry(e, focusAddr, onBank, iterateBank)
		}

		// alter iterate functions according to selected filter
		switch win.filter {
		case filterBank:
			iterateFilter = func(e *disassembly.Entry) bool {
				return e.Level >= disassembly.EntryLevelBlessed && iterateBank == win.selectedBank
			}
			iterateReset = func() {
				iterateBank = win.selectedBank
				iterateIdx = 0
			}
			iterateNext = func() bool {
				iterateIdx++
				return iterateIdx < len(dsmEntries.Entries[iterateBank])
			}
			iterateDraw = func(e *disassembly.Entry) {
				win.drawLabel(e, iterateBank)
				win.drawEntry(e, focusAddr, onBank, iterateBank)
				if currBank.ExecutingCoprocessor && onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					win.drawEntryCoProcessorExecution()
				}
			}
		case filterCPUBug:
			iterateFilter = func(e *disassembly.Entry) bool {
				return e.Level >= disassembly.EntryLevelExecuted && e.Result.CPUBug != ""
			}
		case filterPageFault:
			iterateFilter = func(e *disassembly.Entry) bool {
				return e.Level >= disassembly.EntryLevelExecuted && e.Result.PageFault
			}
		}

		// nothing to do so return immediately
		if dsmEntries == nil {
			return
		}

		win.startTable()

		// number of blessed entries in disasm. being careful to include labels in the count
		ct := 0
		focusCt := 0
		focusCtApply := false

		iterateReset()
		e := dsmEntries.Entries[iterateBank][iterateIdx]
		for {
			if iterateFilter(e) {
				ct++
				if e.Label.String() != "" {
					ct++
				}

				if onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					focusCt = ct
					focusCtApply = true
				}
			}

			if !iterateNext() {
				break
			}
			e = dsmEntries.Entries[iterateBank][iterateIdx]
		}

		// wrap list clipper in anonymous function call. convenient to just
		// return from the function from inside a nested loop
		func() {
			// start iteration again
			iterateReset()
			e = dsmEntries.Entries[iterateBank][iterateIdx]

			var clipper imgui.ListClipper
			clipper.Begin(ct)
			for clipper.Step() {
				// skip entries that aren't to be displayed
				skip := clipper.DisplayStart
				for skip > 1 {
					if e == nil {
						return
					}

					// skip entries counting label as appropriate
					skip--
					if e.Label.String() != "" {
						skip--
					}

					// skip non-blessed entries
					if !iterateNext() {
						return
					}
					e = dsmEntries.Entries[iterateBank][iterateIdx]
					for !iterateFilter(e) {
						if !iterateNext() {
							return
						}
						e = dsmEntries.Entries[iterateBank][iterateIdx]
					}
				}

				for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
					if iterateFilter(e) {
						iterateDraw(e)
					} else {
						i--
					}

					// skip non-blessed entries
					if !iterateNext() {
						return
					}
					e = dsmEntries.Entries[iterateBank][iterateIdx]
					for !iterateFilter(e) {
						if !iterateNext() {
							return
						}
						e = dsmEntries.Entries[iterateBank][iterateIdx]
					}
				}
			}
		}()

		// scroll to correct entry. we do this rather than a SetScrollHereY()
		// call in drawEntry() because we may need to focus on an address that
		// hasn't been drawn - ListClipper will only draw the entries that are
		// currently visible and by it's nature, focusOnAddr will want to work
		// with entries that may not be visible
		if onBank && win.focusOnAddr && focusCtApply {
			y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y

			// leave a small gap between the top of the scroll window and
			// the focused entry
			const focusGap = 7

			y = float32(focusCt-focusGap) * y
			imgui.SetScrollY(y)
		}

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

func (win *winDisasm) drawCoProcTooltip() {
	imguiTooltipSimple("Coprocessor is executing")
}

func (win *winDisasm) drawEntryCoProcessorExecution() {
	imgui.EndTable()
	defer win.startTable()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	defer imgui.PopStyleColorV(2)
	imgui.SelectableV("", false, imgui.SelectableFlagsNone, imgui.Vec2{0, 0})

	win.drawCoProcTooltip()

	imgui.SameLine()
	imgui.Text(fmt.Sprintf("    %c 6507 will resume here", fonts.CoProcExecution))
}

func (win *winDisasm) drawEntry(e *disassembly.Entry, focusAddr uint16, onBank bool, bank int) {
	imgui.TableNextRow()

	// highligh current PC entry
	if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.DisasmStep)
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
	if win.showDetails && imgui.IsItemHovered() {
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
			if !e.Result.Final {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(fmt.Sprintf("%c cycling instruction (%s)", fonts.CyclingInstruction, e.Cycles()))
				imgui.PopStyleColor()
			} else {
				imgui.Text("Cycles:")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(e.Cycles())
				imgui.PopStyleColor()
			}

			if e.Level == disassembly.EntryLevelExecuted {
				notes := e.Notes()
				if notes != "" {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					imgui.Text(fmt.Sprintf("%c %s", fonts.ExecutionNotes, notes))
				}
			} else {
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
				imgui.Text(fmt.Sprintf("%c never executed", fonts.ExecutionNotes))
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
	if !e.Result.Final {
		imgui.Text(string(fonts.CyclingInstruction))
	} else {
		imgui.Text(e.Result.Defn.Cycles.Formatted)
	}
	imgui.PopStyleColor()

	// notes column
	imgui.TableNextColumn()
	if e.Level == disassembly.EntryLevelExecuted {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
		if e.Result.CPUBug != "" {
			imgui.Text(string(fonts.CPUBug))
		} else if e.Result.PageFault {
			imgui.Text(string(fonts.PageFault))
		}
		imgui.PopStyleColor()
	}
}

func (win *winDisasm) toggleBreak(e *disassembly.Entry) {
	f := e // copy of pushed disasm entry
	win.img.dbg.PushRawEvent(func() { win.img.dbg.TogglePCBreak(f) })
}
