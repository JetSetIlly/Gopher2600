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

package arm

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
	"github.com/jetsetilly/gopher2600/logger"
)

// returns an instance of the decodeFunction type. if the value is nil then that
// means the decoding could not complete
func (arm *ARM) decodeThumb(opcode uint16) decodeFunction {
	// working backwards up the table in Figure 5-1 of the ARM7TDMI Data Sheet.
	if opcode&0xf000 == 0xf000 {
		// format 19 - Long branch with link
		return arm.decodeThumbLongBranchWithLink(opcode)
	} else if opcode&0xf000 == 0xe000 {
		// format 18 - Unconditional branch
		return arm.decodeThumbUnconditionalBranch(opcode)
	} else if opcode&0xff00 == 0xdf00 {
		// format 17 - Software interrupt"
		return arm.decodeThumbSoftwareInterrupt(opcode)
	} else if opcode&0xf000 == 0xd000 {
		// format 16 - Conditional branch
		return arm.decodeThumbConditionalBranch(opcode)
	} else if opcode&0xf000 == 0xc000 {
		// format 15 - Multiple load/store
		return arm.decodeThumbMultipleLoadStore(opcode)
	} else if opcode&0xf600 == 0xb400 {
		// format 14 - Push/pop registers
		return arm.decodeThumbPushPopRegisters(opcode)
	} else if opcode&0xff00 == 0xb000 {
		// format 13 - Add offset to stack pointer
		return arm.decodeThumbAddOffsetToSP(opcode)
	} else if opcode&0xf000 == 0xa000 {
		// format 12 - Load address
		return arm.decodeThumbLoadAddress(opcode)
	} else if opcode&0xf000 == 0x9000 {
		// format 11 - SP-relative load/store
		return arm.decodeThumbSPRelativeLoadStore(opcode)
	} else if opcode&0xf000 == 0x8000 {
		// format 10 - Load/store halfword
		return arm.decodeThumbLoadStoreHalfword(opcode)
	} else if opcode&0xe000 == 0x6000 {
		// format 9 - Load/store with immediate offset
		return arm.decodeThumbLoadStoreWithImmOffset(opcode)
	} else if opcode&0xf200 == 0x5200 {
		// format 8 - Load/store sign-extended byte/halfword
		return arm.decodeThumbLoadStoreSignExtendedByteHalford(opcode)
	} else if opcode&0xf200 == 0x5000 {
		// format 7 - Load/store with register offset
		return arm.decodeThumbLoadStoreWithRegisterOffset(opcode)
	} else if opcode&0xf800 == 0x4800 {
		// format 6 - PC-relative load
		return arm.decodeThumbPCrelativeLoad(opcode)
	} else if opcode&0xfc00 == 0x4400 {
		// format 5 - Hi register operations/branch exchange
		return arm.decodeThumbHiRegisterOps(opcode)
	} else if opcode&0xfc00 == 0x4000 {
		// format 4 - ALU operations
		return arm.decodeThumbALUoperations(opcode)
	} else if opcode&0xe000 == 0x2000 {
		// format 3 - Move/compare/add/subtract immediate
		return arm.decodeThumbMovCmpAddSubImm(opcode)
	} else if opcode&0xf800 == 0x1800 {
		// format 2 - Add/subtract
		return arm.decodeThumbAddSubtract(opcode)
	} else if opcode&0xe000 == 0x0000 {
		// format 1 - Move shifted register
		return arm.decodeThumbMoveShiftedRegister(opcode)
	}

	return nil
}

// TODO: the size of the returned decodeFunction() can definitely be
// reduced/improved in the case of all Thumb instruction formats. the work has
// not yet be done because (a) wouldn't gain much performance, and (b) there is
// a danger that damage will be done to the cycle counting logic. this latter
// point means that a proper testing strategy should be developed before
// proceeding

func (arm *ARM) decodeThumbMoveShiftedRegister(opcode uint16) decodeFunction {
	// format 1 - Move shifted register
	op := (opcode & 0x1800) >> 11
	shift := (opcode & 0x7c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	return func() *DisasmEntry {
		// in this class of operation the srcVal register may also be the dest
		// register so we need to make a note of the value before it is
		// overwrittten
		srcVal := arm.state.registers[srcReg]

		switch op {
		case 0b00:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LSL",
					Operand:  fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift),
				}
			}

			// if immed_5 == 0
			//	C Flag = unaffected
			//	Rd = Rm
			// else /* immed_5 > 0 */
			//	C Flag = Rm[32 - immed_5]
			//	Rd = Rm Logical_Shift_Left immed_5

			if shift == 0 {
				arm.state.registers[destReg] = srcVal
			} else {
				m := uint32(0x01) << (32 - shift)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(srcVal&m == m)
				}
				arm.state.registers[destReg] = arm.state.registers[srcReg] << shift
			}
		case 0b01:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LSR",
					Operand:  fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift),
				}
			}

			// if immed_5 == 0
			//		C Flag = Rm[31]
			//		Rd = 0
			// else /* immed_5 > 0 */
			//		C Flag = Rm[immed_5 - 1]
			//		Rd = Rm Logical_Shift_Right immed_5

			if shift == 0 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(srcVal&0x80000000 == 0x80000000)
				}
				arm.state.registers[destReg] = 0x00
			} else {
				m := uint32(0x01) << (shift - 1)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(srcVal&m == m)
				}
				arm.state.registers[destReg] = srcVal >> shift
			}
		case 0b10:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ASR",
					Operand:  fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift),
				}
			}

			// if immed_5 == 0
			//		C Flag = Rm[31]
			//		if Rm[31] == 0 then
			//				Rd = 0
			//		else /* Rm[31] == 1 */]
			//				Rd = 0xFFFFFFFF
			// else /* immed_5 > 0 */
			//		C Flag = Rm[immed_5 - 1]
			//		Rd = Rm Arithmetic_Shift_Right immed_5

			if shift == 0 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(srcVal&0x80000000 == 0x80000000)
				}
				if arm.state.status.carry {
					arm.state.registers[destReg] = 0xffffffff
				} else {
					arm.state.registers[destReg] = 0x00000000
				}
			} else { // shift > 0
				m := uint32(0x01) << (shift - 1)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(srcVal&m == m)
				}
				a := srcVal >> shift
				if srcVal&0x80000000 == 0x80000000 {
					a |= (0xffffffff << (32 - shift))
				}
				arm.state.registers[destReg] = a
			}

		case 0b11:
			panic(fmt.Sprintf("illegal (move shifted register) thumb operation (%04b)", op))
		}

		if arm.state.status.itMask == 0b0000 {
			arm.state.status.isZero(arm.state.registers[destReg])
			arm.state.status.isNegative(arm.state.registers[destReg])
		}

		if destReg == rPC {
			logger.Log("ARM7", "shift and store in PC is not possible in thumb mode")
		}

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		if shift > 0 {
			arm.Icycle()
		}

		return nil
	}
}

