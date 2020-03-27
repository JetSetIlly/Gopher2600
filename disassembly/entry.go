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
}

// FormatResult It is the preferred method of initialising for the Entry type.
// It creates a disassembly.Entry based on the bank and result information.
func (dsm *Disassembly) FormatResult(bank int, result execution.Result, level EntryLevel) (*Entry, error) {
	d := &Entry{
		Result:        result,
		Level:         level,
		Bank:          bank,
		BankDecorated: Bank(bank),
	}

	// set BankDecorated correctly
	if memorymap.IsArea(result.Address, memorymap.RAM) {
		d.BankDecorated = BankRAM
	}

	// if the operator hasn't been decoded yet then use placeholder strings for
	// important fields
	if result.Defn == nil {
		d.Bytecode = "??"
		return d, nil
	}

	// address of instruction
	d.Address = fmt.Sprintf("0x%04x", result.Address)

	// look up address in symbol table
	if v, ok := dsm.Symtable.Locations.Symbols[result.Address]; ok {
		d.Location = v
	}

	// mnemonic is just a string anyway
	d.Mnemonic = result.Defn.Mnemonic

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
			d.Operand = fmt.Sprintf("$%04x", operand)
			d.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0xff00>>8, operand&0x00ff)
		case 2:
			d.Operand = fmt.Sprintf("$??%02x", result.InstructionData)
			d.Bytecode = fmt.Sprintf("%02x %02x ??", result.Defn.OpCode, operand&0xff00>>8)
		case 1:
			d.Operand = "$????"
			d.Bytecode = fmt.Sprintf("%02x ?? ??", result.Defn.OpCode)
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
			d.Operand = fmt.Sprintf("$%02x", operand)
			d.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
			d.Operand = "$??"
			d.Bytecode = fmt.Sprintf("%02x ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 1:
		switch result.ByteCount {
		case 1:
			d.Bytecode = fmt.Sprintf("%02x", result.Defn.OpCode)
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
	d.Bytecode = strings.TrimSpace(d.Bytecode)

	// use symbol for the operand if available/appropriate. we should only do
	// this if operand has been decoded
	if operandDecoded {
		if d.Operand == "" || d.Operand[0] != '?' {
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
							d.Operand = v
						}

					} else {
						if v, ok := dsm.Symtable.Locations.Symbols[operand]; ok {
							d.Operand = v
						}
					}
				case instructions.Read:
					if v, ok := dsm.Symtable.Read.Symbols[operand]; ok {
						d.Operand = v
					}
				case instructions.Write:
					fallthrough
				case instructions.RMW:
					if v, ok := dsm.Symtable.Write.Symbols[operand]; ok {
						d.Operand = v
					}
				}
			}
		}
	}

	// decorate operand with addressing mode indicators
	switch result.Defn.AddressingMode {
	case instructions.Implied:
	case instructions.Immediate:
		d.Operand = fmt.Sprintf("#%s", d.Operand)
	case instructions.Relative:
	case instructions.Absolute:
	case instructions.ZeroPage:
	case instructions.Indirect:
		d.Operand = fmt.Sprintf("(%s)", d.Operand)
	case instructions.PreIndexedIndirect:
		d.Operand = fmt.Sprintf("(%s,X)", d.Operand)
	case instructions.PostIndexedIndirect:
		d.Operand = fmt.Sprintf("(%s),Y", d.Operand)
	case instructions.AbsoluteIndexedX:
		d.Operand = fmt.Sprintf("%s,X", d.Operand)
	case instructions.AbsoluteIndexedY:
		d.Operand = fmt.Sprintf("%s,Y", d.Operand)
	case instructions.IndexedZeroPageX:
		d.Operand = fmt.Sprintf("%s,X", d.Operand)
	case instructions.IndexedZeroPageY:
		d.Operand = fmt.Sprintf("%s,Y", d.Operand)
	default:
	}

	// definintion cycles
	if result.Defn.IsBranch() {
		d.DefnCycles = fmt.Sprintf("%d/%d", result.Defn.Cycles, result.Defn.Cycles+1)
	} else {
		d.DefnCycles = fmt.Sprintf("%d", result.Defn.Cycles)
	}

	// definition notes
	if result.Defn.PageSensitive {
		d.DefnNotes = fmt.Sprintf("%s [+1]", d.DefnNotes)
	}

	// actual cycles
	d.ActualCycles = fmt.Sprintf("%d", result.ActualCycles)

	// actual notes
	if result.PageFault {
		d.ActualNotes = fmt.Sprintf("%s [+1]", d.ActualNotes)
	}
	if result.CPUBug != "" {
		d.ActualNotes = fmt.Sprintf("%s * %s *", d.ActualNotes, result.CPUBug)
	}

	return d, nil
}
