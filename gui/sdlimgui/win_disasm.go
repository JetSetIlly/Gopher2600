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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
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
	debuggerWin

	img *SdlImgui

	// more recently seen emulation state
	lastSeenState govern.State
	lastSeenPC    uint16

	// height of options line at bottom of window. valid after first frame
	optionsHeight float32

	// options
	followCPU  bool
	usingColor bool

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
		img:       img,
		followCPU: true,
	}
	return win, nil
}

func (win *winDisasm) init() {
	win.widthBreak = imgui.CalcTextSize(string(fonts.Breakpoint)+" ", true, 0).X
	win.widthAddr = imgui.CalcTextSize("$FFFF ", true, 0).X
	win.widthOperator = imgui.CalcTextSize("AND ", true, 0).X
	win.widthCycles = imgui.CalcTextSize("2/3 ", true, 0).X
	win.widthNotes = imgui.CalcTextSize(fmt.Sprintf("%c ", fonts.CPUBug), true, 0).X
	win.widthSum = win.widthBreak + win.widthAddr + win.widthOperator + win.widthCycles + win.widthNotes
	win.widthGoto = imgui.CalcTextSize(string(fonts.DisasmGotoCurrent), true, 0).X
}

func (win *winDisasm) id() string {
	return winDisasmID
}

func (win *winDisasm) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{1021, 34}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{500, 552}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winDisasm) draw() {
	if imgui.IsWindowCollapsed() {
		return
	}

	// the currBank that is currently selected
	addr := win.img.cache.VCS.CPU.PC.Address()
	currBank := win.img.cache.VCS.Mem.Cart.GetBank(addr)

	// focus on address if the state has changed to the paused state and
	// followCPU is set; or the PC has changed (this is because the state
	// change might be missed)
	//
	// using the lazy govern.State value rather than the live state - the
	// live state can cause synchronisation problems meaning focus is lost
	if win.followCPU {
		if (win.img.dbg.State() == govern.Paused && win.lastSeenState != govern.Paused) ||
			win.img.cache.VCS.CPU.PC.Address() != win.lastSeenPC {

			win.focusOnAddr = true
			win.selectedBank = currBank.Number
		}
	}
	win.lastSeenPC = win.img.cache.VCS.CPU.PC.Address()
	win.lastSeenState = win.img.dbg.State()

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
		if win.img.cache.Dbg.LiveDisasmEntry.Result.Final || win.img.cache.VCS.CPU.HasReset() {
			focusAddr = win.img.cache.VCS.CPU.PC.Address() & memorymap.CartridgeBits
		} else {
			focusAddr = win.img.cache.Dbg.LiveDisasmEntry.Result.Address & memorymap.CartridgeBits
		}
	}

	win.drawControlBar(currBank)
	win.drawBank(currBank, focusAddr)
	win.drawOptions(currBank)
}

func (win *winDisasm) drawControlBar(currBank mapper.BankInfo) {
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
			win.img.imguiTooltip(func() {
				if currBank.ExecutingCoprocessor {
					imgui.Text("Focus on 6507 resume address")
				} else {
					if currBank.NonCart {
						imgui.Text("Non-Cartridge execution. Nothing to focus on.")
					} else {
						imgui.Text("Focus on PC address")
						imgui.SameLine()
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
						imgui.Text(fmt.Sprintf("$%04x", win.img.cache.VCS.CPU.PC.Address()))
						imgui.PopStyleColor()

						if win.img.cache.VCS.Mem.Cart.NumBanks() > 1 {
							imgui.SameLine()
							imgui.Text("bank")
							imgui.SameLine()
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBank)
							imgui.Text(currBank.String())
							imgui.PopStyleColor()
						}
					}
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
		for n := 0; n < win.img.cache.VCS.Mem.Cart.NumBanks(); n++ {
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

func (win *winDisasm) drawOptions(currBank mapper.BankInfo) {
	// draw options and status line. start height measurement
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		if imgui.Checkbox("Follow CPU", &win.followCPU) {
			// goto current PC on option being set to true
			if win.followCPU {
				win.focusOnAddr = true
				win.selectedBank = currBank.Number
			}
		}
		imgui.SameLineV(0, 15)
		win.usingColor = win.img.prefs.colorDisasm.Get().(bool)
		if imgui.Checkbox("Use Colour", &win.usingColor) {
			win.img.prefs.colorDisasm.Set(win.usingColor)
		}

		// special execution icons
		if currBank.ExecutingCoprocessor {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.CoProcExecution))
			win.drawCoProcTooltip()
		}
		if currBank.NonCart {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.NonCartExecution))
			win.img.imguiTooltipSimple("Executing a non-cartridge address!")
		}
	})
}