func (arm *ARM) decodeThumbAddSubtract(opcode uint16) decodeFunction {
	// format 2 - Add/subtract
	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200
	imm := uint32((opcode & 0x01c0) >> 6)
	srcReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	return func() *DisasmEntry {
		// value to work with is either an immediate value or is in a register
		val := imm
		if !immediate && arm != nil {
			val = arm.state.registers[imm]
		}

		if subtract {
			if arm.decodeOnly {
				if immediate {
					return &DisasmEntry{
						Operator: "SUB",
						Operand:  fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, imm),
					}
				}
				return &DisasmEntry{
					Operator: "SUB",
					Operand:  fmt.Sprintf("R%d, R%d, R%d ", destReg, srcReg, imm),
				}
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isCarry(arm.state.registers[srcReg], ^val, 1)
				arm.state.status.isOverflow(arm.state.registers[srcReg], ^val, 1)
			}
			arm.state.registers[destReg] = arm.state.registers[srcReg] - val
		} else {
			if arm.decodeOnly {
				if immediate {
					return &DisasmEntry{
						Operator: "ADD",
						Operand:  fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, imm),
					}
				}
				return &DisasmEntry{
					Operator: "ADD",
					Operand:  fmt.Sprintf("R%d, R%d, R%d ", destReg, srcReg, imm),
				}
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isCarry(arm.state.registers[srcReg], val, 0)
				arm.state.status.isOverflow(arm.state.registers[srcReg], val, 0)
			}
			arm.state.registers[destReg] = arm.state.registers[srcReg] + val
		}

		if arm.state.status.itMask == 0b0000 {
			arm.state.status.isZero(arm.state.registers[destReg])
			arm.state.status.isNegative(arm.state.registers[destReg])
		}

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return nil
	}
}

// "The instructions in this group perform operations between a Lo register and
// an 8-bit immediate value".
func (arm *ARM) decodeThumbMovCmpAddSubImm(opcode uint16) decodeFunction {
	// format 3 - Move/compare/add/subtract immediate
	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode & 0x00ff)

	return func() *DisasmEntry {
		switch op {
		case 0b00:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "MOV",
					Operand:  fmt.Sprintf("R%d, #$%02x ", destReg, imm),
				}
			}

			arm.state.registers[destReg] = imm
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b01:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "CMP",
					Operand:  fmt.Sprintf("R%d, #$%02x ", destReg, imm),
				}
			}

			// status will be set when in IT block
			arm.state.status.isCarry(arm.state.registers[destReg], ^imm, 1)
			arm.state.status.isOverflow(arm.state.registers[destReg], ^imm, 1)
			cmp := arm.state.registers[destReg] - imm
			arm.state.status.isNegative(cmp)
			arm.state.status.isZero(cmp)
		case 0b10:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ADD",
					Operand:  fmt.Sprintf("R%d, #$%02x ", destReg, imm),
				}
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isCarry(arm.state.registers[destReg], imm, 0)
				arm.state.status.isOverflow(arm.state.registers[destReg], imm, 0)
			}
			arm.state.registers[destReg] += imm
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b11:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "SUB",
					Operand:  fmt.Sprintf("R%d, #$%02x ", destReg, imm),
				}
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isCarry(arm.state.registers[destReg], ^imm, 1)
				arm.state.status.isOverflow(arm.state.registers[destReg], ^imm, 1)
			}
			arm.state.registers[destReg] -= imm
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		}

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return nil
	}
}

