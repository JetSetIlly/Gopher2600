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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/symbols"
)

// EntryLevel describes the level of the Entry
type EntryLevel int

// List of valid EntryL in increasing reliability.
//
// Decoded entries have been decoded as though every byte point is a valid
// instruction. Blessed entries meanwhile take into consideration the preceeding
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
	EntryLevelUnused EntryLevel = iota
	EntryLevelDecoded
	EntryLevelBlessed
	EntryLevelExecuted
)

func (t EntryLevel) String() string {
	// adding space to short strings so that they line up (we're only using
	// this in a single place for a specific purpose so this is okay)
	switch t {
	case EntryLevelUnused:
		return "unused "
	case EntryLevelDecoded:
		return "decoded "
	case EntryLevelBlessed:
		return "blessed "
	case EntryLevelExecuted:
		return "executed "
	}

	return ""
}

// Entry is a disassambled instruction. The constituent parts of the
// disassembly. It is a represenation of execution.Instruction
type Entry struct {
	// the level of reliability of the information in the Entry
	Level EntryLevel

	// execution.Result does not specify which bank the instruction is from
	// because that information isn't available to the CPU. we note it here if
	// possible.
	Bank banks.Details

	// /\/\ the fields above are not set by newEntry() they should be set
	// manually when newEntry() returns

	// \/\/ the entries below are not defined if Level == EntryLevelUnused

	// copy of the CPU execution
	Result execution.Result

	// the remaining fields are not valid for dead entries

	// formatted strings representations of information in execution.Result
	Location string
	Bytecode string
	Address  string
	Mnemonic string
	Operand  string

	// formatted cycles and notes information from instructions.Defintion
	DefnCycles string
	DefnNotes  string

	// actual cycles and notes are the cycles and notes actually seen in
	// the computation
	ActualCycles string
	ActualNotes  string

	// does the entry represent an instruction that might have different
	// "actual" strings depending on the specifics of execution. practically,
	// this means branch and page-sensitive instructions
	UpdateActualOnExecute bool
}

// String returns a very basic representation of an Entry. Provided for
// convenience. Probably not of any use except for the simplest of tools.
func (e *Entry) String() string {
	return fmt.Sprintf("%s %s %s", e.Address, e.Mnemonic, e.Operand)
}

// FormatResult It is the preferred method of initialising for the Entry type.
// It creates a disassembly.Entry based on the bank and result information.
func (dsm *Disassembly) FormatResult(bank banks.Details, result execution.Result, level EntryLevel) (*Entry, error) {
	// protect against empty definitions. we shouldn't hit this condition from
	// the disassembly package itself, but it is possible to get it from ad-hoc
	// formatting from GUI interfaces (see CPU window in sdlimgui)
	if result.Defn == nil {
		return &Entry{}, nil
	}

	return dsm.formatResult(bank, result, level)
}

