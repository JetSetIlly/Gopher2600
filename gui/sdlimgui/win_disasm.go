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

	"github.com/jetsetilly/imgui-go/v5"
)

const winDisasmID = "Disassembly"

type disasmFilter int

const (
	filterBank disasmFilter = iota
	filterCPUBug
	filterPageFault
)

// focusing the disassembly scroll view on the correct address is tricky. all
// the information we need to track is in this type
type disasmFocus struct {
	active      bool
	lastState   govern.State
	lastAddress uint16
}

type winDisasm struct {
	debuggerWin

	img *SdlImgui

	// height of options line at bottom of window. valid after first frame
	optionsHeight float32

	// options
	followCPU  bool
	usingColor bool

	// selected filter and selected bank to display when filter is 'filterBank'
	filter       disasmFilter
	selectedBank int

	// special handling of a new ROM with a different number of banks is achieved by
	// checking whether the CPU has been recently reset. this flag allows us to
	// change the selected bank after a CPU has been reset and before it's been
	// executed (the CPU will report being reset until the first instruction has
	// been executed)
	reset bool

	// flag stating whether bank combo is open
	selectedBankComboOpen bool

	// focusing the disassembly scroll view on the correct address is tricky
	focus disasmFocus

	// widths of columns in the disasm table
	// widthOperands is implied and is the width of the window minus widthSum
	widthBreak    float32
	widthLabel    float32
	widthAddr     float32
	widthOperator float32
	widthCycles   float32
	widthNotes    float32

	// sum of all the widths above
	widthSum float32

	// widths of focus button in control bar. bank selector column takes the remainder of the space
	widthFocusCurrent float32
}

func newWinDisasm(img *SdlImgui) (window, error) {
	win := &winDisasm{
		img:       img,
		followCPU: true,
	}
	return win, nil
}

func (win *winDisasm) init() {
	// width of the focus button in the top toolbar of the window
	win.widthFocusCurrent = imgui.CalcTextSize(string(fonts.DisasmFocusCurrent), true, 0).X

	// the widths of the columns in the disassembly table
	win.widthBreak = imgui.CalcTextSize(string(fonts.Breakpoint), true, 0).X
	win.widthLabel = imgui.CalcTextSize(string(fonts.Label), true, 0).X
	win.widthAddr = imgui.CalcTextSize("$FFFF ", true, 0).X
	win.widthOperator = imgui.CalcTextSize("AND ", true, 0).X
	win.widthCycles = imgui.CalcTextSize("2/3 ", true, 0).X
	win.widthNotes = imgui.CalcTextSize(string(fonts.CPUBug), true, 0).X

	// we need to take into account the possibility of a scrollbar
	scrollbar := imgui.CalcTextSize("  ", true, 0).X

	// the total width of the disassembly table
	win.widthSum = win.widthBreak + win.widthAddr + win.widthOperator + win.widthCycles + win.widthNotes + scrollbar
}

func (win *winDisasm) id() string {
	return winDisasmID
}

func (win *winDisasm) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 1021, Y: 34}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 552}, imgui.ConditionFirstUseEver)
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

	// handle a change of cartridge by monitoring the CPU reset flag. this gives us
	// the opportunity to change the selectedBank value
	if win.img.cache.VCS.CPU.HasReset() {
		if !win.reset {
			win.selectedBank = currBank.Number
			win.reset = true
		}
	} else {
		win.reset = false
	}

	// focus on address if the state has changed to the paused state and
	// followCPU is set; or the PC has changed (this is because the state
	// change might be missed)
	//
	// using the lazy govern.State value rather than the live state - the
	// live state can cause synchronisation problems meaning focus is lost
	if win.followCPU {
		if (win.img.dbg.State() == govern.Paused && win.focus.lastState != govern.Paused) ||
			win.img.cache.VCS.CPU.PC.Address() != win.focus.lastAddress {

			win.focus.active = true
			win.selectedBank = currBank.Number
		}
	}
	win.focus.lastAddress = win.img.cache.VCS.CPU.PC.Address()
	win.focus.lastState = win.img.dbg.State()

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
		if win.img.cache.Dbg.LiveDisasmEntry.Result.Final {
			focusAddr = win.img.cache.VCS.CPU.PC.Address() & memorymap.CartridgeBits
		} else {
			focusAddr = win.img.cache.Dbg.LiveDisasmEntry.Result.Address & memorymap.CartridgeBits
		}
	}

	win.drawControlBar(currBank)
	win.drawBank(currBank, focusAddr)
	win.drawOptionsBar(currBank)
}