// "The following instructions perform ALU operations on a Lo register pair".
func (arm *ARM) decodeThumbALUoperations(opcode uint16) decodeFunction {
	// format 4 - ALU operations
	op := (opcode & 0x03c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	return func() *DisasmEntry {
		var shift uint32
		var mul bool
		var mulOperand uint32

		switch op {
		case 0b0000:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "AND",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] &= arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0001:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "EOR",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] ^= arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0010:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LSL",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			shift = arm.state.registers[srcReg]

			// if Rs[7:0] == 0
			//		C Flag = unaffected
			//		Rd = unaffected
			// else if Rs[7:0] < 32 then
			//		C Flag = Rd[32 - Rs[7:0]]
			//		Rd = Rd Logical_Shift_Left Rs[7:0]
			// else if Rs[7:0] == 32 then
			//		C Flag = Rd[0]
			//		Rd = 0
			// else /* Rs[7:0] > 32 */
			//		C Flag = 0
			//		Rd = 0
			// N Flag = Rd[31]
			// Z Flag = if Rd == 0 then 1 else 0
			// V Flag = unaffected

			if shift > 0 && shift < 32 {
				m := uint32(0x01) << (32 - shift)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&m == m)
				}
				arm.state.registers[destReg] <<= shift
			} else if shift == 32 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&0x01 == 0x01)
				}
				arm.state.registers[destReg] = 0x00
			} else if shift > 32 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(false)
				}
				arm.state.registers[destReg] = 0x00
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0011:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LSR",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			shift = arm.state.registers[srcReg]

			// if Rs[7:0] == 0 then
			//		C Flag = unaffected
			//		Rd = unaffected
			// else if Rs[7:0] < 32 then
			//		C Flag = Rd[Rs[7:0] - 1]
			//		Rd = Rd Logical_Shift_Right Rs[7:0]
			// else if Rs[7:0] == 32 then
			//		C Flag = Rd[31]
			//		Rd = 0
			// else /* Rs[7:0] > 32 */
			//		C Flag = 0
			//		Rd = 0
			// N Flag = Rd[31]
			// Z Flag = if Rd == 0 then 1 else 0
			// V Flag = unaffected

			if shift > 0 && shift < 32 {
				m := uint32(0x01) << (shift - 1)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&m == m)
				}
				arm.state.registers[destReg] >>= shift
			} else if shift == 32 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&0x80000000 == 0x80000000)
				}
				arm.state.registers[destReg] = 0x00
			} else if shift > 32 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(false)
				}
				arm.state.registers[destReg] = 0x00
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0100:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ASR",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			shift = arm.state.registers[srcReg]

			// if Rs[7:0] == 0 then
			//		C Flag = unaffected
			//		Rd = unaffected
			// else if Rs[7:0] < 32 then
			//		C Flag = Rd[Rs[7:0] - 1]
			//		Rd = Rd Arithmetic_Shift_Right Rs[7:0]
			// else /* Rs[7:0] >= 32 */
			//		C Flag = Rd[31]
			//		if Rd[31] == 0 then
			//			Rd = 0
			//		else /* Rd[31] == 1 */
			//			Rd = 0xFFFFFFFF
			// N Flag = Rd[31]
			// Z Flag = if Rd == 0 then 1 else 0
			// V Flag = unaffected
			if shift > 0 && shift < 32 {
				src := arm.state.registers[destReg]
				m := uint32(0x01) << (shift - 1)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(src&m == m)
				}
				a := src >> shift
				if src&0x80000000 == 0x80000000 {
					a |= (0xffffffff << (32 - shift))
				}
				arm.state.registers[destReg] = a
			} else if shift >= 32 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&0x80000000 == 0x80000000)
				}
				if !arm.state.status.carry {
					arm.state.registers[destReg] = 0x00
				} else {
					arm.state.registers[destReg] = 0xffffffff
				}
			}
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0101:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ADC",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			if arm.state.status.carry {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.isCarry(arm.state.registers[destReg], arm.state.registers[srcReg], 1)
					arm.state.status.isOverflow(arm.state.registers[destReg], arm.state.registers[srcReg], 1)
				}
				arm.state.registers[destReg] += arm.state.registers[srcReg]
				arm.state.registers[destReg]++
			} else {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.isCarry(arm.state.registers[destReg], arm.state.registers[srcReg], 0)
					arm.state.status.isOverflow(arm.state.registers[destReg], arm.state.registers[srcReg], 0)
				}
				arm.state.registers[destReg] += arm.state.registers[srcReg]
			}
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0110:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "SBC",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			if !arm.state.status.carry {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.isCarry(arm.state.registers[destReg], ^arm.state.registers[srcReg], 0)
					arm.state.status.isOverflow(arm.state.registers[destReg], ^arm.state.registers[srcReg], 0)
				}
				arm.state.registers[destReg] -= arm.state.registers[srcReg]
				arm.state.registers[destReg]--
			} else {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.isCarry(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
					arm.state.status.isOverflow(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
				}
				arm.state.registers[destReg] -= arm.state.registers[srcReg]
			}
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b0111:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ROR",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			shift = arm.state.registers[srcReg]

			// if Rs[7:0] == 0 then
			//		C Flag = unaffected
			//		Rd = unaffected
			// else if Rs[4:0] == 0 then
			//		C Flag = Rd[31]
			//		Rd = unaffected
			// else /* Rs[4:0] > 0 */
			//		C Flag = Rd[Rs[4:0] - 1]
			//		Rd = Rd Rotate_Right Rs[4:0]
			// N Flag = Rd[31]
			// Z Flag = if Rd == 0 then 1 else 0
			// V Flag = unaffected
			if shift&0xff == 0 {
				// unaffected
			} else if shift&0x1f == 0 {
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&0x80000000 == 0x80000000)
				}
			} else {
				m := uint32(0x01) << (shift - 1)
				if arm.state.status.itMask == 0b0000 {
					arm.state.status.setCarry(arm.state.registers[destReg]&m == m)
				}
				arm.state.registers[destReg] = bits.RotateLeft32(arm.state.registers[destReg], -int(shift))
			}
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b1000:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "TST",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			w := arm.state.registers[destReg] & arm.state.registers[srcReg]
			// status will be set when in IT block
			arm.state.status.isZero(w)
			arm.state.status.isNegative(w)
		case 0b1001:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "NEG",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isCarry(0, ^arm.state.registers[srcReg], 1)
				arm.state.status.isOverflow(0, ^arm.state.registers[srcReg], 1)
			}
			arm.state.registers[destReg] = -arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b1010:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "CMP",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			// status will be set when in IT block
			arm.state.status.isCarry(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
			arm.state.status.isOverflow(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
			cmp := arm.state.registers[destReg] - arm.state.registers[srcReg]
			arm.state.status.isZero(cmp)
			arm.state.status.isNegative(cmp)
		case 0b1011:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "CMN",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			// status will be set when in IT block
			arm.state.status.isCarry(arm.state.registers[destReg], arm.state.registers[srcReg], 0)
			arm.state.status.isOverflow(arm.state.registers[destReg], arm.state.registers[srcReg], 0)
			cmp := arm.state.registers[destReg] + arm.state.registers[srcReg]
			arm.state.status.isZero(cmp)
			arm.state.status.isNegative(cmp)
		case 0b1100:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ORR",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] |= arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b1101:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "MUL",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			mul = true
			mulOperand = arm.state.registers[srcReg]
			arm.state.registers[destReg] *= arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b1110:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "BIC",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] &= ^arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		case 0b1111:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "MVN",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] = ^arm.state.registers[srcReg]
			if arm.state.status.itMask == 0b0000 {
				arm.state.status.isZero(arm.state.registers[destReg])
				arm.state.status.isNegative(arm.state.registers[destReg])
			}
		default:
			panic(fmt.Sprintf("unimplemented (ALU) thumb operation (%04b)", op))
		}

		// page 7-11 in "ARM7TDMI-S Technical Reference Manual r4p3"
		if shift > 0 && destReg == rPC {
			logger.Log("ARM7", "shift and store in PC is not possible in thumb mode")
		}

		if mul {
			// "7.7 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			//  and
			// "7.2 Instruction Cycle Count Summary"  in "ARM7TDMI-S Technical
			// Reference Manual r4p3" ...
			p := bits.OnesCount32(mulOperand & 0xffffff00)
			if p == 0 || p == 24 {
				// ... Is 1 if bits [32:8] of the multiplier operand are all zero or one.
				arm.Icycle()
			} else {
				p := bits.OnesCount32(mulOperand & 0xffff0000)
				if p == 0 || p == 16 {
					// ... Is 2 if bits [32:16] of the multiplier operand are all zero or one.
					arm.Icycle()
					arm.Icycle()
				} else {
					p := bits.OnesCount32(mulOperand & 0xff000000)
					if p == 0 || p == 8 {
						// ... Is 3 if bits [31:24] of the multiplier operand are all zero or one.
						arm.Icycle()
						arm.Icycle()
						arm.Icycle()
					} else {
						// ... Is 4 otherwise.
						arm.Icycle()
						arm.Icycle()
						arm.Icycle()
						arm.Icycle()
					}
				}
			}
		} else {
			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			if shift > 0 {
				arm.Icycle()
			}
		}

		return nil
	}
}

