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
	"fmt"
	"math/bits"

	"github.com/jetsetilly/gopher2600/logger"
)

func (arm *ARM) decodeThumb(opcode uint16) func(uint16) {
	// working backwards up the table in Figure 5-1 of the ARM7TDMI Data Sheet.
	if opcode&0xf000 == 0xf000 {
		// format 19 - Long branch with link
		return arm.thumbLongBranchWithLink
	} else if opcode&0xf000 == 0xe000 {
		// format 18 - Unconditional branch
		return arm.thumbUnconditionalBranch
	} else if opcode&0xff00 == 0xdf00 {
		// format 17 - Software interrupt"
		return arm.thumbSoftwareInterrupt
	} else if opcode&0xf000 == 0xd000 {
		// format 16 - Conditional branch
		return arm.thumbConditionalBranch
	} else if opcode&0xf000 == 0xc000 {
		// format 15 - Multiple load/store
		return arm.thumbMultipleLoadStore
	} else if opcode&0xf600 == 0xb400 {
		// format 14 - Push/pop registers
		return arm.thumbPushPopRegisters
	} else if opcode&0xff00 == 0xb000 {
		// format 13 - Add offset to stack pointer
		return arm.thumbAddOffsetToSP
	} else if opcode&0xf000 == 0xa000 {
		// format 12 - Load address
		return arm.thumbLoadAddress
	} else if opcode&0xf000 == 0x9000 {
		// format 11 - SP-relative load/store
		return arm.thumbSPRelativeLoadStore
	} else if opcode&0xf000 == 0x8000 {
		// format 10 - Load/store halfword
		return arm.thumbLoadStoreHalfword
	} else if opcode&0xe000 == 0x6000 {
		// format 9 - Load/store with immediate offset
		return arm.thumbLoadStoreWithImmOffset
	} else if opcode&0xf200 == 0x5200 {
		// format 8 - Load/store sign-extended byte/halfword
		return arm.thumbLoadStoreSignExtendedByteHalford
	} else if opcode&0xf200 == 0x5000 {
		// format 7 - Load/store with register offset
		return arm.thumbLoadStoreWithRegisterOffset
	} else if opcode&0xf800 == 0x4800 {
		// format 6 - PC-relative load
		return arm.thumbPCrelativeLoad
	} else if opcode&0xfc00 == 0x4400 {
		// format 5 - Hi register operations/branch exchange
		return arm.thumbHiRegisterOps
	} else if opcode&0xfc00 == 0x4000 {
		// format 4 - ALU operations
		return arm.thumbALUoperations
	} else if opcode&0xe000 == 0x2000 {
		// format 3 - Move/compare/add/subtract immediate
		return arm.thumbMovCmpAddSubImm
	} else if opcode&0xf800 == 0x1800 {
		// format 2 - Add/subtract
		return arm.thumbAddSubtract
	} else if opcode&0xe000 == 0x0000 {
		// format 1 - Move shifted register
		return arm.thumbMoveShiftedRegister
	}

	panic(fmt.Sprintf("undecoded thumb instruction (%04x)", opcode))
}

func (arm *ARM) thumbMoveShiftedRegister(opcode uint16) {
	// format 1 - Move shifted register
	op := (opcode & 0x1800) >> 11
	shift := (opcode & 0x7c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// in this class of operation the src register may also be the dest
	// register so we need to make a note of the value before it is
	// overwrittten
	src := arm.registers[srcReg]

	switch op {
	case 0b00:
		// if immed_5 == 0
		//	C Flag = unaffected
		//	Rd = Rm
		// else /* immed_5 > 0 */
		//	C Flag = Rm[32 - immed_5]
		//	Rd = Rm Logical_Shift_Left immed_5

		if shift == 0 {
			arm.registers[destReg] = src
		} else {
			m := uint32(0x01) << (32 - shift)
			arm.status.carry = src&m == m
			arm.registers[destReg] = arm.registers[srcReg] << shift
		}
	case 0b01:
		// if immed_5 == 0
		//		C Flag = Rm[31]
		//		Rd = 0
		// else /* immed_5 > 0 */
		//		C Flag = Rm[immed_5 - 1]
		//		Rd = Rm Logical_Shift_Right immed_5

		if shift == 0 {
			arm.status.carry = src&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = src >> shift
		}
	case 0b10:
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
			arm.status.carry = src&0x80000000 == 0x80000000
			if arm.status.carry {
				arm.registers[destReg] = 0xffffffff
			} else {
				arm.registers[destReg] = 0x00000000
			}
		} else { // shift > 0
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			a := src >> shift
			if src&0x80000000 == 0x80000000 {
				a |= (0xffffffff << (32 - shift))
			}
			arm.registers[destReg] = a
		}

	case 0x11:
		panic(fmt.Sprintf("illegal (move shifted register) thumb operation (%04b)", op))
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])

	if destReg == rPC {
		logger.Log("ARM7", "shift and store in PC is not possible in thumb mode")
	}

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	if shift > 0 {
		arm.Icycle()
	}
}

