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
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/cpu/instructions"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/symbols"
	"strings"
)

// EntryType describes the level of reliability of the Entry.
type EntryType int

// List of valid EntryTypes in increasing reliability.
//
// Naive entries have been decoded as though every byte point is a valid
// instruction. Decode entries meanwhile take into consideration the preceeding
// instruction and the number of bytes it would have consumed.
//
// Naive entries are useful in the event of the CPU landing on an address that
// didn't look like an instruction at disassembly time. Unlikely but possible.
//
// Flow instructions are deemed to be more accurate because they have been
// reached according to the flow of the instructions from the start address
// through the CPU.
//
// Live instructions are the most reliable because they contain information
// from the last actual execution of the entire system (not just a mock CPU, as
// in the case of the Flow type)
const (
	EntryTypeNaive EntryType = iota
	EntryTypeDecode
	EntryTypeAnalysis
	EntryTypeLive
)

// Entry is a disassambled instruction. The constituent parts of the
// disassembly. It is a represenation of execution.Instruction
type Entry struct {
	// the level of reliability of the information in the Entry
	Type EntryType

	Result execution.Result

	// execution.Result does not specify which bank the instruction is from
	// because that information isn't available to the CPU. note that
	// information here for completeness
	Bank int

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

	// addresses from which the instruction can be reached
	Prev []uint16

	// address to which the instruction flows to next
	// subroutines
	Next []uint16
}

// format execution.Result and create a new instance of Entry
func newEntry(result execution.Result, symtable *symbols.Table) (*Entry, error) {
	if symtable == nil {
		symtable = &symbols.Table{}
	}

	d := &Entry{
		Result: result,
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
	if v, ok := symtable.Locations.Symbols[result.Address]; ok {
		d.Location = v
	}

	// mnemonic is just a string anyway
	d.Mnemonic = result.Defn.Mnemonic

	// operands
	var operand uint16

	switch v := result.InstructionData.(type) {
	case uint8:
		d.Operand = fmt.Sprintf("$%02x", v)
	case uint16:
		d.Operand = fmt.Sprintf("$%04x", v)
	case nil:
		if result.Defn.Bytes == 2 {
			d.Operand = "??"
		} else if result.Defn.Bytes == 3 {
			d.Operand = "????"
		}
	}

	// Bytecode is assembled depending on the number of expected bytes
	// (result.Defn.Bytes) and the number of bytes read so far
	// (result.ByteCount).
	//
	// the panics cover situations that should never exists. if result
	// validation has been run then the panic situations will have been caught
	// then. if validation is not running then the code could theoretically
	// panic but that's okay, they should have been caught in testing.
	switch result.Defn.Bytes {
	case 3:
		switch result.ByteCount {
		case 3:
			d.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0xff00>>8, operand&0x00ff)
		case 2:
			d.Bytecode = fmt.Sprintf("%02x %02x ??", result.Defn.OpCode, operand&0xff00>>8)
		case 1:
			d.Bytecode = fmt.Sprintf("%02x ?? ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 2:
		switch result.ByteCount {
		case 2:
			d.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
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

	// ... and use assembler symbol for the operand if available/appropriate
	if result.InstructionData != nil && (d.Operand == "" || d.Operand[0] != '?') {
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
					if v, ok := symtable.Locations.Symbols[pc.Address()]; ok {
						d.Operand = v
					}

				} else {
					if v, ok := symtable.Locations.Symbols[operand]; ok {
						d.Operand = v
					}
				}
			case instructions.Read:
				if v, ok := symtable.Read.Symbols[operand]; ok {
					d.Operand = v
				}
			case instructions.Write:
				fallthrough
			case instructions.RMW:
				if v, ok := symtable.Write.Symbols[operand]; ok {
					d.Operand = v
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