func (win *winDisasm) drawControlBar(currBank mapper.BankInfo) {
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	numColumns := 2
	imgui.BeginTableV("##controlBar", numColumns, flgs, imgui.Vec2{}, 0)

	bankWidth := imgui.ContentRegionAvail().X - imgui.CurrentStyle().ItemSpacing().X*float32(numColumns)
	bankWidth -= win.widthFocusCurrent
	imgui.TableSetupColumnV("focus", imgui.TableColumnFlagsNone, win.widthFocusCurrent, 0)
	imgui.TableSetupColumnV("bank", imgui.TableColumnFlagsNone, bankWidth, 1)

	imgui.TableNextRow()

	// focus on current CPU address
	imgui.TableNextColumn()
	imgui.AlignTextToFramePadding()
	imgui.Text(string(fonts.DisasmFocusCurrent))
	if imgui.IsItemHovered() {
		if imgui.IsItemClicked() {
			win.focus.active = true
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

func (win *winDisasm) drawBank(currBank mapper.BankInfo, focusAddr uint16) {
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	defer imgui.PopStyleColorV(2)

	height := imguiRemainingWinHeight() - win.optionsHeight
	imgui.BeginChildV(fmt.Sprintf("##bank %d", win.selectedBank), imgui.Vec2{X: 0, Y: height}, false, imgui.ChildFlagsNone)
	defer imgui.EndChild()

	numColumns := 7
	flgs := imgui.TableFlagsNone
	flgs |= imgui.TableFlagsSizingFixedFit
	if !imgui.BeginTableV("disasmbank", numColumns, flgs, imgui.Vec2{}, 0) {
		return
	}
	defer imgui.EndTable()

	operandWidth := imgui.ContentRegionAvail().X - imgui.CurrentStyle().ItemSpacing().X*float32(numColumns)
	operandWidth -= win.widthSum

	imgui.TableSetupColumnV("##break", imgui.TableColumnFlagsNone, win.widthBreak, 0)
	imgui.TableSetupColumnV("##label", imgui.TableColumnFlagsNone, win.widthLabel, 1)
	imgui.TableSetupColumnV("##address", imgui.TableColumnFlagsNone, win.widthAddr, 2)
	imgui.TableSetupColumnV("##operator", imgui.TableColumnFlagsNone, win.widthOperator, 3)
	imgui.TableSetupColumnV("##operand", imgui.TableColumnFlagsNone, operandWidth, 4)
	imgui.TableSetupColumnV("##cycles", imgui.TableColumnFlagsNone, win.widthCycles, 5)
	imgui.TableSetupColumnV("##notes", imgui.TableColumnFlagsNone, win.widthNotes, 6)

	// draw is called for each column. it handles the colour preference
	draw := func(s string, col imgui.Vec4) {
		if win.usingColor {
			imgui.PushStyleColor(imgui.StyleColorText, col)
			defer imgui.PopStyleColor()
		}
		imgui.Text(s)
	}

	// render is called via a call to BorrowDisasm()
	render := func(dsm *disassembly.DisasmEntries) {
		// because we're running concurrently with the emulation there may be instances
		// current bank number is out of date when compared to the disassembly. this can
		// happen when loading a new ROM with fewer banks than the previous ROM
		if currBank.Number >= len(dsm.Entries) {
			return
		}

		var current int

		// pre-filter blessed entries
		var entries []*disassembly.Entry
		for _, e := range dsm.Entries[currBank.Number] {
			if e == nil {
				continue
			}
			if e.Level >= disassembly.EntryLevelBlessed {
				switch win.filter {
				case filterBank:
					entries = append(entries, e)
				case filterCPUBug:
					if e.Result.CPUBug != "" {
						entries = append(entries, e)
					}
				case filterPageFault:
					if e.Result.PageFault {
						entries = append(entries, e)
					}
				}
			}

			if e.Result.Address&memorymap.CartridgeBits == focusAddr {
				current = len(entries) - 1
			}
		}

		results := imgui.ListClipperAll(len(entries), func(i int) {
			lbl := entries[i].Label.Resolve()
			nts := entries[i].Notes()

			// does this entry/address have a PC break applied to it
			var hasPCbreak bool
			if win.img.cache.Dbg.Breakpoints != nil {
				hasPCbreak, _ = win.img.cache.Dbg.Breakpoints.HasPCBreak(entries[i].Result.Address, currBank.Number)
			}

			imgui.TableNextRow()
			if imgui.TableNextColumn() {
				if hasPCbreak {
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
					imgui.SelectableV(string(fonts.Breakpoint), i == current, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{X: 0, Y: 0})
					imgui.PopStyleColor()
				} else {
					imgui.SelectableV("", i == current, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{X: 0, Y: 0})
				}

				// single click on the address entry toggles a PC breakpoint
				if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
					win.img.dbg.PushTogglePCBreak(entries[i])
				}

				// tooltip information about the instruction
				win.img.imguiTooltip(func() {
					if lbl != "" {
						imgui.Spacing()
						imgui.Text(fmt.Sprintf("%c %s", fonts.Label, lbl))
					}
					if imgui.BeginTableV("disasmtooltip", 4, imgui.TableFlagsBorders, imgui.Vec2{}, 0) {
						imgui.TableSetupColumn("Bytecode")
						imgui.TableSetupColumn("Address")
						imgui.TableSetupColumn("Operator")
						imgui.TableSetupColumn("Operand")
						imgui.TableHeadersRow()
						imgui.TableNextRow()
						imgui.TableNextColumn()
						draw(entries[i].Bytecode, win.img.cols.DisasmByteCode)
						imgui.TableNextColumn()
						draw(entries[i].Address, win.img.cols.DisasmAddress)
						imgui.TableNextColumn()
						draw(entries[i].Operator, win.img.cols.DisasmOperator)
						imgui.TableNextColumn()
						draw(entries[i].Operand.Resolve(), win.img.cols.DisasmOperand)
						imgui.EndTable()
					}
					if hasPCbreak {
						imgui.Spacing()
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
						imgui.Text(string(fonts.Breakpoint))
						imgui.PopStyleColor()
						imgui.SameLine()
						imgui.Textf("breakpoint on %s", entries[i].Address)
					}
					if current == i && currBank.ExecutingCoprocessor {
						imgui.Spacing()
						draw(fmt.Sprintf("%c coprocessor executing", fonts.CoProcExecution), win.img.cols.DisasmNotes)
					}
					if entries[i].Level < disassembly.EntryLevelExecuted {
						imgui.Spacing()
						draw("never been executed", win.img.cols.DisasmNotes)
					} else {
						imgui.Spacing()
						draw(fmt.Sprintf("last took %s cycles", entries[i].Cycles()), win.img.cols.DisasmCycles)
					}
					if nts != "" {
						imgui.Spacing()
						draw(fmt.Sprintf("%c %s", fonts.Notes, nts), win.img.cols.DisasmNotes)
					}
				}, true)
			}
			if imgui.TableNextColumn() {
				if lbl != "" {
					draw(string(fonts.Label), win.img.cols.DisasmLabel)
				}
			}
			if imgui.TableNextColumn() {
				draw(entries[i].Address, win.img.cols.DisasmAddress)
			}
			if imgui.TableNextColumn() {
				draw(entries[i].Operator, win.img.cols.DisasmOperator)
			}
			if imgui.TableNextColumn() {
				draw(entries[i].Operand.Resolve(), win.img.cols.DisasmOperand)
			}
			if imgui.TableNextColumn() {
				draw(entries[i].Cycles(), win.img.cols.DisasmCycles)
			}
			if imgui.TableNextColumn() {
				if current == i && currBank.ExecutingCoprocessor {
					draw(string(fonts.CoProcExecution), win.img.cols.DisasmNotes)
				} else {
					if nts != "" {
						draw(string(fonts.Notes), win.img.cols.DisasmNotes)
					}
				}
			}
		})

		if win.focus.active {
			const margin = 3
			imgui.SetScrollY(float32(current-margin) * results.ItemsHeight)
			win.focus.active = false
		}
	}

	if !win.img.dbg.Disasm.BorrowDisasm(render) {
		imgui.Text("disassembling...")
	}
}

func (win *winDisasm) drawOptionsBar(currBank mapper.BankInfo) {
	// draw options and status line. start height measurement
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		if imgui.Checkbox("Follow CPU", &win.followCPU) {
			// set focus options immediately
			if win.followCPU {
				win.focus.active = true
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
			win.img.imguiTooltipSimple("Coprocessor is executing")
		}
		if currBank.NonCart {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text(string(fonts.NonCartExecution))
			win.img.imguiTooltipSimple("Executing a non-cartridge address!")
		}
	})
}