func (arm *ARM) decodeThumbHiRegisterOps(opcode uint16) decodeFunction {
	// format 5 - Hi register operations/branch exchange
	op := (opcode & 0x300) >> 8
	hi1 := opcode&0x80 == 0x80
	hi2 := opcode&0x40 == 0x40
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// labels used to decoraate operands indicating Hi/Lo register usage
	if hi1 {
		destReg += 8
	}
	if hi2 {
		srcReg += 8
	}

	return func() *DisasmEntry {
		// when disassembling format 5 instructions, some documentation suggests that
		// the registers are labelled Rn or Hn, depending on whether the register is
		// a "high" register or not. earlier versions of this implementation
		// followed that convention but for simplicity we now use the Rn form only

		switch op {
		case 0b00:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ADD",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			// not two's complement
			arm.state.registers[destReg] += arm.state.registers[srcReg]

			// "5.5.5 Using R15 as an operand If R15 is used as an operand, the
			//   value will be the address of the instruction + 4 with bit 0
			//   cleared. Executing a BX PC in THUMB state from a non-word aligned
			//   address will result in unpredictable execution"
			//
			//   "ARM7TDMI-S Technical Reference Manual r4p3"
			if destReg == rPC {
				// adding 2 to PC and not 4 because the PC has already been
				// advanced on from the "address of the instruction"
				arm.state.registers[destReg] += 2
				arm.state.registers[destReg] &= 0xfffffffe
			}

			// status register not changed

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary

			return nil
		case 0b01:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "CMP",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			// alu_out = Rn - Rm
			// N Flag = alu_out[31]
			// Z Flag = if alu_out == 0 then 1 else 0
			// C Flag = NOT BorrowFrom(Rn - Rm)
			// V Flag = OverflowFrom(Rn - Rm)

			// status will be set when in IT block
			arm.state.status.isCarry(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
			arm.state.status.isOverflow(arm.state.registers[destReg], ^arm.state.registers[srcReg], 1)
			cmp := arm.state.registers[destReg] - arm.state.registers[srcReg]
			arm.state.status.isZero(cmp)
			arm.state.status.isNegative(cmp)

			// it's not clear to whether section 5.5 of the "ARM7TDMI-S Technical
			// Reference Manual r4p3" applies to the CMP instruction

			return nil
		case 0b10:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "MOV",
					Operand:  fmt.Sprintf("R%d, R%d ", destReg, srcReg),
				}
			}

			arm.state.registers[destReg] = arm.state.registers[srcReg]

			// "5.5.5 Using R15 as an operand If R15 is used as an operand, the
			//   value will be the address of the instruction + 4 with bit 0
			//   cleared. Executing a BX PC in THUMB state from a non-word aligned
			//   address will result in unpredictable execution"
			//
			//   "ARM7TDMI-S Technical Reference Manual r4p3"
			if destReg == rPC {
				// adding 2 to PC and not 4 because the PC has already been
				// advanced on from the "address of the instruction"
				arm.state.registers[destReg] += 2
				arm.state.registers[destReg] &= 0xfffffffe
			}

			// status register not changed

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary

			return nil
		case 0b11:
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "BX",
					Operand:  fmt.Sprintf("R%d ", srcReg),
				}
			}

			switch arm.mmap.ARMArchitecture {
			case architecture.ARMv7_M:
				if opcode&0x0080 == 0x0080 {
					// "A7.7.19 BLX (register)" in "ARMv7-M"
					target := arm.state.registers[srcReg]
					nextPC := arm.state.registers[rPC] - 2
					arm.state.registers[rLR] = nextPC | 0x01
					if target&0x01 == 0x00 {
						// cannot switch to ARM mode in the ARMv7-M architecture
						arm.state.yield.Type = coprocessor.YieldUndefinedBehaviour
						arm.state.yield.Error = errors.New("cannot switch to ARM mode in ARMv7-M architecture")
					}
					arm.state.registers[rPC] = (target + 2) & 0xfffffffe
				} else {
					// "A7.7.20 BX " in "ARMv7-M"
					target := arm.state.registers[srcReg]
					if target&0x01 == 0x00 {
						// cannot switch to ARM mode in the ARMv7-M architecture
						arm.state.yield.Type = coprocessor.YieldUndefinedBehaviour
						arm.state.yield.Error = errors.New("cannot switch to ARM mode in ARMv7-M architecture")
					}
					arm.state.registers[rPC] = (target + 2) & 0xfffffffe
				}

				// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - fillPipeline() will be called if necessary
				return nil
			case architecture.ARM7TDMI:
				thumbMode := arm.state.registers[srcReg]&0x01 == 0x01

				var newPC uint32

				// "ARM7TDMI Data Sheet" page 5-15:
				//
				// "If R15 is used as an operand, the value will be the address of the instruction + 4 with
				// bit 0 cleared. Executing a BX PC in THUMB state from a non-word aligned address
				// will result in unpredictable execution."
				if srcReg == rPC {
					newPC = arm.state.registers[rPC] + 2
				} else {
					newPC = (arm.state.registers[srcReg] & 0x7ffffffe) + 2
				}

				if thumbMode {
					arm.state.registers[rPC] = newPC

					// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
					// - fillPipeline() will be called if necessary
					return nil
				}

				// switch to ARM mode. emulate function call.
				res, err := arm.hook.ARMinterrupt(arm.state.registers[rPC]-4, arm.state.registers[2], arm.state.registers[3])
				if err != nil {
					arm.state.yield.Type = coprocessor.YieldExecutionError
					arm.state.yield.Error = err
					return nil
				}

				// if ARMinterrupt returns false this indicates that the
				// function at the quoted program counter is not recognised and
				// has nothing to do with the cartridge mapping. at this point
				// we can assume that the main() function call is done and we
				// can return to the VCS emulation.
				if !res.InterruptServiced {
					arm.state.yield.Type = coprocessor.YieldProgramEnded
					arm.state.yield.Error = nil
					// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
					//  - interrupted
					return nil
				}

				// ARM function updates the ARM registers
				if res.SaveResult {
					arm.state.registers[res.SaveRegister] = res.SaveValue
				}

				// the end of the emulated function will have an operation that
				// switches back to thumb mode, and copies the link register to the
				// program counter. we need to emulate that too.
				arm.state.registers[rPC] = arm.state.registers[rLR] + 2

				// add cycles used by the ARM program
				arm.armInterruptCycles(res)

				// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - fillPipeline() will be called if necessary
			}
		}

		return nil
	}
}