// the guts of FormatResult(). we use this within the disassembly package when
// we're sure we don't need the additional special condition handling
func (dsm *Disassembly) formatResult(bank banks.Details, result execution.Result, level EntryLevel) (*Entry, error) {
	e := &Entry{
		Result: result,
		Level:  level,
		Bank:   bank,
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// look up address in symbol table
	if v, ok := dsm.Symtable.Locations.Symbols[result.Address]; ok {
		e.Location = v
	}

	// mnemonic is just a string anyway
	e.Mnemonic = result.Defn.Mnemonic

	// bytecode and operand string is assembled depending on the number of
	// expected bytes (result.Defn.Bytes) and the number of bytes read so far
	// (result.ByteCount).
	//
	// the panics cover situations that should never exists. if result
	// validation is active then the panic situations will have been caught
	// then. if validation is not running then the code could theoretically
	// panic but that's okay, they should have been caught in testing.
	var operand uint16
	var operandDecoded bool
	switch result.Defn.Bytes {
	case 3:
		switch result.ByteCount {
		case 3:
			operandDecoded = true
			operand = result.InstructionData
			e.Operand = fmt.Sprintf("$%04x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0x00ff, operand&0xff00>>8)
		case 2:
			operand = result.InstructionData
			e.Operand = fmt.Sprintf("$??%02x", result.InstructionData)
			e.Bytecode = fmt.Sprintf("%02x %02x ?? ", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand = "$????"
			e.Bytecode = fmt.Sprintf("%02x ?? ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 2:
		switch result.ByteCount {
		case 2:
			operandDecoded = true
			operand = result.InstructionData
			e.Operand = fmt.Sprintf("$%02x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand = "$??"
			e.Bytecode = fmt.Sprintf("%02x ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 1:
		switch result.ByteCount {
		case 1:
			e.Bytecode = fmt.Sprintf("%02x", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we shoud not be able to read more bytes than the expected number")
		}
	case 0:
		panic("instructions of zero bytes is not possible")
	default:
		panic("instructions of more than 3 bytes is not possible")
	}
	e.Bytecode = strings.TrimSpace(e.Bytecode)

	// use symbol for the operand if available/appropriate. we should only do
	// this if operand has been decoded
	if operandDecoded {
		if e.Operand == "" || e.Operand[0] != '?' {
			if result.Defn.AddressingMode != instructions.Immediate {

				switch result.Defn.Effect {
				case instructions.Flow:
					if result.Defn.IsBranch() {
						e.Operand = formatBranchOperand(e.Result.Address, operand, e.Result.ByteCount, dsm.Symtable)
					} else {
						if v, ok := dsm.Symtable.Locations.Symbols[operand]; ok {
							e.Operand = v
						}
					}
				case instructions.Read:
					mappedOperand, _ := memorymap.MapAddress(operand, true)
					if v, ok := dsm.Symtable.Read.Symbols[mappedOperand]; ok {
						e.Operand = v
					}
				case instructions.Write:
					fallthrough
				case instructions.RMW:
					mappedOperand, _ := memorymap.MapAddress(operand, false)
					if v, ok := dsm.Symtable.Write.Symbols[mappedOperand]; ok {
						e.Operand = v
					}
				}
			}
		}
	}

	// decorate operand with addressing mode indicators
	switch result.Defn.AddressingMode {
	case instructions.Implied:
	case instructions.Immediate:
		e.Operand = fmt.Sprintf("#%s", e.Operand)
	case instructions.Relative:
	case instructions.Absolute:
	case instructions.ZeroPage:
	case instructions.Indirect:
		e.Operand = fmt.Sprintf("(%s)", e.Operand)
	case instructions.IndexedIndirect:
		e.Operand = fmt.Sprintf("(%s,X)", e.Operand)
	case instructions.IndirectIndexed:
		e.Operand = fmt.Sprintf("(%s),Y", e.Operand)
	case instructions.AbsoluteIndexedX:
		e.Operand = fmt.Sprintf("%s,X", e.Operand)
	case instructions.AbsoluteIndexedY:
		e.Operand = fmt.Sprintf("%s,Y", e.Operand)
	case instructions.ZeroPageIndexedX:
		e.Operand = fmt.Sprintf("%s,X", e.Operand)
	case instructions.ZeroPageIndexedY:
		e.Operand = fmt.Sprintf("%s,Y", e.Operand)
	default:
	}

	// definintion cycles
	if result.Defn.IsBranch() {
		e.DefnCycles = fmt.Sprintf("%d/%d", result.Defn.Cycles, result.Defn.Cycles+1)
	} else {
		e.DefnCycles = fmt.Sprintf("%d", result.Defn.Cycles)
	}

	if level == EntryLevelExecuted {
		e.updateActual()
	}

	// note instructions that required active updating on execution
	e.UpdateActualOnExecute = result.Defn.IsBranch() || result.Defn.PageSensitive

	return e, nil
}

// build entry fields that are really dependent on accurate, actual execution,
// rather than a fake disassembly execution. these fields will likely be
// updated frequently during the course of a real execution
func (e *Entry) updateActual() {
	// actual cycles
	e.ActualCycles = fmt.Sprintf("%d", e.Result.ActualCycles)

	// actual notes
	s := strings.Builder{}

	if e.Result.PageFault {
		s.WriteString("[+1] ")
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

	e.ActualNotes = strings.TrimSpace(s.String())
}

// format and return a formatted  branch operand. if a symbol is available for
// the target address then that will be used, otherwise use the target address
// rather than the offset value.
func formatBranchOperand(addr uint16, operand uint16, bytes int, symtable *symbols.Table) string {
	// relative labels. to get the correct label we have to
	// simulate what a successful branch instruction would do:

	// create a mock register with the instruction's address as the initial value
	pc := registers.NewProgramCounter(addr)

	// add the number of instruction bytes to get the PC as
	// it would be at the end of the instruction
	pc.Add(uint16(bytes))

	// because we're doing 16 bit arithmetic with an 8bit
	// value, we need to make sure the sign bit has been
	// propogated to the more-significant bits
	if operand&0x0080 == 0x0080 {
		operand |= 0xff00
	}

	// add the 2s-complement value to the mock program
	// counter
	pc.Add(operand)

	// look up mock program counter value in symbol table
	if v, ok := symtable.Locations.Symbols[pc.Address()]; ok {
		return v
	}

	return fmt.Sprintf("$%04x", pc.Address())

}