func (arm *ARM) thumbAddSubtract(opcode uint16) {
	// format 2 - Add/subtract
	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200
	imm := uint32((opcode & 0x01c0) >> 6)
	srcReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	// value to work with is either an immediate value or is in a register
	val := imm
	if !immediate {
		val = arm.registers[imm]
	}

	if subtract {
		arm.status.setCarry(arm.registers[srcReg], ^val, 1)
		arm.status.setOverflow(arm.registers[srcReg], ^val, 1)
		arm.registers[destReg] = arm.registers[srcReg] - val
	} else {
		arm.status.setCarry(arm.registers[srcReg], val, 0)
		arm.status.setOverflow(arm.registers[srcReg], val, 0)
		arm.registers[destReg] = arm.registers[srcReg] + val
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

// "The instructions in this group perform operations between a Lo register and
// an 8-bit immediate value".
func (arm *ARM) thumbMovCmpAddSubImm(opcode uint16) {
	// format 3 - Move/compare/add/subtract immediate
	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode & 0x00ff)

	switch op {
	case 0b00:
		arm.registers[destReg] = imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b01:
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		cmp := arm.registers[destReg] - imm
		arm.status.isNegative(cmp)
		arm.status.isZero(cmp)
	case 0b10:
		arm.status.setCarry(arm.registers[destReg], imm, 0)
		arm.status.setOverflow(arm.registers[destReg], imm, 0)
		arm.registers[destReg] += imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b11:
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		arm.registers[destReg] -= imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	}

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

// "The following instructions perform ALU operations on a Lo register pair".
func (arm *ARM) thumbALUoperations(opcode uint16) {
	// format 4 - ALU operations
	op := (opcode & 0x03c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	var shift uint32
	var mul bool
	var mulOperand uint32

	switch op {
	case 0b0000:
		arm.registers[destReg] &= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0001:
		arm.registers[destReg] ^= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0010:
		shift = arm.registers[srcReg]

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
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] <<= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x01 == 0x01
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0011:
		shift = arm.registers[srcReg]

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
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] >>= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0100:
		shift = arm.registers[srcReg]

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
			src := arm.registers[destReg]
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			a := src >> shift
			if src&0x80000000 == 0x80000000 {
				a |= (0xffffffff << (32 - shift))
			}
			arm.registers[destReg] = a
		} else if shift >= 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			if !arm.status.carry {
				arm.registers[destReg] = 0x00
			} else {
				arm.registers[destReg] = 0xffffffff
			}
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0101:
		if arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.registers[destReg] += arm.registers[srcReg]
			arm.registers[destReg]++
		} else {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.registers[destReg] += arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0110:
		if !arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.registers[destReg] -= arm.registers[srcReg]
			arm.registers[destReg]--
		} else {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.registers[destReg] -= arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0111:
		shift = arm.registers[srcReg]

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
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] = bits.RotateLeft32(arm.registers[destReg], -int(shift))
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1000:
		w := arm.registers[destReg] & arm.registers[srcReg]
		arm.status.isZero(w)
		arm.status.isNegative(w)
	case 0b1001:
		arm.status.setCarry(0, ^arm.registers[srcReg], 1)
		arm.status.setOverflow(0, ^arm.registers[srcReg], 1)
		arm.registers[destReg] = -arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1010:
		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1011:
		arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
		arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
		cmp := arm.registers[destReg] + arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1100:
		arm.registers[destReg] |= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1101:
		mul = true
		mulOperand = arm.registers[srcReg]
		arm.registers[destReg] *= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1110:
		arm.registers[destReg] &= ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1111:
		arm.registers[destReg] = ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
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
}

