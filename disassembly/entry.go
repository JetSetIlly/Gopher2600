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
)

// EntryLevel describes the level of the Entry.
type EntryLevel int

// List of valid EntryL in increasing reliability.
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
// For normal debugging operations there is no need to use EntryLevelUnused
// outside of the disassembly package. It used for the unusual case where a
// bank is not able to be referenced from the Entry address. See M-Network for
// an example of this, where Bank 7 cannot be mapped to the lower segment.
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

	// the level of reliability of the information in the Entry
	Level EntryLevel

	// execution.Result does not specify which bank the instruction is from
	// because that information isn't available to the CPU. we note it here if
	// possible.
	Bank mapper.BankInfo

	// the entries below are not defined if Level == EntryLevelUnused

	// copy of the CPU execution. must not be updated except through
	// updateExecutionEntry() function
	Result execution.Result

	// string representations of information in execution.Result. GetField()
	// will apply white-space padding and should be preferred in most
	// instances.
	Label    Label
	Bytecode string
	Address  string
	Operator string
	Operand  Operand

	// formatted cycles information from instructions.Defintion
	DefnCycles string

	// actual number of cycles. consider using Cycles() for presentation
	// purposes
	ActualCycles string

	// information about the most recent execution of the entry
	//
	// should be empty if EntryLevel != EntryLevelExecuted
	ExecutionNotes string
}

// some fields in the disassembly entry are updated on every execution.
func (e *Entry) updateExecutionEntry(result execution.Result) {
	e.Result = result

	// update result instance in Label and Operand fields
	e.Label.result = e.Result
	e.Operand.result = e.Result

	// indicate that entry has been executed
	e.Level = EntryLevelExecuted

	// actual cycles
	e.ActualCycles = fmt.Sprintf("%d", e.Result.Cycles)

	// actual notes
	s := strings.Builder{}

	if e.Result.PageFault {
		s.WriteString("page-fault [+1] ")
	}

	if e.Result.Defn.IsBranch() {
		if e.Result.BranchSuccess {
			s.WriteString("branched")
		} else {
			s.WriteString("next")
		}
	}

	if e.Result.CPUBug != "" {
		s.WriteString(e.Result.CPUBug)
		s.WriteString(" ")
	}

	e.ExecutionNotes = strings.TrimSpace(s.String())
}

// Cycles returns the number of cycles annotated if actual cycles differs from
// the number of cycles in the definition. for executed branch instructions this
// will always be the case.
func (e *Entry) Cycles() string {
	if e.Level < EntryLevelExecuted || e.DefnCycles == e.ActualCycles {
		return e.DefnCycles
	}

	// if entry hasn't been executed yet or if actual cycles is different to
	// the cycles defined for the entry then return an annotated string
	return fmt.Sprintf("%s [%s]", e.DefnCycles, e.ActualCycles)
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
	bank   int
}

func (l Label) String() string {
	s, _ := l.genString()
	return s
}

// genString returns the address label as a symbol (if a symbol is available)
// if a symbol is not available then the the bool return value will be false.
func (l Label) genString() (string, bool) {
	if l.dsm.Prefs.Symbols.Get().(bool) {
		ma, _ := memorymap.MapAddress(l.result.Address, true)
		if v, ok := l.dsm.Symbols.GetLabel(l.bank, ma); ok {
			return v, true
		}
	}

	return "", false
}

// Operand implements the Stringer interface. The String() implementation
// returns the operand (with symbols if appropriate). Use GetField function for
// white-space padded operand string.
type Operand struct {
	nonSymbolic string
	dsm         *Disassembly
	result      execution.Result
	bank        int
}

func (l Operand) String() string {
	return l.genString()
}

// genString returns the operand as a symbol (if a symbol is available) if
// a symbol is not available then the the bool return value will be false.
func (l Operand) genString() string {
	if !l.dsm.Prefs.Symbols.Get().(bool) {
		return l.nonSymbolic
	}

	s := l.nonSymbolic

	// use symbol for the operand if available/appropriate. we should only do
	// this if operand has been decoded
	if l.result.Defn.AddressingMode == instructions.Immediate {
		// TODO: immediate symbols

	} else if l.result.ByteCount > 1 {
		// instruction data is only valid if bytecount is 2 or more

		operand := l.result.InstructionData

		switch l.result.Defn.Effect {
		case instructions.Flow:
			if l.result.Defn.IsBranch() {
				operand = absoluteBranchDestination(l.result.Address, operand)

				// look up mock program counter value in symbol table
				if v, ok := l.dsm.Symbols.GetLabel(l.bank, operand); ok {
					s = v
				}
			} else if v, ok := l.dsm.Symbols.GetLabel(l.bank, operand); ok {
				s = addrModeDecoration(v, l.result.Defn.AddressingMode)
			}
		case instructions.Read:
			if v, ok := l.dsm.Symbols.GetReadSymbol(operand); ok {
				s = addrModeDecoration(v, l.result.Defn.AddressingMode)
			}

		case instructions.Write:
			fallthrough

		case instructions.RMW:
			if v, ok := l.dsm.Symbols.GetWriteSymbol(operand); ok {
				s = addrModeDecoration(v, l.result.Defn.AddressingMode)
			}
		}
	}

	return s
}