func (arm *ARM) decodeThumbPCrelativeLoad(opcode uint16) decodeFunction {
	// format 6 - PC-relative load
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	return func() *DisasmEntry {
		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "LDR",
				Operand:  fmt.Sprintf("R%d, [PC, #$%02x] ", destReg, imm),
			}
		}

		// "Bit 1 of the PC value is forced to zero for the purpose of this
		// calculation, so the address is always word-aligned."
		pc := AlignTo32bits(arm.state.registers[rPC])

		// immediate value is not two's complement (surprisingly)
		addr := pc + imm
		arm.state.registers[destReg] = arm.read32bit(addr, false)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return nil
	}
}

func (arm *ARM) decodeThumbLoadStoreWithRegisterOffset(opcode uint16) decodeFunction {
	// format 7 - Load/store with register offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	return func() *DisasmEntry {
		// the actual address we'll be loading from (or storing to)
		addr := arm.state.registers[baseReg] + arm.state.registers[offsetReg]

		if load {
			if byteTransfer {
				if arm.decodeOnly {
					return &DisasmEntry{
						Operator: "LDRB",
						Operand:  fmt.Sprintf("R%d, [R%d, R%d]", reg, baseReg, offsetReg),
					}
				}

				arm.state.registers[reg] = uint32(arm.read8bit(addr))

				// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - fillPipeline() will be called if necessary
				arm.Ncycle(dataRead, addr)
				arm.Icycle()

				return nil
			}
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDR",
					Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
				}
			}

			arm.state.registers[reg] = arm.read32bit(addr, false)

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		if byteTransfer {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "STRB",
					Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
				}
			}

			arm.write8bit(addr, uint8(arm.state.registers[reg]))

			// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			arm.storeRegisterCycles(addr)

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "STR",
				Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
			}
		}

		arm.write32bit(addr, arm.state.registers[reg], false)

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return nil
	}
}