func (arm *ARM) thumbHiRegisterOps(opcode uint16) {
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

	switch op {
	case 0b00:
		// not two's complement
		arm.registers[destReg] += arm.registers[srcReg]

		// status register not changed

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	case 0b01:
		// alu_out = Rn - Rm
		// N Flag = alu_out[31]
		// Z Flag = if alu_out == 0 then 1 else 0
		// C Flag = NOT BorrowFrom(Rn - Rm)
		// V Flag = OverflowFrom(Rn - Rm)

		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)

		return
	case 0b10:
		// check to see if we're copying the LR to the PC. if we are than adjust
		// the PC by 2 (as though the prefetch has occurred)
		if srcReg == rLR && destReg == rPC {
			arm.registers[destReg] = arm.registers[srcReg] + 2
		} else {
			arm.registers[destReg] = arm.registers[srcReg]
		}

		// status register not changed

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	case 0b11:
		thumbMode := arm.registers[srcReg]&0x01 == 0x01

		var newPC uint32

		// "ARM7TDMI Data Sheet" page 5-15:
		//
		// "If R15 is used as an operand, the value will be the address of the instruction + 4 with
		// bit 0 cleared. Executing a BX PC in THUMB state from a non-word aligned address
		// will result in unpredictable execution."
		if srcReg == rPC {
			newPC = arm.registers[rPC] + 2
		} else {
			newPC = (arm.registers[srcReg] & 0x7ffffffe) + 2
		}

		if thumbMode {
			arm.registers[rPC] = newPC

			if arm.disasm != nil {
				arm.disasmExecutionNotes = "branch exchange to thumb code"
				arm.disasmUpdateNotes = true
			}

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			return
		}

		// switch to ARM mode. emulate function call.
		res, err := arm.hook.ARMinterrupt(arm.registers[rPC]-4, arm.registers[2], arm.registers[3])
		if err != nil {
			arm.continueExecution = false
			arm.executionError = err
			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			//  - interrupted
			return
		}

		if arm.disasm != nil {
			if res.InterruptEvent != "" {
				arm.disasmExecutionNotes = fmt.Sprintf("ARM function (%08x) %s", arm.registers[rPC]-4, res.InterruptEvent)
			} else {
				arm.disasmExecutionNotes = fmt.Sprintf("ARM function (%08x)", arm.registers[rPC]-4)
			}
			arm.disasmUpdateNotes = true
		}

		// if ARMinterrupt returns false this indicates that the
		// function at the quoted program counter is not recognised and
		// has nothing to do with the cartridge mapping. at this point
		// we can assume that the main() function call is done and we
		// can return to the VCS emulation.
		if !res.InterruptServiced {
			arm.continueExecution = false
			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			//  - interrupted
			return
		}

		// ARM function updates the ARM registers
		if res.SaveResult {
			arm.registers[res.SaveRegister] = res.SaveValue
		}

		// the end of the emulated function will have an operation that
		// switches back to thumb mode, and copies the link register to the
		// program counter. we need to emulate that too.
		arm.registers[rPC] = arm.registers[rLR] + 2

		// add cycles used by the ARM program
		arm.armInterruptCycles(res)

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
	}
}

func (arm *ARM) thumbPCrelativeLoad(opcode uint16) {
	// format 6 - PC-relative load
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	// "Bit 1 of the PC value is forced to zero for the purpose of this
	// calculation, so the address is always word-aligned."
	pc := arm.registers[rPC] & 0xfffffffc

	// immediate value is not two's complement (surprisingly)
	addr := pc + imm
	arm.registers[destReg] = arm.read32bit(addr)

	// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
	arm.Ncycle(dataRead, addr)
	arm.Icycle()
}

