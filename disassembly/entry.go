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

package disassembly

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// EntryLevel describes the level of the Entry.
type EntryLevel int

// List of valid EntryLevel in increasing reliability.
//
// Decoded entries have been decoded as though every byte point is a valid
// instruction. Blessed entries meanwhile take into consideration the preceding
// instruction and the number of bytes it would have consumed.
//
// Decoded entries are useful in the event of the CPU landing on an address that
// didn't look like an instruction at disassembly time.
//
// Blessed instructions are deemed to be more accurate because they have been
// reached according to the flow of the instructions from the start address.
//
// For normal debugging operations there is no need to use EntryLevelUnmappable
// outside of the disassembly package. It used for the unusual case where a bank
// is not able to be referenced from the Entry address. See M-Network for an
// example of this, where Bank 7 cannot be mapped to the lower segment.
const (
	EntryLevelUnmappable EntryLevel = iota
	EntryLevelDecoded
	EntryLevelBlessed
	EntryLevelExecuted
)

// Entry is a disassambled instruction. The constituent parts of the
// disassembly. It is a representation of execution.Instruction.
type Entry struct {
	dsm *Disassembly

	// the bank this entry belongs to. note that this is just the bank number;
	// we're not storing a copy of mapper.BankInfo. that's not needed for
	// disassembly purposes
	Bank int

	// the level of reliability of the information in the Entry.
	//
	// note that it is possible for EntryLevelExecuted entries to be partially
	// executed. check Result.Final if required.
	Level EntryLevel

	// copy of the CPU execution. must not be updated except through
	// updateExecutionEntry() function.
	//
	// not that the the Final field of execution.Result may be false is the
	// emulation is stopped mid-execution.
	Result execution.Result

	// the entries below are not defined if Level == EntryLevelUnused

	// string representations of information in execution.Result
	// entry.GetField() will apply white spacing padding suitable for columnation
	Label    Label
	Bytecode string
	Address  string
	Operator string
	Operand  Operand

	// when the disassembly entry was created
	Coords coords.TelevisionCoords
}

// some fields in the disassembly entry are updated on every execution.
func (e *Entry) updateExecutionEntry(result execution.Result) {
	// update result instance
	e.Result = result

	// update result instance in Label. we probably don't need to do this but it
	// might be useful to know what the *actual* address of the instruction. ie.
	// which mirror is used by the program at that point in the execution.
	e.Label.result = e.Result

	// update result instance in Operand
	e.Operand.result = e.Result

	// indicate that entry has been executed
	e.Level = EntryLevelExecuted
}

// Cycles returns the number of cycles annotated if actual cycles differs from
// the number of cycles in the definition. for executed branch instructions this
// will always be the case.
func (e *Entry) Cycles() string {
	// the Defn field may be unassigned
	if e.Result.Defn == nil {
		return "?"
	}

	if e.Level < EntryLevelExecuted {
		return e.Result.Defn.Cycles.Formatted
	}

	if e.Result.Final {
		return fmt.Sprintf("%d", e.Result.Cycles)
	}

	// if entry hasn't been executed yet or if actual cycles is different to
	// the cycles defined for the entry then return an annotated string
	return fmt.Sprintf("%d of %s", e.Result.Cycles, e.Result.Defn.Cycles.Formatted)
}

// Notes returns a string returning notes about the most recent execution. The
// information is made up of the BranchSuccess, PageFault and CPUBug fields.
func (e *Entry) Notes() string {
	if e.Level < EntryLevelExecuted {
		return ""
	}

	if !e.Result.Final {
		return ""
	}

	s := strings.Builder{}

	if e.Result.Defn != nil && e.Result.Defn.IsBranch() {
		if e.Result.BranchSuccess {
			s.WriteString("branch succeeded ")
		} else {
			s.WriteString("branch failed ")
		}

		if e.Result.PageFault {
			s.WriteString("with page-fault ")
		}
	} else {
		if e.Result.PageFault {
			s.WriteString("page-fault ")
		}
	}

	if e.Result.CPUBug != "" {
		s.WriteString(e.Result.CPUBug)
	}

	return s.String()
}