func (win *winDisasm) startTable() bool {
	numColumns := 6
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	if !imgui.BeginTableV("bank", numColumns, flgs, imgui.Vec2{}, 0) {
		return false
	}

	operandWidth := imgui.ContentRegionAvail().X - imgui.CurrentStyle().ItemSpacing().X*float32(numColumns)
	operandWidth -= win.widthSum

	imgui.TableSetupColumnV("##break", imgui.TableColumnFlagsNone, win.widthBreak, 0)
	imgui.TableSetupColumnV("##address", imgui.TableColumnFlagsNone, win.widthAddr, 1)
	imgui.TableSetupColumnV("##operator", imgui.TableColumnFlagsNone, win.widthOperator, 2)
	imgui.TableSetupColumnV("##operand", imgui.TableColumnFlagsNone, operandWidth, 3)
	imgui.TableSetupColumnV("##cycles", imgui.TableColumnFlagsNone, win.widthCycles, 4)
	imgui.TableSetupColumnV("##notes", imgui.TableColumnFlagsNone, win.widthNotes, 5)

	return true
}

// drawBank specified by bank argument.
func (win *winDisasm) drawBank(currBank mapper.BankInfo, focusAddr uint16) {
	// part of the onBank condition was to test whether cart currBank.NonCart
	// was false but I now don't believe this is required
	onBank := win.selectedBank == currBank.Number

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("##bank %d", win.selectedBank), imgui.Vec2{X: 0, Y: height}, false, imgui.WindowFlagsAlwaysVerticalScrollbar)

	win.img.dbg.Disasm.BorrowDisasm(func(dsmEntries *disassembly.DisasmEntries) {
		// disassembly is not valid and so dsmEntires is nil
		if dsmEntries == nil {
			imgui.Text("No disassembly available")
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
		iterateDraw := func(e *disassembly.Entry) bool {
			if headerRequired {
				imgui.EndTable()
				imgui.Text(fmt.Sprintf("Bank %d", iterateBank))
				if !win.startTable() {
					return false
				}
				headerRequired = false
			}
			win.drawEntry(currBank, e, focusAddr, onBank, iterateBank)
			return true
		}

		// alter iterate functions according to selected filter
		switch win.filter {
		case filterBank:
			iterateFilter = func(e *disassembly.Entry) bool {
				if e == nil {
					return false
				}
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
			iterateDraw = func(e *disassembly.Entry) bool {
				if e == nil {
					return true
				}
				win.drawLabel(e, iterateBank)
				win.drawEntry(currBank, e, focusAddr, onBank, iterateBank)
				if currBank.ExecutingCoprocessor && onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					if !win.drawEntryCoProcessorExecution() {
						return false
					}
				}
				return true
			}
		case filterCPUBug:
			iterateFilter = func(e *disassembly.Entry) bool {
				if e == nil {
					return false
				}
				return e.Level >= disassembly.EntryLevelExecuted && e.Result.CPUBug != ""
			}
		case filterPageFault:
			iterateFilter = func(e *disassembly.Entry) bool {
				if e == nil {
					return false
				}
				return e.Level >= disassembly.EntryLevelExecuted && e.Result.PageFault
			}
		}

		// nothing to do so return immediately
		if dsmEntries == nil {
			return
		}

		// number of blessed entries in disasm. being careful to include labels in the count
		ct := 0
		focusCt := 0
		focusCtApply := false

		iterateReset()
		e := dsmEntries.Entries[iterateBank][iterateIdx]
		for {
			if iterateFilter(e) {
				ct++
				if e.Label.Resolve() != "" {
					ct++
				}

				if win.focusOnAddr && onBank && e.Result.Address&memorymap.CartridgeBits == focusAddr {
					focusCt = ct
					focusCtApply = true
				}
			}

			if !iterateNext() {
				break
			}
			e = dsmEntries.Entries[iterateBank][iterateIdx]
		}

		// no entries in disassembly. display message and exit borrow early
		if ct == 0 {
			imgui.Text("Disassembly not available")
			return
		}

		if !win.startTable() {
			return
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
					if e.Label.Resolve() != "" {
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
		if onBank && focusCtApply {
			y := imgui.FontSize() + imgui.CurrentStyle().ItemInnerSpacing().Y

			// leave a small gap between the top of the scroll window and
			// the focused entry
			const focusGap = 7

			y = float32(focusCt-focusGap) * y
			imgui.SetScrollY(y)

			// address has been focused so we turn off the focus flag - it will
			// be set again if necessary
			win.focusOnAddr = false
		}

		// dummy entry at end of table. stops a "bouncing" effect and also
		// allows the last entry in the disassembly to be seen.
		//
		// not a good solution but it works. i'm sure the real cause of the
		// problem is in somewhere in the ListClipper loop
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.Text("")

		imgui.EndTable()
	})

	imgui.EndChild()
}

func (win *winDisasm) drawLabel(e *disassembly.Entry, bank int) bool {
	if len(e.Label.Resolve()) == 0 {
		return true
	}

	// end existing disasm table before drawing label. (re)start table before
	// the end of the function
	imgui.EndTable()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	imgui.SelectableV("", false, imgui.SelectableFlagsNone, imgui.Vec2{0, 0})
	imgui.SameLine()
	imgui.Text(e.Label.Resolve())
	imgui.PopStyleColorV(2)

	return win.startTable()
}

func (win *winDisasm) drawCoProcTooltip() {
	win.img.imguiTooltipSimple("Coprocessor is executing")
}

func (win *winDisasm) drawEntryCoProcessorExecution() bool {
	imgui.EndTable()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	imgui.SelectableV("", false, imgui.SelectableFlagsNone, imgui.Vec2{0, 0})
	win.drawCoProcTooltip()
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("    %c 6507 will resume here", fonts.CoProcExecution))
	imgui.PopStyleColorV(2)

	return win.startTable()
}

func (win *winDisasm) drawEntry(currBank mapper.BankInfo, e *disassembly.Entry, focusAddr uint16, onBank bool, bank int) {
	imgui.TableNextRow()

	// highligh current PC entry
	if onBank && (e.Result.Address&memorymap.CartridgeBits == focusAddr) {
		imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.DisasmStep)

		// focused entry has been drawn so unset focus flag
		win.focusOnAddr = false
	}

	// does this entry/address have a PC break applied to it
	var hasPCbreak bool
	if win.img.cache.Dbg.Breakpoints != nil {
		hasPCbreak, _ = win.img.cache.Dbg.Breakpoints.HasPCBreak(e.Result.Address, bank)
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

	// first column contains the breakpoint indicator and is also the selectable for the entire row
	if hasPCbreak {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
		imgui.SelectableV(string(fonts.Breakpoint), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
		imgui.PopStyleColor()
	} else {
		imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
	}

	// single click on the address entry toggles a PC breakpoint
	if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
		win.img.dbg.PushTogglePCBreak(e)
	}

	// tooltip on hover and context menu on right mouse button
	if imgui.IsItemHovered() {
		win.img.imguiTooltip(func() {
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
			imgui.Text(e.Operand.Resolve())
			imgui.PopStyleColor()

			// treat an instruction that is "cycling" differently
			if !e.Result.Final {
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
				imgui.Text(fmt.Sprintf("%c cycling instruction (%s)", fonts.Paw, e.Cycles()))
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

	// use the "no color" color if usingColour is false. without this, the
	// assembly is just a wall of white text, which is too harsh IMO
	if !win.usingColor {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNoColour)
		defer imgui.PopStyleColor()
	}

	// address column
	imgui.TableNextColumn()
	if win.usingColor {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
		defer imgui.PopStyleColor()
	}
	imgui.Text(e.Address)

	// operator column
	imgui.TableNextColumn()
	if win.usingColor {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
		defer imgui.PopStyleColor()
	}
	imgui.Text(e.Operator)

	// operand column
	imgui.TableNextColumn()
	if win.usingColor {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
		defer imgui.PopStyleColor()
	}
	imgui.Text(e.Operand.Resolve())

	// cycles column
	imgui.TableNextColumn()
	if win.usingColor {
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmCycles)
		defer imgui.PopStyleColor()
	}
	if e.Result.Defn != nil {
		imgui.Text(e.Result.Defn.Cycles.Formatted)
	}

	// notes column
	imgui.TableNextColumn()

	// test to see if cycling instructions icon should be displayed in the notes column
	//
	// 1) not the final result in the CPU
	// 2) the CPU has not been reset
	// 3) the coprocessor is not being executed
	// 4) the entry to be displayed is the same as the one in the CPU bank
	//		and address
	if !win.img.cache.VCS.CPU.LastResult.Final && !win.img.cache.VCS.CPU.HasReset() && !currBank.ExecutingCoprocessor {
		exeAddress := win.img.cache.VCS.CPU.LastResult.Address & memorymap.CartridgeBits
		entryAddress := e.Result.Address & memorymap.CartridgeBits
		if exeAddress == entryAddress && currBank.Number == bank {
			imgui.Text(string(fonts.Paw))
		}
	} else if e.Level == disassembly.EntryLevelExecuted {
		if win.usingColor {
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmNotes)
			defer imgui.PopStyleColor()
		}
		if e.Result.CPUBug != "" {
			imgui.Text(string(fonts.CPUBug))
		} else if e.Result.PageFault {
			imgui.Text(string(fonts.PageFault))
		}
	}
}