func (arm *ARM) thumbLoadStoreWithRegisterOffset(opcode uint16) {
	// format 7 - Load/store with register offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if load {
		if byteTransfer {
			arm.registers[reg] = uint32(arm.read8bit(addr))

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}

		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if byteTransfer {
		arm.write8bit(addr, uint8(arm.registers[reg]))

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) thumbLoadStoreSignExtendedByteHalford(opcode uint16) {
	// format 8 - Load/store sign-extended byte/halfword
	hi := opcode&0x0800 == 0x800
	sign := opcode&0x0400 == 0x400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if sign {
		if hi {
			// load sign-extended halfword
			arm.registers[reg] = uint32(arm.read16bit(addr))

			// masking after cycle accumulation
			if arm.registers[reg]&0x8000 == 0x8000 {
				arm.registers[reg] |= 0xffff0000
			}

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}
		// load sign-extended byte
		arm.registers[reg] = uint32(arm.read8bit(addr))
		if arm.registers[reg]&0x0080 == 0x0080 {
			arm.registers[reg] |= 0xffffff00
		}

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if hi {
		// load halfword
		arm.registers[reg] = uint32(arm.read16bit(addr))

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	// store halfword
	arm.write16bit(addr, uint16(arm.registers[reg]))

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) thumbLoadStoreWithImmOffset(opcode uint16) {
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

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		if byteTransfer {
			arm.registers[reg] = uint32(arm.read8bit(addr))

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}

		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	// store
	if byteTransfer {
		arm.write8bit(addr, uint8(arm.registers[reg]))

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) thumbLoadStoreHalfword(opcode uint16) {
	// format 10 - Load/store halfword
	load := opcode&0x0800 == 0x0800
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "#Imm is a full 6-bit address but must be halfword-aligned (ie with bit 0 set to 0) since
	// the assembler places #Imm >> 1 in the Offset5 field." -- ARM7TDMI Data Sheet
	offset <<= 1

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		arm.registers[reg] = uint32(arm.read16bit(addr))

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	arm.write16bit(addr, uint16(arm.registers[reg]))

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) thumbSPRelativeLoadStore(opcode uint16) {
	// format 11 - SP-relative load/store
	load := opcode&0x0800 == 0x0800
	reg := (opcode & 0x07ff) >> 8
	offset := uint32(opcode & 0xff)

	// The offset supplied in #Imm is a full 10-bit address, but must always be word-aligned
	// (ie bits 1:0 set to 0), since the assembler places #Imm >> 2 in the Word8 field.
	offset <<= 2

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[rSP] + offset

	if load {
		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) thumbLoadAddress(opcode uint16) {
	// format 12 - Load address
	sp := opcode&0x0800 == 0x800
	destReg := (opcode & 0x700) >> 8
	offset := opcode & 0x00ff

	// offset is a word aligned 10 bit address
	offset <<= 2

	if sp {
		arm.registers[destReg] = arm.registers[rSP] + uint32(offset)

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	}

	arm.registers[destReg] = arm.registers[rPC] + uint32(offset)

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

func (arm *ARM) thumbAddOffsetToSP(opcode uint16) {
	// format 13 - Add offset to stack pointer
	sign := opcode&0x80 == 0x80
	imm := uint32(opcode & 0x7f)

	// The offset specified by #Imm can be up to -/+ 508, but must be word-aligned (ie with
	// bits 1:0 set to 0) since the assembler converts #Imm to an 8-bit sign + magnitude
	// number before placing it in field SWord7.
	imm <<= 2

	if sign {
		arm.registers[rSP] -= imm

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - no additional cycles

		return
	}

	arm.registers[rSP] += imm

	// status register not changed

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - no additional cycles
}

func (arm *ARM) thumbPushPopRegisters(opcode uint16) {
	// format 14 - Push/pop registers

	// the ARM pushes registers in descending order and pops in ascending
	// order. in other words the LR is pushed first and PC is popped last

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := uint8(opcode & 0x00ff)

	if load {
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
		addr := arm.registers[rSP]

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

				arm.registers[i] = arm.read32bit(addr)
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
			v := arm.read32bit(addr) & 0xfffffffe

			// adjust popped LR value before assigning to the PC
			arm.registers[rPC] = v + 2
			addr += 4
		}

		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()

		// leave stackpointer at final address
		arm.registers[rSP] = addr

		return
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
	addr := arm.registers[rSP] - c

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

			arm.write32bit(addr, arm.registers[i])
			addr += 4
		}
	}

	// write LR register after all the other registers
	if pclr {
		numMatches++

		lr := arm.registers[rLR]
		arm.write32bit(addr, lr)

		// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		if numMatches == 1 {
			arm.storeRegisterCycles(addr)
		} else {
			arm.Scycle(dataWrite, addr)
		}
	}

	// update stack pointer. note that this is the address we started the push
	// sequence from above. this is correct.
	arm.registers[rSP] -= c
}

func (arm *ARM) thumbMultipleLoadStore(opcode uint16) {
	// format 15 - Multiple load/store
	load := opcode&0x0800 == 0x0800
	baseReg := uint32(opcode&0x07ff) >> 8
	regList := opcode & 0xff

	// load/store the registers in the list starting at address
	// in the base register
	addr := arm.registers[baseReg]

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

				arm.registers[i] = arm.read32bit(addr)
				addr += 4
			}
		}

		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()

		// no updating of base register if base register was part of the regList
		if updateBaseReg {
			arm.registers[baseReg] = addr
		}

		return
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

			arm.write32bit(addr, arm.registers[i])
			addr += 4
		}
	}

	// write back the new base address
	arm.registers[baseReg] = addr
}