func (arm *ARM) decodeThumbLoadStoreSignExtendedByteHalford(opcode uint16) decodeFunction {
	// format 8 - Load/store sign-extended byte/halfword
	hi := opcode&0x0800 == 0x800
	sign := opcode&0x0400 == 0x400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	return func() *DisasmEntry {
		// the actual address we'll be loading from (or storing to)
		addr := arm.state.registers[baseReg] + arm.state.registers[offsetReg]

		if sign {
			if hi {
				if arm.decodeOnly {
					return &DisasmEntry{
						Operator: "LDSH",
						Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
					}
				}

				// load sign-extended halfword
				arm.state.registers[reg] = uint32(arm.read16bit(addr, false))

				// masking after cycle accumulation
				if arm.state.registers[reg]&0x8000 == 0x8000 {
					arm.state.registers[reg] |= 0xffff0000
				}

				// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - fillPipeline() will be called if necessary
				arm.Ncycle(dataRead, addr)
				arm.Icycle()

				return nil
			}

			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDSB",
					Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
				}
			}

			// load sign-extended byte
			arm.state.registers[reg] = uint32(arm.read8bit(addr))
			if arm.state.registers[reg]&0x0080 == 0x0080 {
				arm.state.registers[reg] |= 0xffffff00
			}

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		if hi {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDRH",
					Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
				}
			}

			// load halfword
			arm.state.registers[reg] = uint32(arm.read16bit(addr, false))

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "STRH",
				Operand:  fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg),
			}
		}

		// store halfword
		arm.write16bit(addr, uint16(arm.state.registers[reg]), false)

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return nil
	}
}

func (arm *ARM) decodeThumbLoadStoreWithImmOffset(opcode uint16) decodeFunction {
	// format 9 - Load/store with immediate offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x1000 == 0x1000

	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "For word accesses (B = 0), the value specified by #Imm is a full 7-bit address, but must
	// be word-aligned (ie with bits 1:0 set to 0), since the assembler places #Imm >> 2 in
	// the Offset5 field." -- ARM7TDMI Data Sheet
	if !byteTransfer {
		offset <<= 2
	}

	return func() *DisasmEntry {
		// the actual address we'll be loading from (or storing to)
		addr := arm.state.registers[baseReg] + uint32(offset)

		if load {
			if byteTransfer {
				if arm.decodeOnly {
					return &DisasmEntry{
						Operator: "LDRB",
						Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
					}
				}

				arm.state.registers[reg] = uint32(arm.read8bit(addr))

				// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - fillPipeline() will be called if necessary
				arm.Ncycle(dataRead, addr)
				arm.Icycle()

				return nil
			}

			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDR",
					Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
				}
			}

			arm.state.registers[reg] = arm.read32bit(addr, false)

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		// store
		if byteTransfer {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "STRB",
					Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
				}
			}

			arm.write8bit(addr, uint8(arm.state.registers[reg]))

			// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			arm.storeRegisterCycles(addr)

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "STR",
				Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
			}
		}

		arm.write32bit(addr, arm.state.registers[reg], false)

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return nil
	}
}

func (arm *ARM) decodeThumbLoadStoreHalfword(opcode uint16) decodeFunction {
	// format 10 - Load/store halfword
	load := opcode&0x0800 == 0x0800
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "#Imm is a full 6-bit address but must be halfword-aligned (ie with bit 0 set to 0) since
	// the assembler places #Imm >> 1 in the Offset5 field." -- ARM7TDMI Data Sheet
	offset <<= 1

	return func() *DisasmEntry {
		// the actual address we'll be loading from (or storing to)
		addr := arm.state.registers[baseReg] + uint32(offset)

		if load {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDRH",
					Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
				}
			}

			arm.state.registers[reg] = uint32(arm.read16bit(addr, false))

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "STRH",
				Operand:  fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset),
			}
		}

		arm.write16bit(addr, uint16(arm.state.registers[reg]), false)

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return nil
	}
}

func (arm *ARM) decodeThumbSPRelativeLoadStore(opcode uint16) decodeFunction {
	// format 11 - SP-relative load/store
	load := opcode&0x0800 == 0x0800
	reg := (opcode & 0x07ff) >> 8
	offset := uint32(opcode & 0xff)

	// The offset supplied in #Imm is a full 10-bit address, but must always be word-aligned
	// (ie bits 1:0 set to 0), since the assembler places #Imm >> 2 in the Word8 field.
	offset <<= 2

	return func() *DisasmEntry {
		// the actual address we'll be loading from (or storing to)
		addr := arm.state.registers[rSP] + offset

		if load {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "LDR",
					Operand:  fmt.Sprintf("R%d, [SP, #$%02x] ", reg, offset),
				}
			}

			arm.state.registers[reg] = arm.read32bit(addr, false)

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "STR",
				Operand:  fmt.Sprintf("R%d, [SP, #$%02x] ", reg, offset),
			}
		}

		arm.write32bit(addr, arm.state.registers[reg], false)

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return nil
	}
}

func (arm *ARM) decodeThumbLoadAddress(opcode uint16) decodeFunction {
	// format 12 - Load address
	sp := opcode&0x0800 == 0x800
	destReg := (opcode & 0x700) >> 8
	offset := opcode & 0x00ff

	// offset is a word aligned 10 bit address
	offset <<= 2

	return func() *DisasmEntry {
		if sp {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ADD",
					Operand:  fmt.Sprintf("R%d, [SP, #$%02x] ", destReg, offset),
				}
			}

			arm.state.registers[destReg] = arm.state.registers[rSP] + uint32(offset)

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "ADD",
				Operand:  fmt.Sprintf("R%d, [PC, #$%02x] ", destReg, offset),
			}
		}

		// "Where the PC is used as the source register (SP = 0), bit 1 of the PC is always read
		// as 0. The value of the PC will be 4 bytes greater than the address of the instruction
		// before bit 1 is forced to 0"
		pc := arm.state.registers[rPC]&0xfffffffd + uint32(offset)
		arm.state.registers[destReg] = AlignTo32bits(pc)

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return nil
	}
}