// add decoration to operand according to the addressing mode of the entry.
// operand taken as an argument because it is called from two different contexts.
func addrModeDecoration(operand string, mode instructions.AddressingMode) string {
	s := operand

	switch mode {
	case instructions.Implied:
	case instructions.Immediate:
		s = fmt.Sprintf("#%s", operand)
	case instructions.Relative:
	case instructions.Absolute:
	case instructions.ZeroPage:
	case instructions.Indirect:
		s = fmt.Sprintf("(%s)", operand)
	case instructions.IndexedIndirect:
		s = fmt.Sprintf("(%s,X)", operand)
	case instructions.IndirectIndexed:
		s = fmt.Sprintf("(%s),Y", operand)
	case instructions.AbsoluteIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.AbsoluteIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	case instructions.ZeroPageIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.ZeroPageIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	}

	return s
}

// absolute branch destination returns the branch operand as the address of the
// branched PC, rather than an offset value.
func absoluteBranchDestination(addr uint16, operand uint16) uint16 {
	// create a mock register with the instruction's address as the initial value
	pc := registers.NewProgramCounter(addr)

	// all 6502 branch instructions are 2 bytes in length
	pc.Add(2)

	// because we're doing 16 bit arithmetic with an 8bit value, we need to
	// make sure the sign bit has been propogated to the more-significant bits
	if operand&0x0080 == 0x0080 {
		operand |= 0xff00
	}

	// add the 2s-complement value to the mock program counter
	pc.Add(operand)

	return pc.Address()
}

// Label implements the Stringer interface. The String() implementation
// returns any address label for the entry. Use GetField() function for
// a white-space padded label.
type Label struct {
	dsm    *Disassembly
	result execution.Result
	bank   mapper.BankInfo
}

// Resolve returns the address label as a symbol (if a symbol is available)
// if a symbol is not available then the the bool return value will be false.
func (l Label) Resolve() string {
	if l.dsm.Prefs.Symbols.Get().(bool) {
		ma, _ := memorymap.MapAddress(l.result.Address, true)
		if e, ok := l.dsm.Sym.GetLabel(l.bank.Number, ma); ok {
			return e.Symbol
		}
	}

	return ""
}

// Operand implements the Stringer interface. The String() implementation
// returns the operand (with symbols if appropriate). Use GetField function for
// white-space padded operand string.
type Operand struct {
	dsm    *Disassembly
	result execution.Result
	bank   mapper.BankInfo

	// partial is the operand that will be used as the result from Resolve()
	// when the execution result is not complete (ie. when not enough bytes have
	// been read)
	partial string
}

// Resolve returns the operand as a symbol (if a symbol is available) if a symbol
// is not available then the returned value will be be numeric possibly with
// placeholders for unknown bytes
func (op Operand) Resolve() string {
	if op.result.Defn == nil {
		return op.partial
	}

	if op.dsm == nil || !op.dsm.Prefs.Symbols.Get().(bool) {
		return op.partial
	}

	if op.result.ByteCount != op.result.Defn.Bytes {
		return op.partial
	}

	res := op.partial

	// use symbol for the operand if available/appropriate. we should only do
	// this if at least part of an operand has been decoded
	if op.result.Defn.Bytes > 1 {
		data := op.result.InstructionData

		switch op.result.Defn.Effect {
		case instructions.Flow:
			if op.result.Defn.IsBranch() {
				data = absoluteBranchDestination(op.result.Address, data)

				// look up mock program counter value in symbol table
				if e, ok := op.dsm.Sym.GetLabel(op.bank.Number, data); ok {
					res = e.Symbol
				}
			} else if e, ok := op.dsm.Sym.GetLabel(op.bank.Number, data); ok {
				res = addrModeDecoration(e.Symbol, op.result.Defn.AddressingMode)
			}

		case instructions.Subroutine:
			if e, ok := op.dsm.Sym.GetLabel(op.bank.Number, data); ok {
				res = e.Symbol
			}

		case instructions.Read:
			if e, ok := op.dsm.Sym.GetReadSymbol(data, op.result.Defn.AddressingMode != instructions.Immediate); ok {
				res = addrModeDecoration(e.Symbol, op.result.Defn.AddressingMode)
			}

		case instructions.Write:
			fallthrough

		case instructions.RMW:
			if e, ok := op.dsm.Sym.GetWriteSymbol(data); ok {
				res = addrModeDecoration(e.Symbol, op.result.Defn.AddressingMode)
			}
		}
	}

	return res
}
