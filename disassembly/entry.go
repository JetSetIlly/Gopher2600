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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package disassembly

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Bank refers to the cartridge bank or one of a group of special conditions.
type Bank int

// List of allowed Bank special conditions.
const (
	BankUnknown Bank = -1
	BankRAM     Bank = -2
)

func (b Bank) String() string {
	switch b {
	case BankUnknown:
		return "?"
	case BankRAM:
		return "R"
	}
	return fmt.Sprintf("%d", b)
}

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
const (
	EntryLevelDead EntryLevel = iota
	EntryLevelDecoded
	EntryLevelBlessed
	EntryLevelExecuted
)

func (t EntryLevel) String() string {
	// adding space to short strings so that they line up (we're only using
	// this in a single place for a specific purpose so this is okay)
	switch t {
	case EntryLevelDead:
		return "dead    "
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
	Bank int

	// BankDecorated is a "decorated" instance of the Bank integer. Positive
	// values can be cast to int and treated just like Bank. However,
	// BankDecorated can also take other values that indicate special
	// conditions. The allowed values are defined above.
	BankDecorated Bank

	// /\/\ the fields above are not set by newEntry() they should be set
	// manually when newEntry() returns

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

	// does the entry represent an instruction that might have different "actual"
	// strings depending on the specifics of execution
	UpdateActualOnExecute bool
}

// String returns a very basic representation of an Entry. Provided for
// convenience. Probably not of any use except for the simplest of tools.
func (e *Entry) String() string {
	return fmt.Sprintf("%s %s %s", e.Address, e.Mnemonic, e.Operand)
}

// FormatResult It is the preferred method of initialising for the Entry type.
// It creates a disassembly.Entry based on the bank and result information.
func (dsm *Disassembly) FormatResult(bank int, result execution.Result, level EntryLevel) (*Entry, error) {
	e := &Entry{
		Result:        result,
		Level:         level,
		Bank:          bank,
		BankDecorated: Bank(bank),
	}

	// set BankDecorated correctly
	if memorymap.IsArea(result.Address, memorymap.RAM) {
		e.BankDecorated = BankRAM
	}

	// if the operator hasn't been decoded yet then use placeholder strings for
	// important fields
	if result.Defn == nil {
		e.Bytecode = "??"
		return e, nil
	}

	// address of instruction
	e.Address = fmt.Sprintf("0x%04x", result.Address)

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
			e.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0xff00>>8, operand&0x00ff)
		case 2:
			e.Operand = fmt.Sprintf("$??%02x", result.InstructionData)
			e.Bytecode = fmt.Sprintf("%02x %02x ??", result.Defn.OpCode, operand&0xff00>>8)
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
						// relative labels. to get the correct label we have to
						// simulate what a successful branch instruction would do:

						// 	-- we create a mock register with the instruction's
						// 	address as the initial value
						pc := registers.NewProgramCounter(result.Address)

						// -- add the number of instruction bytes to get the PC as
						// it would be at the end of the instruction
						pc.Add(uint16(result.Defn.Bytes))

						// -- because we're doing 16 bit arithmetic with an 8bit
						// value, we need to make sure the sign bit has been
						// propogated to the more-significant bits
						if operand&0x0080 == 0x0080 {
							operand |= 0xff00
						}

						// -- add the 2s-complement value to the mock program
						// counter
						pc.Add(operand)

						// -- look up mock program counter value in symbol table
						if v, ok := dsm.Symtable.Locations.Symbols[pc.Address()]; ok {
							e.Operand = v
						} else {
							// -- if no symbol exists change operand to the
							// address (rather than the branch offset)
							e.Operand = fmt.Sprintf("$%04x", pc.Address())
						}

					} else {
						if v, ok := dsm.Symtable.Locations.Symbols[operand]; ok {
							e.Operand = v
						}
					}
				case instructions.Read:
					if v, ok := dsm.Symtable.Read.Symbols[operand]; ok {
						e.Operand = v
					}
				case instructions.Write:
					fallthrough
				case instructions.RMW:
					if v, ok := dsm.Symtable.Write.Symbols[operand]; ok {
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
