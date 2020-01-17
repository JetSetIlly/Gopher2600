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

package display

import (
	"fmt"
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/cpu/instructions"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/symbols"
	"strings"
)

// Instruction is the fully annotated, columnular representation of
// a instance of execution.Instruction
type Instruction struct {
	Location string
	Bytecode string
	Address  string
	Mnemonic string
	Operand  string
	Cycles   string
	Notes    string
}

// Format execution.Result and create a new instance of Instruction
func Format(result execution.Result, symtable *symbols.Table) (*Instruction, error) {
	if symtable == nil {
		symtable = &symbols.Table{}
	}

	d := &Instruction{}

	// if the operator hasn't been decoded yet then use placeholder strings in
	// key columns
	if result.Defn == nil {
		d.Mnemonic = "???"
		d.Operand = "?"
		return d, nil
	}

	d.Address = fmt.Sprintf("0x%04x", result.Address)

	if v, ok := symtable.Locations.Symbols[result.Address]; ok {
		d.Location = v
	}

	d.Mnemonic = result.Defn.Mnemonic

	// operands
	var operand uint16

	switch result.InstructionData.(type) {
	case uint8:
		operand = uint16(result.InstructionData.(uint8))
		d.Operand = fmt.Sprintf("$%02x", operand)
	case uint16:
		operand = uint16(result.InstructionData.(uint16))
		d.Operand = fmt.Sprintf("$%04x", operand)
	case nil:
		if result.Defn.Bytes == 2 {
			d.Operand = "??"
		} else if result.Defn.Bytes == 3 {
			d.Operand = "????"
		}
	}

	// Bytecode
	if result.Final {
		switch result.Defn.Bytes {
		case 3:
			d.Bytecode = fmt.Sprintf("%02x", operand&0xff00>>8)
			fallthrough
		case 2:
			d.Bytecode = fmt.Sprintf("%02x %s", operand&0x00ff, d.Bytecode)
			fallthrough
		case 1:
			d.Bytecode = fmt.Sprintf("%02x %s", result.Defn.OpCode, d.Bytecode)
		default:
			d.Bytecode = fmt.Sprintf("(%d bytes) %s", result.Defn.Bytes, d.Bytecode)
		}

		d.Bytecode = strings.TrimSpace(d.Bytecode)
	}

	// ... and use assembler symbol for the operand if available/appropriate
	if result.InstructionData != nil && (d.Operand == "" || d.Operand[0] != '?') {
		if result.Defn.AddressingMode != instructions.Immediate {

			switch result.Defn.Effect {
			case instructions.Flow:
				if result.Defn.AddressingMode == instructions.Relative {
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

	// cycles
	d.Cycles = fmt.Sprintf("%d", result.ActualCycles)

	// notes
	if result.PageFault {
		d.Notes = fmt.Sprintf("%s page-fault", d.Notes)
	}
	if result.CPUBug != "" {
		d.Notes = fmt.Sprintf("%s * %s *", d.Notes, result.CPUBug)
	}

	return d, nil
}