func (arm *ARM) thumbConditionalBranch(opcode uint16) {
	// format 16 - Conditional branch
	cond := (opcode & 0x0f00) >> 8
	offset := uint32(opcode & 0x00ff)

	b := false

	switch cond {
	case 0b0000:
		// BEQ
		b = arm.status.zero
	case 0b0001:
		// BNE
		b = !arm.status.zero
	case 0b0010:
		// BCS
		b = arm.status.carry
	case 0b0011:
		// BCC
		b = !arm.status.carry
	case 0b0100:
		// BMI
		b = arm.status.negative
	case 0b0101:
		// BPL
		b = !arm.status.negative
	case 0b0110:
		// BVS
		b = arm.status.overflow
	case 0b0111:
		// BVC
		b = !arm.status.overflow
	case 0b1000:
		// BHI
		b = arm.status.carry && !arm.status.zero
	case 0b1001:
		// BLS
		b = !arm.status.carry || arm.status.zero
	case 0b1010:
		// BGE
		b = (arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow)
	case 0b1011:
		// BLT
		b = (arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow)
	case 0b1100:
		// BGT
		b = !arm.status.zero && ((arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow))
	case 0b1101:
		// BLE
		b = arm.status.zero || ((arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow))
	case 0b1110:
		// undefined branch
		b = true
	case 0b1111:
		b = false
	}

	// offset is a nine-bit two's complement value
	offset <<= 1
	offset++

	var newPC uint32

	// get new PC value
	if offset&0x100 == 0x100 {
		// two's complement before subtraction
		offset ^= 0x1ff
		offset++
		newPC = arm.registers[rPC] - offset + 1
	} else {
		newPC = arm.registers[rPC] + offset + 1
	}

	// do branch
	if b {
		// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.registers[rPC] = newPC
	}

	if arm.disasm != nil {
		if b {
			arm.disasmExecutionNotes = "branched"
		} else {
			arm.disasmExecutionNotes = "next"
		}
		arm.disasmUpdateNotes = true
	}
}

func (arm *ARM) thumbSoftwareInterrupt(opcode uint16) {
	// format 17 - Software interrupt"
	panic(fmt.Sprintf("unimplemented (software interrupt) thumb instruction (%04x)", opcode))
}

func (arm *ARM) thumbUnconditionalBranch(opcode uint16) {
	// format 18 - Unconditional branch
	offset := uint32(opcode&0x07ff) << 1

	if offset&0x800 == 0x0800 {
		// two's complement before subtraction
		offset ^= 0xfff
		offset++
		arm.registers[rPC] -= offset - 2
	} else {
		arm.registers[rPC] += offset + 2
	}

	// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

func (arm *ARM) thumbLongBranchWithLink(opcode uint16) {
	// format 19 - Long branch with link
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	// there is no direct ARM equivalent for this instruction.

	if low {
		// second instruction

		offset <<= 1
		pc := arm.registers[rPC]
		arm.registers[rPC] = arm.registers[rLR] + offset
		arm.registers[rLR] = pc - 1

		// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// -- no additional cycles for second instruction in BL
		// -- change of PC is captured by expectedPC check in Run() function loop

		return
	}

	// first instruction

	offset <<= 12

	if offset&0x400000 == 0x400000 {
		// two's complement before subtraction
		offset ^= 0x7fffff
		offset++
		arm.registers[rLR] = arm.registers[rPC] - offset + 2
	} else {
		arm.registers[rLR] = arm.registers[rPC] + offset + 2
	}

	// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// -- no additional cycles for first instruction in BL
}