func (arm *ARM) decodeThumbAddOffsetToSP(opcode uint16) decodeFunction {
	// format 13 - Add offset to stack pointer
	sign := opcode&0x80 == 0x80
	imm := uint32(opcode & 0x7f)

	// The offset specified by #Imm can be up to -/+ 508, but must be word-aligned (ie with
	// bits 1:0 set to 0) since the assembler converts #Imm to an 8-bit sign + magnitude
	// number before placing it in field SWord7.
	imm <<= 2

	return func() *DisasmEntry {
		if sign {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "ADD",
					Operand:  fmt.Sprintf("SP, -#%d ", imm),
				}
			}

			arm.state.registers[rSP] -= imm

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - no additional cycles

			return nil
		}

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "ADD",
				Operand:  fmt.Sprintf("SP, #$%02x ", imm),
			}
		}

		arm.state.registers[rSP] += imm

		// status register not changed

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - no additional cycles

		return nil
	}
}

func (arm *ARM) decodeThumbPushPopRegisters(opcode uint16) decodeFunction {
	// format 14 - Push/pop registers

	// the ARM pushes registers in descending order and pops in ascending
	// order. in other words the LR is pushed first and PC is popped last

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := uint8(opcode & 0x00ff)

	return func() *DisasmEntry {
		if load {
			if arm.decodeOnly {
				if pclr {
					return &DisasmEntry{
						Operator: "POP",
						Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', regList, "PC")),
					}
				}
				return &DisasmEntry{
					Operator: "POP",
					Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', regList, "")),
				}
			}

			// start_address = SP
			// end_address = SP + 4*(R + Number_Of_Set_Bits_In(register_list))
			// address = start_address
			// for i = 0 to 7
			//		if register_list[i] == 1 then
			//			Ri = Memory[address,4]
			//			address = address + 4
			// if R == 1 then
			//		value = Memory[address,4]
			//		PC = value AND 0xFFFFFFFE
			// if (architecture version 5 or above) then
			//		T Bit = value[0]
			// address = address + 4
			// assert end_address = address
			// SP = end_address

			// start at stack pointer at work upwards
			addr := arm.state.registers[rSP]

			// read each register in turn (from lower to highest)
			numMatches := 0
			for i := 0; i <= 7; i++ {
				// shift single-bit mask
				m := uint8(0x01 << i)

				// read register if indicated by regList
				if regList&m == m {
					numMatches++

					// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
					// - N cycle on first match
					// - S cycles on subsequent matches
					if numMatches == 1 {
						arm.Ncycle(dataRead, addr)
					} else {
						arm.Scycle(dataRead, addr)
					}

					arm.state.registers[i] = arm.read32bit(addr, true)
					addr += 4
				}
			}

			// load PC register after all other registers
			if pclr {
				numMatches++

				// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - N cycle if first match and S cycle otherwise
				// - fillPipeline() will be called if necessary
				if numMatches == 1 {
					arm.Ncycle(dataRead, addr)
				} else {
					arm.Scycle(dataRead, addr)
				}

				// chop the odd bit off the new PC value
				v := arm.read32bit(addr, true) & 0xfffffffe

				// adjust popped LR value before assigning to the PC
				arm.state.registers[rPC] = v + 2
				addr += 4
			}

			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			arm.Icycle()

			// leave stackpointer at final address
			arm.state.registers[rSP] = addr

			return nil
		}

		if arm.decodeOnly {
			if pclr {
				return &DisasmEntry{
					Operator: "PUSH",
					Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', regList, "LR")),
				}
			}
			return &DisasmEntry{
				Operator: "PUSH",
				Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', regList, "")),
			}
		}

		// store

		// start_address = SP - 4*(R + Number_Of_Set_Bits_In(register_list))
		// end_address = SP - 4
		// address = start_address
		// for i = 0 to 7
		//		if register_list[i] == 1
		//			Memory[address,4] = Ri
		//			address = address + 4
		// if R == 1
		//		Memory[address,4] = LR
		//		address = address + 4
		// assert end_address == address - 4
		// SP = SP - 4*(R + Number_Of_Set_Bits_In(register_list))

		// number of pushes to perform. count number of bits in regList and adjust
		// for PC/LR flag. each push requires 4 bytes of space
		var c uint32
		if pclr {
			c = (uint32(bits.OnesCount8(regList)) + 1) * 4
		} else {
			c = uint32(bits.OnesCount8(regList)) * 4
		}

		// push occurs from the new low stack address upwards to the current stack
		// address (before the pushes)
		addr := arm.state.registers[rSP] - c

		// write each register in turn (from lower to highest)
		numMatches := 0
		for i := 0; i <= 7; i++ {
			// shift single-bit mask
			m := uint8(0x01 << i)

			// write register if indicated by regList
			if regList&m == m {
				numMatches++

				// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - storeRegisterCycles() on first match
				// - S cycles on subsequent match
				// - next prefetch cycle will be N
				if numMatches == 1 {
					arm.storeRegisterCycles(addr)
				} else {
					arm.Scycle(dataWrite, addr)
				}

				arm.write32bit(addr, arm.state.registers[i], true)
				addr += 4
			}
		}

		// write LR register after all the other registers
		if pclr {
			numMatches++

			lr := arm.state.registers[rLR]
			arm.write32bit(addr, lr, true)

			// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			if numMatches == 1 {
				arm.storeRegisterCycles(addr)
			} else {
				arm.Scycle(dataWrite, addr)
			}
		}

		// update stack pointer. note that this is the address we started the push
		// sequence from above. this is correct.
		arm.state.registers[rSP] -= c

		return nil
	}
}

