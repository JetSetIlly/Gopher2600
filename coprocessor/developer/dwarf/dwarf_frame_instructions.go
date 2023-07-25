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

package dwarf

import (
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf/leb128"
)

type frameTableRow struct {
	// location (program counter) to which the current table state corresponds
	location uint32

	cfaRegister int
	cfaOffset   uint32

	// the frame table can also contain the unwinding rules for the machine
	// registers. we're not currently worried about these locations
}

type frameTable struct {
	rows [2]frameTableRow
}

func (tab *frameTable) newRow() {
	tab.rows[1] = tab.rows[0]
}

func (tab frameTable) String() string {
	return fmt.Sprintf("loc: %08x; reg: %d; offset: %08x", tab.rows[0].location, tab.rows[0].cfaRegister, tab.rows[0].cfaOffset)
}

type frameInstruction struct {
	length int
	opcode string
}

func (ins frameInstruction) String() string {
	return fmt.Sprintf("%s [%d]", ins.opcode, ins.length)
}

var frameInstructionNotImplemented = errors.New("not implemented")

// returns number of bytes used in instructions array
func decodeFrameInstruction(cie *frameSectionCIE, instructions []byte, tab *frameTable) (frameInstruction, error) {
	// opcode descriptions taken from "6.4.2 Call Frame Instructions" of
	// the "DWARF-4 Standard". Page numbers specified in the comment for
	// each opcode
	//
	// opcode/operand values taken from "7.23 Call Frame Information" of
	// the "DWARF-4 Standard"

	opcode := (instructions[0] & 0xc0) >> 6
	extendedOpcode := instructions[0] & 0x3f

	switch opcode {
	case 0x00:
		switch extendedOpcode {
		case 0x00:
			// DW_CFA_nop
			// (padding instruction)
			// "The DW_CFA_nop instruction has no operands and no required actions. It is
			// used as padding to make a CIE or FDE an appropriate size", page 136
			return frameInstruction{length: 1, opcode: "DW_CFA_nop"}, nil

		case 0x01:
			// DW_CFA_set_loc
			// (row creation instructions)
			// "The DW_CFA_set_loc instruction takes a single operand that represents a target address. The
			// required action is to create a new table row using the specified address as the location. All
			// other values in the new row are initially identical to the current row. The new location value
			// is always greater than the current one. If the segment_size field of this FDE's CIE is non-
			// zero, the initial location is preceded by a segment selector of the given length"
			tab.newRow()
			address := uint32(instructions[1]) | uint32(instructions[2])<<8 | uint32(instructions[3])<<16 | uint32(instructions[4])<<24
			tab.rows[0].location = address
			return frameInstruction{length: 5, opcode: "DW_CFA_set_loc"}, nil

		case 0x02:
			// DW_CFA_advance_loc1
			// (row creation instructions)
			// "The DW_CFA_advance_loc1 instruction takes a single ubyte operand that represents a
			// constant delta. This instruction is identical to DW_CFA_advance_loc except for the encoding
			// and size of the delta operand", page 132
			tab.newRow()
			tab.rows[0].location += uint32(instructions[1])
			return frameInstruction{length: 2, opcode: "DW_CFA_advance_loc1"}, nil

		case 0x03:
			// DW_CFA_advance_loc2
			// (row creation instructions)
			// "The DW_CFA_advance_loc2 instruction takes a single uhalf operand that represents a
			// constant delta. This instruction is identical to DW_CFA_advance_loc except for the encoding
			// and size of the delta operand", page 132
			tab.newRow()
			offset := uint32(instructions[1]) | uint32(instructions[2])<<8
			tab.rows[0].location += offset
			return frameInstruction{length: 3, opcode: "DW_CFA_advance_loc2"}, nil

		case 0x04:
			// DW_CFA_advance_loc4
			// (row creation instructions)
			// "The DW_CFA_advance_loc4 instruction takes a single uword operand that represents a
			// constant delta. This instruction is identical to DW_CFA_advance_loc except for the encoding
			// and size of the delta operand", page 132
			tab.newRow()
			offset := uint32(instructions[1]) | uint32(instructions[2])<<8 | uint32(instructions[3])<<16 | uint32(instructions[4])<<24
			tab.rows[0].location += offset
			return frameInstruction{length: 5, opcode: "DW_CFA_advance_loc4"}, nil

		case 0x05:
			// DW_CFA_offset_extended
			// (register rule instructions)
			// "The DW_CFA_offset_extended instruction takes two unsigned LEB128 operands
			// representing a register number and a factored offset. This instruction is identical to
			// DW_CFA_offset except for the encoding and size of the register operand", page 134

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_offset_extended"}, frameInstructionNotImplemented

		case 0x06:
			// DW_CFA_restore_extended
			// (register rule instructions)
			// "The DW_CFA_restore_extended instruction takes a single unsigned LEB128 operand that
			// represents a register number. This instruction is identical to DW_CFA_restore except for the
			// encoding and size of the register operand", page 136

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_restore_extended"}, frameInstructionNotImplemented

		case 0x07:
			// DW_CFA_undefined
			// (register rule instructions)
			// "The DW_CFA_undefined instruction takes a single unsigned LEB128 operand that
			// represents a register number. The required action is to set the rule for the
			// specified register to 'undefined'", page 134

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_undefined"}, frameInstructionNotImplemented

		case 0x08:
			// DW_CFA_same_value
			// (register rule instructions)
			// "The DW_CFA_same_value instruction takes a single unsigned LEB128 operand that
			// represents a register number. The required action is to set the rule for the specified register to
			// 'same value'", page 134

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_same_value"}, frameInstructionNotImplemented

		case 0x09:
			// DW_CFA_register
			// (register rule instructions)
			// "The DW_CFA_register instruction takes two unsigned LEB128 operands representing
			// register numbers. The required action is to set the rule for the first register to be register(R)
			// where R is the second register", page 135

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_register"}, frameInstructionNotImplemented

		case 0x0a:
			// DW_CFA_remember_state
			// (row state instructions)
			// "The DW_CFA_remember_state instruction takes no operands. The required action is to push
			// the set of rules for every register onto an implicit stack", page 136

			// unimplemented
			return frameInstruction{length: 1, opcode: "DW_CFA_remember_state"}, frameInstructionNotImplemented

		case 0x0b:
			// DW_CFA_restore_state
			// (row state instructions)
			// "The DW_CFA_restore_state instruction takes no operands. The required action is to pop the
			// set of rules off the implicit stack and place them in the current row", page 136

			// unimplemented
			return frameInstruction{length: 1, opcode: "DW_CFA_restore_state"}, frameInstructionNotImplemented

		case 0x0c:
			// DW_CFA_def_cfa
			// (CFA Definition Instructions)
			// "The DW_CFA_def_cfa instruction takes two unsigned LEB128 operands representing
			// a register number and a (non-factored) offset. The required action is to define
			// the current CFA rule to use the provided register and offset", page 132
			n := 1
			reg, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			offset, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			tab.rows[0].cfaRegister = int(reg)
			tab.rows[0].cfaOffset = uint32(offset)
			return frameInstruction{length: n, opcode: "DW_CFA_def_cfa"}, nil

		case 0x0d:
			// DW_CFA_def_cfa_register
			// (CFA definition instructions)
			// "The DW_CFA_def_cfa_register instruction takes a single unsigned LEB128 operand
			// representing a register number. The required action is to define the current CFA rule to use
			// the provided register (but to keep the old offset). This operation is valid only if the current
			// CFA rule is defined to use a register and offset"
			n := 1
			reg, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			tab.rows[0].cfaRegister = int(reg)
			return frameInstruction{length: n, opcode: "DW_CFA_def_cfa_register"}, nil

		case 0x0e:
			// DW_CFA_def_cfa_offset
			// (CFA definition instructions)
			// "The DW_CFA_def_cfa_offset instruction takes a single unsigned LEB128 operand
			// representing a (non-factored) offset. The required action is to define the current CFA rule to
			// use the provided offset (but to keep the old register). This operation is valid only if the
			// current CFA rule is defined to use a register and offset", page 133
			n := 1
			offset, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			tab.rows[0].cfaOffset = uint32(offset)
			return frameInstruction{length: n, opcode: "DW_CFA_def_cfa_offset"}, nil

		case 0x0f:
			// DW_CFA_def_cfa_expression
			// (CFA definition instructions)
			// "The DW_CFA_def_cfa_expression instruction takes a single operand encoded as a
			// DW_FORM_exprloc value representing a DWARF expression. The required action is
			// to establish that expression as the means by which the current CFA is computed",
			// page 133
			return frameInstruction{length: 0, opcode: "DW_CFA_def_cfa_expression"}, frameInstructionNotImplemented

		case 0x10:
			// DW_CFA_expression
			// (register rule instructions)
			// "The DW_CFA_expression instruction takes two operands: an unsigned LEB128 value
			// representing a register number, and a DW_FORM_block value representing a DWARF
			// expression. The required action is to change the rule for the register indicated by the register
			// number to be an expression(E) rule where E is the DWARF expression. That is, the DWARF
			// expression computes the address. The value of the CFA is pushed on the DWARF evaluation
			// stack prior to execution of the DWARF expression", page 135
			return frameInstruction{length: 0, opcode: "DW_CFA_expression"}, frameInstructionNotImplemented

		case 0x11:
			// DW_CFA_offset_extended_sf
			// (register rule instructions)
			// "The DW_CFA_offset_extended_sf instruction takes two operands: an unsigned LEB128
			// value representing a register number and a signed LEB128 factored offset. This instruction is
			// identical to DW_CFA_offset_extended except that the second operand is signed and
			// factored. The resulting offset is factored_offset * data_alignment_factor", page 134

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeSLEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_offset_extended_sf"}, frameInstructionNotImplemented

		case 0x12:
			// DW_CFA_def_cfa_sf
			// (CFA definition instructions)
			// "The DW_CFA_def_cfa_sf instruction takes two operands: an unsigned LEB128 value
			// representing a register number and a signed LEB128 factored offset. This instruction is
			// identical to DW_CFA_def_cfa except that the second operand is signed and factored. The
			// resulting offset is factored_offset * data_alignment_factor", page 133

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeSLEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_def_cfa_sf"}, frameInstructionNotImplemented

		case 0x13:
			// DW_CFA_def_cfa_offset_sf
			// (CFA definition instructions)
			// "The DW_CFA_def_cfa_offset_sf instruction takes a signed LEB128 operand representing a
			// factored offset. This instruction is identical to DW_CFA_def_cfa_offset except that the
			// operand is signed and factored. The resulting offset is factored_offset *
			// data_alignment_factor. This operation is valid only if the current CFA rule is defined to
			// use a register and offset", page 133

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeSLEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_def_cfa_offset_sf"}, frameInstructionNotImplemented

		case 0x14:
			// DW_CFA_val_offset
			// (register rule instructions)
			// "The DW_CFA_val_offset instruction takes two unsigned LEB128 operands representing a
			// register number and a factored offset. The required action is to change the rule for the
			// register indicated by the register number to be a val_offset(N) rule where the value of N is
			// factored_offset * data_alignment_factor", page 134

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeULEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_val_offset"}, frameInstructionNotImplemented

		case 0x15:
			// DW_CFA_val_offset_sf
			// (register rule instructions)
			// "The DW_CFA_val_offset_sf instruction takes two operands: an unsigned LEB128 value
			// representing a register number and a signed LEB128 factored offset. This instruction is
			// identical to DW_CFA_val_offset except that the second operand is signed and factored. The
			// resulting offset is factored_offset * data_alignment_factor", page 135

			// unimplemented but we need to know how many bytes to consume
			n := 1
			_, l := leb128.DecodeULEB128(instructions[n:])
			n += l
			_, l = leb128.DecodeSLEB128(instructions[n:])
			n += l
			return frameInstruction{length: n, opcode: "DW_CFA_val_offset_sf"}, frameInstructionNotImplemented

		case 0x16:
			// DW_CFA_val_expression
			// (register rule instructions)
			// "The DW_CFA_val_expression instruction takes two operands: an unsigned LEB128 value
			// representing a register number, and a DW_FORM_block value representing a DWARF
			// expression. The required action is to change the rule for the register indicated by the register
			// number to be a val_expression(E) rule where E is the DWARF expression. That is, the
			// DWARF expression computes the value of the given register. The value of the CFA is
			// pushed on the DWARF evaluation stack prior to execution of the DWARF expression"
			return frameInstruction{length: 0, opcode: "DW_CFA_val_expression"}, fmt.Errorf("DW_CFA_val_expression not implemented")

		case 0x1c:
			// DW_CFA_lo_user
			return frameInstruction{length: 1, opcode: "DW_CFA_lo_user"}, frameInstructionNotImplemented

		case 0x3f:
			// DW_CFA_hi_user
			return frameInstruction{length: 1, opcode: "DW_CFA_hi_user"}, frameInstructionNotImplemented
		}

	case 0x01:
		// DW_CFA_advance_loc
		// (row creation instructions)
		// "The DW_CFA_advance instruction takes a single operand (encoded with the opcode) that
		// represents a constant delta. The required action is to create a new table row with a location
		// value that is computed by taking the current entry’s location value and adding the value of
		// delta * code_alignment_factor. All other values in the new row are initially identical
		// to the current row", page 132
		tab.rows[0].location += uint32(extendedOpcode) * uint32(cie.codeAlignment)
		return frameInstruction{length: 1, opcode: "DW_CFA_advance_loc"}, nil

	case 0x02:
		// DW_CFA_offset
		// (register rule instructions)
		// "The DW_CFA_offset instruction takes two operands: a register number (encoded with the
		// opcode) and an unsigned LEB128 constant representing a factored offset. The required action
		// is to change the rule for the register indicated by the register number to be an offset(N) rule
		// where the value of N is factored offset * data_alignment_factor", page 134

		// unimplemented but we need to know how many bytes to consume
		n := 1
		_, l := leb128.DecodeULEB128(instructions[n:])
		n += l
		return frameInstruction{length: n, opcode: "DW_CFA_offset"}, frameInstructionNotImplemented

	case 0x03:
		// DW_CFA_restore
		// (register rule instructions)
		// "The DW_CFA_restore instruction takes a single operand (encoded with the opcode) that
		// represents a register number. The required action is to change the rule for the indicated
		// register to the rule assigned it by the initial_instructions in the CIE"

		// unimplemented
		return frameInstruction{length: 1, opcode: "DW_CFA_restore"}, frameInstructionNotImplemented
	}

	return frameInstruction{}, fmt.Errorf("unknown call frame instruction: %02x", opcode)
}