func (arm *ARM) decodeThumbMultipleLoadStore(opcode uint16) decodeFunction {
	// format 15 - Multiple load/store
	load := opcode&0x0800 == 0x0800
	baseReg := uint32(opcode&0x07ff) >> 8
	regList := uint8(opcode & 0x00ff)

	return func() *DisasmEntry {
		if arm.decodeOnly {
			if load {
				return &DisasmEntry{
					Operator: "LDMIA",
					Operand:  fmt.Sprintf("R%d!, {%s}", baseReg, reglistToMnemonic('R', regList, "")),
				}
			}
			return &DisasmEntry{
				Operator: "STMIA",
				Operand:  fmt.Sprintf("R%d!, {%s}", baseReg, reglistToMnemonic('R', regList, "")),
			}
		}

		// load/store the registers in the list starting at address
		// in the base register
		addr := arm.state.registers[baseReg]

		// all ARM references say that the base register is updated as a result of
		// the multi-load. what isn't clear is what happens if the base register is
		// part of the update. observation of a bug in a confidential Andrew Davie
		// project however, demonstrates that we should *not* update the base
		// registere in those situations.
		//
		// this rule is not required for multiple store or for push/pop, where the
		// potential conflict never arises.
		updateBaseReg := true

		if load {
			numMatches := 0
			for i := 0; i <= 15; i++ {
				r := regList >> i
				if r&0x01 == 0x01 {
					// check if baseReg is being updated
					if i == int(baseReg) {
						updateBaseReg = false
					}

					numMatches++

					// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
					// - N cycle on first match
					// - S cycles on subsequent matches
					// - fillPipeline() will be called if PC register is matched
					if numMatches == 1 {
						arm.Ncycle(dataWrite, addr)
					} else {
						arm.Scycle(dataWrite, addr)
					}

					arm.state.registers[i] = arm.read32bit(addr, true)
					addr += 4
				}
			}

			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			arm.Icycle()

			// no updating of base register if base register was part of the regList
			if updateBaseReg {
				arm.state.registers[baseReg] = addr
			}

			return nil
		}

		// store

		numMatches := 0
		for i := 0; i <= 15; i++ {
			r := regList >> i
			if r&0x01 == 0x01 {
				numMatches++

				// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - storeRegisterCycles() on first match
				// - S cycles on subsequent match
				// - next prefetch cycle will be N
				if numMatches == 1 {
					arm.storeRegisterCycles(addr)
				} else {
					arm.Scycle(dataWrite, addr)
				}

				arm.write32bit(addr, arm.state.registers[i], true)
				addr += 4
			}
		}

		// write back the new base address
		arm.state.registers[baseReg] = addr

		return nil
	}
}

func (arm *ARM) decodeThumbConditionalBranch(opcode uint16) decodeFunction {
	// format 16 - Conditional branch
	cond := uint8((opcode & 0x0f00) >> 8)
	offset := uint32(opcode & 0x00ff)

	// offset is a nine-bit two's complement value
	offset <<= 1

	// sign extend
	if offset&0x100 == 0x100 {
		offset |= 0xffffff00
	}
	offset += 2

	// branch target as a string
	operand := arm.branchTargetOffsetFromPC(int64(offset))

	return func() *DisasmEntry {
		passed, mnemonic := arm.state.status.condition(cond)

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: mnemonic,
				Operand:  operand,
			}
		}

		// adjust PC if condition has been met
		if passed {
			arm.state.registers[rPC] += offset
		}

		// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return nil
	}
}

func (arm *ARM) decodeThumbSoftwareInterrupt(opcode uint16) decodeFunction {
	// format 17 - Software interrupt"
	panic(fmt.Sprintf("unimplemented (software interrupt) thumb instruction (%04x)", opcode))
}

func (arm *ARM) decodeThumbUnconditionalBranch(opcode uint16) decodeFunction {
	// format 18 - Unconditional branch
	offset := uint32(opcode & 0x07ff)

	// offset is a nine-bit two's complement value
	offset <<= 1

	// sign extend
	if offset&0x800 == 0x0800 {
		offset |= 0xfffff800
	}
	offset += 2

	// branch target as a string
	operand := arm.branchTargetOffsetFromPC(int64(offset))

	return func() *DisasmEntry {
		// we'll be adjusting the offset value so we need to make a copy of it
		offset := offset

		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "BAL",
				Operand:  operand,
			}
		}

		arm.state.registers[rPC] += offset

		// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return nil
	}
}

func (arm *ARM) decodeThumbLongBranchWithLink(opcode uint16) decodeFunction {
	// format 19 - Long branch with link
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	if low {
		offset <<= 1

		return func() *DisasmEntry {
			tgt := arm.state.registers[rLR] + offset

			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "BL",
					Operand:  arm.branchTarget(tgt),
				}
			}
			pc := arm.state.registers[rPC]
			arm.state.registers[rPC] = tgt
			arm.state.registers[rLR] = pc - 1
			return nil
		}
	}

	offset <<= 12

	// sign extend
	if offset&0x400000 == 0x400000 {
		offset |= 0xffc00000
	}
	offset += 2

	return func() *DisasmEntry {
		if arm.decodeOnly {
			return nil
		}
		arm.state.registers[rLR] = arm.state.registers[rPC] + offset
		return nil
	}
}
