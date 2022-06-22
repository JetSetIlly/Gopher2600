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

// see thumb2.go for documentation information

package arm

import (
	"fmt"
	"math/bits"
)

func (arm *ARM) decode32bitThumb2(opcode uint16) func(uint16) {
	// Two tables for top level decoding of 32bit Thumb-2 instructions.
	//
	// "3.3 Instruction encoding for 32-bit Thumb instructions" of "Thumb-2 Supplement"
	//		and
	// "A5.3 32-bit Thumb instruction encoding" of "ARMv7-M"
	//
	// Both with different emphasis but the table in the "Thumb-2 Supplement"
	// was used.

	if opcode&0xef00 == 0xef00 {
		// coprocessor
		panic("coprocessor")
	} else if opcode&0xf800 == 0xf000 {
		// branches, miscellaneous control
		//  OR
		// data processing: immediate, including bitfield and saturate
		return arm.thumb2BranchesORDataProcessing
	} else if opcode&0xfe40 == 0xe800 {
		// load and store multiple, RFE and SRS
		return arm.thumb2LoadStoreMultiple
	} else if opcode&0xfe40 == 0xe840 {
		// load and store double and exclusive and table branch
		return arm.thumb2LoadStoreDoubleEtc
	} else if opcode&0xfe00 == 0xf800 {
		// load and store single data item, memory hints
		return arm.thumb2LoadStoreSingle
	} else if opcode&0xee00 == 0xea00 {
		// data processing, no immediate operand
		return arm.thumb2DataProcessingNonImmediate
	}

	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (upper half-word) (%04x)", opcode))
}

func (arm *ARM) thumb2DataProcessingNonImmediate(opcode uint16) {
	// "3.3.2 Data processing instructions, non-immediate" of "Thumb-2 Supplement"

	Rn := arm.function32bitOpcode & 0x000f
	Rm := opcode & 0x000f
	Rd := (opcode & 0x0f00) >> 8

	if arm.function32bitOpcode&0xfe00 == 0xea00 {
		// "Data processing instructions with constant shift"
		// page 3-18 of "Thumb-2 Supplement"
		op := (arm.function32bitOpcode & 0x01e0) >> 5
		setFlags := arm.function32bitOpcode&0x0010 == 0x0010
		// sbz := (opcode & 0x8000) >> 15
		imm3 := (opcode & 0x7000) >> 12
		// Rd := (opcode & 0x0f00) >> 8
		imm2 := (opcode & 0x00c0) >> 6
		typ := (opcode & 0x0030) >> 4
		imm5 := (imm3 << 2) | imm2

		switch op {
		case 0b0000:
			if Rd == rPC && setFlags {
				panic("TST")
			} else {
				// "4.6.9 AND (register)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "AND"

				switch typ {
				case 0b00:
					carry := arm.registers[Rm]&0x80000000 == 0x80000000
					shifted := arm.registers[Rm] << imm5
					arm.registers[Rd] = arm.registers[Rn] & shifted
					if setFlags {
						arm.Status.isNegative(arm.registers[Rd])
						arm.Status.isZero(arm.registers[Rd])
						arm.Status.setCarry(carry)
						// overflow unchanged
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}
			}

		case 0b0001:
			// "4.6.16 BIC (register)" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "BIC"
			switch typ {
			case 0b00:
			default:
				carry := arm.registers[Rm]&0x80000000 == 0x80000000
				shifted := arm.registers[Rm] << imm5
				arm.registers[Rd] = arm.registers[Rn] & ^shifted
				if setFlags {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0010:
			// "Move, and immediate shift instructions"
			// page 3-19 of "Thumb-2 Supplement"

			if Rn == rPC {
				switch typ {
				case 0b00:
					if imm5 == 0b00000 {
						panic("move")
					} else {
						// "4.6.68 LSL (immediate)" of "Thumb-2 Supplement"
						// T2 encoding
						arm.fudge_thumb2disassemble32bit = "LSL"

						carry := arm.registers[Rm]&0x80000000 == 0x80000000
						arm.registers[Rd] = arm.registers[Rm] << imm5
						if setFlags {
							arm.Status.isNegative(arm.registers[Rd])
							arm.Status.isZero(arm.registers[Rd])
							arm.Status.setCarry(carry)
							// overflow unchanged
						}
					}
				case 0b01:
					// "4.6.70 LSR (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					arm.fudge_thumb2disassemble32bit = "LSR"

					carry := arm.registers[Rm]&0x80000000 == 0x80000000
					arm.registers[Rd] = arm.registers[Rm] >> imm5
					if setFlags {
						arm.Status.isNegative(arm.registers[Rd])
						arm.Status.isZero(arm.registers[Rd])
						arm.Status.setCarry(carry)
						// overflow unchanged
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}
			} else {
				panic("ORR")
			}

		case 0b0100:
			// "4.6.37 EOR (register)" of "Thumb-2 Supplement
			// T2 encoding
			arm.fudge_thumb2disassemble32bit = "EOR"

			switch typ {
			case 0b00:
				// with logical left shift
				carry := arm.registers[Rm]&0x80000000 == 0x80000000
				arm.registers[Rd] = arm.registers[Rn] ^ (arm.registers[Rm] << imm5)
				if setFlags {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			case 0b01:
				// with logical right shift
				carry := arm.registers[Rm]&0x80000000 == 0x80000000
				arm.registers[Rd] = arm.registers[Rn] ^ (arm.registers[Rm] >> imm5)
				if setFlags {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
			}

		case 0b1101:
			if Rd == rPC {
				// "4.6.30 CMP (register)" of "Thumb-2 Supplement"
				// T3 encoding
				arm.fudge_thumb2disassemble32bit = "CMP"

				switch typ {
				case 0b00:
					// with logical left shift
					shifted := arm.registers[Rm] << imm5
					result, carry, overflow := AddWithCarry(arm.registers[Rn], ^shifted, 1)
					if setFlags {
						arm.Status.isNegative(result)
						arm.Status.isZero(result)
						arm.Status.setCarry(carry)
						arm.Status.setOverflow(overflow)
					}
				case 0b01:
					// with logical right shift
					shifted := arm.registers[Rm] >> imm5
					result, carry, overflow := AddWithCarry(arm.registers[Rn], ^shifted, 1)
					if setFlags {
						arm.Status.isNegative(result)
						arm.Status.isZero(result)
						arm.Status.setCarry(carry)
						arm.Status.setOverflow(overflow)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) CMP", op, typ))
				}

			} else {
				// "4.6.177 SUB (register)" of "Thumb-2 Supplement"
				// T2 encoding
				arm.fudge_thumb2disassemble32bit = "SUB"

				switch typ {
				case 0b00:
					// with logical left shift
					shifted := arm.registers[Rm] << imm5
					result, carry, overflow := AddWithCarry(arm.registers[Rn], ^shifted, 1)
					arm.registers[Rd] = result
					if setFlags {
						arm.Status.isNegative(arm.registers[Rd])
						arm.Status.isZero(arm.registers[Rd])
						arm.Status.setCarry(carry)
						arm.Status.setOverflow(overflow)
					}
				case 0b01:
					// with logical right shift
					shifted := arm.registers[Rm] >> imm5
					result, carry, overflow := AddWithCarry(arm.registers[Rn], ^shifted, 1)
					arm.registers[Rd] = result
					if setFlags {
						arm.Status.isNegative(arm.registers[Rd])
						arm.Status.isZero(arm.registers[Rd])
						arm.Status.setCarry(carry)
						arm.Status.setOverflow(overflow)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) SUB", op, typ))
				}
			}

		default:
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b)", op))
		}
	} else if arm.function32bitOpcode&0xff80 == 0xfa00 {
		if opcode&0x0080 == 0x0000 {
			// "Register-controlled shift instructions"
			// page 3-19 of "Thumb-2 Supplement"
			op := (arm.function32bitOpcode & 0x0060) >> 5
			// s := (arm.function32bitOpcode & 0x0010) == 0x0010
			switch op {
			case 0b10:
				// "4.6.10 ASR (immediate)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "ASR"

				imm3 := (opcode & 0x7000) >> 12
				imm2 := (opcode & 0x00c0) >> 6
				imm5 := (imm3 << 2) | imm2
				s := arm.function32bitOpcode&0x0010 == 0x0010
				carry := arm.registers[Rm]&0x0001 == 0x0001
				arm.registers[Rd] = arm.registers[Rm] >> imm5
				if arm.Status.carry {
					arm.registers[Rd] |= 0x80000000
				}
				if s {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (reg controlled shift) (%02b)", op))
			}
		} else {
			// "Signed and unsigned extend instructions with optional addition"
			// page 3-20 of "Thumb-2 Supplement"
			op := (arm.function32bitOpcode & 0x0070) >> 4
			// sbz := opcode&0x0040 == 0x0040
			rot := (opcode & 0x0030) >> 4

			switch op {
			case 0b001:
				if Rn == rPC {
					// "4.6.226 UXTH" of "Thumb-2 Supplement"
					arm.fudge_thumb2disassemble32bit = "UXTH"

					v, _ := ROR_C(arm.registers[Rm], uint32(rot)<<3)
					arm.registers[Rd] = v & 0x0000ffff
				} else {
					// "4.6.223 UXTAH" of "Thumb-2 Supplement"
					arm.fudge_thumb2disassemble32bit = "UXTAH"

					v, _ := ROR_C(arm.registers[Rm], uint32(rot)<<3)
					arm.registers[Rd] = arm.registers[Rn] + (v & 0x0000ffff)
				}
			case 0b101:
				if Rn == rPC {
					// "4.6.224 UXTB" of "Thumb-2 Supplement"
					arm.fudge_thumb2disassemble32bit = "UXTB"

					v, _ := ROR_C(arm.registers[Rm], uint32(rot)<<3)
					arm.registers[Rd] = v & 0x000000ff
				} else {
					// "4.6.221 UXTAB" of "Thumb-2 Supplement"
					arm.fudge_thumb2disassemble32bit = "UXTAB"

					v, _ := ROR_C(arm.registers[Rm], uint32(rot)<<3)
					arm.registers[Rd] = arm.registers[Rn] + (v & 0x000000ff)
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (sign or zero extension with opt addition) (%03b)", op))
			}
		}
	} else if arm.function32bitOpcode&0xff80 == 0xfa80 {
		if opcode&0x0080 == 0x0000 {
			// "SIMD add and subtract"
			// page 3-21 of "Thumb-2 Supplement"
		} else {
			// "Other three-register data processing instructions"
			// page 3-23 of "Thumb-2 Supplement"
		}
	} else if arm.function32bitOpcode&0xff80 == 0xfb00 {
		// "32-bit multiplies and sum of absolute differences, with or without accumulate"
		// page 3-24 of "Thumb-2 Supplement"
		op := (arm.function32bitOpcode & 0x0070) >> 4
		Ra := (opcode & 0xf000) >> 12
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b000 && op2 == 0b0001 {
			// "4.6.75 MLS" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "MLS"

			arm.registers[Rd] = uint32(int32(arm.registers[Ra]) - int32(arm.registers[Rn])*int32(arm.registers[Rm]))
		} else {
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (32bit multiplies) (%03b/%04b)", op, op2))
		}
	} else if arm.function32bitOpcode&0xff80 == 0xfb80 {
		// "64-bit multiply, multiply-accumulate, and divide instructions"
		// page 3-25 of "Thumb-2 Supplement"
		op := (arm.function32bitOpcode & 0x0070) >> 4
		RdLo := (opcode & 0xf000) >> 12
		RdHi := Rd
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b010 && op2 == 0b0000 {
			// "4.6.207 UMULL" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "UMULL"

			result := uint64(arm.registers[Rn]) * uint64(arm.registers[Rm])
			arm.registers[RdHi] = uint32((result & 0xffffffff00000000) >> 32)
			arm.registers[RdLo] = uint32(result & 0x00000000ffffffff)
		} else if op == 0b011 && op2 == 0b1111 {
			// "4.6.198 UDIV" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "UDIV"

			// don't allow divide by zero
			if arm.registers[Rm] != 0 {
				arm.registers[Rd] = arm.registers[Rn] / arm.registers[Rm]
			} else {
				arm.registers[Rd] = 0
			}
		} else {
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (64bit multiplies) (%03b/%04b)", op, op2))
		}
	} else {
		panic("reserved data processing instructions, non-immediate")
	}
}

func (arm *ARM) thumb2LoadStoreDoubleEtc(opcode uint16) {
	// "3.3.4 Load/store double and exclusive, and table branch" of "Thumb-2 Supplement"

	p := (arm.function32bitOpcode & 0x0100) == 0x0100
	u := (arm.function32bitOpcode & 0x0080) == 0x0080
	w := (arm.function32bitOpcode & 0x0020) == 0x0020
	l := (arm.function32bitOpcode & 0x0010) == 0x0010

	Rn := arm.function32bitOpcode & 0x000f
	Rt := (opcode & 0xf000) >> 12
	Rt2 := (opcode & 0x0f00) >> 8
	imm8 := opcode & 0x00ff
	imm32 := imm8 << 2

	if p || w {
		// "Load and Store Double"
		addr := arm.registers[Rn]
		if p {
			if u {
				addr += uint32(imm32)
			} else {
				addr -= uint32(imm32)
			}
		}

		if l {
			// "A7.7.50 LDRD (immediate)" of "ARMv7-M"
			arm.fudge_thumb2disassemble32bit = "LDRD (immediate)"

			arm.registers[Rt] = arm.read32bit(addr)
			arm.registers[Rt2] = arm.read32bit(addr + 4)
		} else {
			// "A7.7.166 STRD (immediate)" of "ARMv7-M"
			arm.fudge_thumb2disassemble32bit = "STRD (immediate)"

			arm.write32bit(addr, arm.registers[Rt])
			arm.write32bit(addr+4, arm.registers[Rt])
		}

		if w {
			arm.registers[Rn] = addr
		}

	} else if arm.function32bitOpcode&0x0080 == 0x0080 {
		// "Load and Store Exclusive Byte Halfword, Doubleword, and Table Branch"

		op := (opcode & 0x00f0) >> 4

		switch op {
		case 0b0000:
			// "4.6.188 TBB" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "TBB"

			Rm := opcode & 0x000f
			idx := arm.registers[Rn] + arm.registers[Rm]
			if Rn == rPC || Rm == rPC {
				idx -= 2
			}
			halfwords := arm.read8bit(idx)
			arm.registers[rPC] += uint32(halfwords) << 1
		default:
			panic(fmt.Sprintf("unhandled load and store double and exclusive and table branch (load and store exclusive byte etc.) (%04b)", op))
		}
	} else {
		// "Load and Store Exclusive"
		panic("unhandled load and store double and exclusive and table branch (load and store exclusive)")
	}
}

func (arm *ARM) thumb2BranchesORDataProcessing(opcode uint16) {
	if opcode&0x8000 == 0x8000 {
		arm.thumb2BranchesMiscControl(opcode)
	} else {
		arm.thumb2DataProcessing(opcode)
	}
}

func (arm *ARM) thumb2DataProcessing(opcode uint16) {
	// "3.3.1 Data processing instructions: immediate, including bitfield and saturate" of "Thumb-2 Supplement"

	if arm.function32bitOpcode&0xfa00 == 0xf000 {
		// "Data processing instructions with modified 12-bit immediate"
		// page 3-14 of "Thumb-2 Supplement"

		i := (arm.function32bitOpcode & 0x0400) >> 10
		op := (arm.function32bitOpcode & 0x01e0) >> 5
		setFlags := (arm.function32bitOpcode & 0x0010) == 0x0010

		Rn := arm.function32bitOpcode & 0x000f

		imm3 := (opcode & 0x7000) >> 12
		Rd := (opcode & 0x0f00) >> 8
		imm8 := opcode & 0x00ff
		imm12 := (i << 11) | (imm3 << 8) | imm8
		imm32, carry := ThumbExpandImm_C(uint32(imm12), arm.Status.carry)

		switch op {
		case 0b0000:
			if Rd == 0b1111 {
				// "4.6.192 TST (immediate)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "TST"

				result := arm.registers[Rn] & imm32
				if setFlags {
					arm.Status.isNegative(result)
					arm.Status.isZero(result)
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			} else {
				// "4.6.8 AND (immediate)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "AND"

				arm.registers[Rd] = arm.registers[Rn] & imm32
				if setFlags {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			}
		case 0b0010:
			if Rn == 0xf {
				// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
				// T2 encoding
				arm.fudge_thumb2disassemble32bit = "MOV (immediate)"

				arm.registers[Rd] = imm32

				if setFlags {
					arm.Status.isNegative(arm.registers[Rn])
					arm.Status.isZero(arm.registers[Rn])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			} else {
				// "4.6.91 ORR (immediate)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "ORR (immediate)"

				arm.registers[Rd] = arm.registers[Rn] | imm32

				if setFlags {
					arm.Status.isNegative(arm.registers[Rn])
					arm.Status.isZero(arm.registers[Rn])
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0100:
			// "4.6.36 EOR (immediate)" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "EOR (immediate)"

			arm.registers[Rd] = arm.registers[Rn] ^ imm32
			if setFlags {
				arm.Status.isNegative(arm.registers[Rd])
				arm.Status.isZero(arm.registers[Rd])
				arm.Status.setCarry(carry)
				// overflow unchanged
			}

		case 0b1000:
			if arm.function32bitOpcode&0x100 == 0x100 {
				// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				arm.fudge_thumb2disassemble32bit = "ADD (immediate)"

				if Rn == 0x000f {
					panic("Rn register cannot be 0b1111 for ADD immediate")
				}

				src := arm.registers[Rn]
				arm.registers[Rd] = src + imm32

				if setFlags {
					arm.Status.isNegative(arm.registers[Rn])
					arm.Status.isZero(arm.registers[Rn])
					if arm.Status.carry {
						arm.Status.isCarry(src, imm32, 0)
						arm.Status.isOverflow(src, imm32, 0)
					} else {
						arm.Status.isCarry(src, imm32, 1)
						arm.Status.isOverflow(src, imm32, 1)
					}
				}
			} else {
				// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				panic("unimplemented 'ADD (immediate)' T4 encoding")
			}

		case 0b1101:
			if Rd == 0b1111 {
				// "4.6.29 CMP (immediate)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "CMP (immediate)"

				result, carry, overflow := AddWithCarry(arm.registers[Rn], ^imm32, 1)
				arm.Status.isNegative(result)
				arm.Status.isZero(result)
				arm.Status.setCarry(carry)
				arm.Status.setOverflow(overflow)
			} else {
				// "A7.7.174 SUB (immediate)" of "ARMv7-M"
				// T3 encoding
				arm.fudge_thumb2disassemble32bit = "SUB (immediate)"

				result, carry, _ := AddWithCarry(arm.registers[Rn], ^imm32, 1)
				arm.registers[Rd] = result
				if setFlags {
					arm.Status.isNegative(result)
					arm.Status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b1110:
			// "4.6.118 RSB (immediate)" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "RSB"

			result, carry, overflow := AddWithCarry(^arm.registers[Rn], imm32, 1)
			arm.registers[Rd] = result
			if setFlags {
				arm.Status.isNegative(arm.registers[Rd])
				arm.Status.isZero(arm.registers[Rd])
				arm.Status.setCarry(carry)
				arm.Status.setOverflow(overflow)
			}

		default:
			panic(fmt.Sprintf("unimplemented 'data processing instructions with modified 12bit immediate' (%04b)", op))
		}
	} else if arm.function32bitOpcode&0xfb40 == 0xf200 {
		// "Data processing instructions with plain 12-bit immediate"
		// page 3-15 of "Thumb-2 Supplement"
	} else if arm.function32bitOpcode&0xfb40 == 0xf240 {
		// "Data processing instructions with plain 16-bit immediate"
		// page 3-15 of "Thumb-2 Supplement"

		op := (arm.function32bitOpcode & 0x0080) >> 7
		op2 := (arm.function32bitOpcode & 0x0030) >> 4

		if op == 0b0 && op2 == 0b00 {
			// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
			// T3 encoding
			arm.fudge_thumb2disassemble32bit = "MOV"

			i := (arm.function32bitOpcode & 0x0400) >> 10
			imm4 := arm.function32bitOpcode & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm8 := opcode & 0x00ff

			imm32 := uint32((imm4 << 12) | (i << 11) | (imm3 << 8) | imm8)
			arm.registers[Rd] = imm32
		} else {
			panic("unimplemented MOVT")
		}

	} else if arm.function32bitOpcode&0xfb10 == 0xf300 {
		// "Data processing instructions, bitfield and saturate"
		// page 3-16 of "Thumb-2 Supplement"

		op := (arm.function32bitOpcode & 0x00e0) >> 5
		switch op {
		case 0b0110:
			// "4.6.197 UBFX" of "Thumb-2 Supplement"
			arm.fudge_thumb2disassemble32bit = "UBFX"

			Rn := arm.function32bitOpcode & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm2 := (opcode & 0x00c0) >> 6
			widthm1 := opcode & 0x001f

			lsbit := (imm3 << 2) | imm2
			msbit := lsbit + widthm1
			width := widthm1 + 1
			if msbit <= 31 {
				arm.registers[Rd] = (arm.registers[Rn] >> uint32(lsbit)) & ((1 << width) - 1)
			}
		default:
			panic(fmt.Sprintf("unimplemented 'bitfield operation' (%03b)", op))
		}
	} else {
		panic("reserved data processing instructions: immediate, including bitfield and saturate")
	}
}

func (arm *ARM) thumb2LoadStoreSingle(opcode uint16) {
	// "3.3.3 Load and store single data item, and memory hints" of "Thumb-2 Supplement"

	// Addressing mode discussed in "A4.6.5 Addressing modes" of "ARMv7-M"

	size := (arm.function32bitOpcode & 0x0060) >> 5
	s := arm.function32bitOpcode&0x0100 == 0x0100
	l := arm.function32bitOpcode&0x0010 == 0x0010
	Rn := arm.function32bitOpcode & 0x000f
	Rt := (opcode & 0xf000) >> 12

	if Rt == rPC {
		panic("PLD and PLI not thought about yet")
	}

	if s {
		panic("unhandled sign extend for 'load and store single data item, and memory hints'")
	}

	if arm.function32bitOpcode&0xfe1f == 0xf81f {
		// PC +/ imm12 (format 1 in the table)
		// further depends on size. l is always true

		u := arm.function32bitOpcode&0x0080 == 0x0080
		imm12 := opcode & 0x0fff
		addr := arm.registers[Rn] & 0xfffffffc
		if u {
			addr += uint32(imm12)
		} else {
			addr -= uint32(imm12)
		}

		switch size {
		case 0b00:
			arm.fudge_thumb2disassemble32bit = "LDRB (literal PC relative)"
			arm.registers[Rt] = uint32(arm.read8bit(addr))
			if s && arm.registers[Rt]&0x80 == 0x80 {
				arm.registers[Rt] |= 0xffffff00
			}
		case 0b01:
			arm.fudge_thumb2disassemble32bit = "LDRH (literal PC relative)"
			arm.registers[Rt] = uint32(arm.read16bit(addr))
			if s && arm.registers[Rt]&0x8000 == 0x8000 {
				arm.registers[Rt] |= 0xffff0000
			}
		case 0b10:
			arm.fudge_thumb2disassemble32bit = "LDR (literal PC relative)"
			arm.registers[Rt] = arm.read32bit(addr)
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'PC +/- imm12'", size))
		}
	} else if arm.function32bitOpcode&0xfe80 == 0xf880 {
		// Rn + imm12 (format 2 in the table)
		// further depends on size and L bit

		// U is always up for this format meaning that we add the index to
		// the base address
		imm12 := opcode & 0x0fff
		addr := arm.registers[Rn] + uint32(imm12)

		switch size {
		case 0b00:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(1)"
				arm.registers[Rt] = uint32(arm.read8bit(addr))
				if s && arm.registers[Rt]&0x80 == 0x80 {
					arm.registers[Rt] |= 0xffffff00
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write8bit(addr, uint8(arm.registers[Rt]))
			}
		case 0b01:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(2)"
				arm.registers[Rt] = uint32(arm.read16bit(addr))
				if s && arm.registers[Rt]&0x8000 == 0x8000 {
					arm.registers[Rt] |= 0xffff0000
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write16bit(addr, uint16(arm.registers[Rt]))
			}
		case 0b10:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(3)"
				arm.registers[Rt] = arm.read32bit(addr)
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write32bit(addr, arm.registers[Rt])
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + imm12'", size))
		}

	} else if (opcode & 0x0f00) == 0x0c00 {
		// Rn -imm8 (format 3 in the table)
		// imm8 := opcode & 0x00ff
		panic("umimplemented Rn -imm8")

	} else if (opcode & 0x0f00) == 0x0e00 {
		// Rn +imm8, user privilege (format 4 in the table)
		// imm8 := opcode & 0x00ff
		panic("umimplemented Rn +imm8, user privilege")

	} else if (opcode & 0x0d00) == 0x0900 {
		// Rn post-index by +/- imm8 (format 5 in the table)
		imm8 := opcode & 0x00ff
		addr := arm.registers[Rn]

		switch size {
		case 0b00:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(4)"
				arm.registers[Rt] = uint32(arm.read8bit(addr))
				if s && arm.registers[Rt]&0x80 == 0x80 {
					arm.registers[Rt] |= 0xffffff00
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write8bit(addr, uint8(arm.registers[Rt]))
			}
		case 0b01:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(5)"
				arm.registers[Rt] = uint32(arm.read16bit(addr))
				if s && arm.registers[Rt]&0x8000 == 0x8000 {
					arm.registers[Rt] |= 0xffff0000
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write16bit(addr, uint16(arm.registers[Rt]))
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + imm12'", size))
		}

		arm.registers[Rn] = addr + uint32(imm8)

	} else if (opcode & 0x0d00) == 0x0d00 {
		// Rn pre-indexed by +/- imm8 (format 6 in the table)
		imm8 := opcode & 0x00ff
		addr := arm.registers[Rn] + uint32(imm8)

		switch size {
		case 0b00:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(6)"
				arm.registers[Rt] = uint32(arm.read8bit(addr))
				if s && arm.registers[Rt]&0x80 == 0x80 {
					arm.registers[Rt] |= 0xffffff00
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write8bit(addr, uint8(arm.registers[Rt]))
			}
		case 0b01:
			if l {
				arm.fudge_thumb2disassemble32bit = "LDR(7)"
				arm.registers[Rt] = uint32(arm.read16bit(addr))
				if s && arm.registers[Rt]&0x8000 == 0x8000 {
					arm.registers[Rt] |= 0xffff0000
				}
			} else {
				arm.fudge_thumb2disassemble32bit = "STR"
				arm.write16bit(addr, uint16(arm.registers[Rt]))
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + imm12'", size))
		}

		arm.registers[Rn] = addr

	} else if (opcode & 0x0fc0) == 0x0000 {
		// Rn + shifted register (format 7 in the table)
		shift := (opcode & 0x0030) >> 4
		Rm := opcode & 0x0007

		addr := arm.registers[Rn] + (arm.registers[Rm] << shift)

		if l {
			switch size {
			case 0b00:
				// "A7.7.48 LDRB (register)" of "Thumb-2 Supplement"
				arm.fudge_thumb2disassemble32bit = "LDRB"

				arm.registers[Rt] = uint32(arm.read8bit(addr))
			default:
				panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + shifted register' (load)", size))
			}
		} else {
			panic("unhandled save 'Rn + shifted register'")
		}

	} else {
		panic("unhandled bit pattern in 'load and store single data item, and memory hints'")
	}
}

func (arm *ARM) thumb2LoadStoreMultiple(opcode uint16) {
	// "3.3.5 Load and store multiple, RFE, and SRS" of "Thumb-2 Supplement"
	//		and
	// "A5.3.5 Load Multiple and Store Multiple" of "ARMv7-M"

	op := (arm.function32bitOpcode & 0x0180) >> 7
	l := (arm.function32bitOpcode & 0x0010) >> 4
	w := (arm.function32bitOpcode & 0x0020) >> 5
	Rn := arm.function32bitOpcode & 0x000f
	WRn := Rn | (w << 4)

	switch op {
	case 0b00:
		panic("load and store multiple: illegal op (0b00)")
	case 0b01:
		if l == 0x1 {
			switch WRn {
			case 0b11101:
				// "A7.7.99 POP" of "ARMv7-M"
				arm.fudge_thumb2disassemble32bit = "POP (ldmia)"

				regList := opcode & 0xdfff
				addr := arm.registers[rSP]
				arm.registers[rSP] += uint32(bits.OnesCount16(regList) * 4)

				// read each register in turn (from lower to highest)
				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// read register if indicated by regList
					if regList&m == m {
						arm.registers[i] = arm.read32bit(addr)
						addr += 4
					}
				}

				// write PC
				if regList&0x8000 == 0x8000 {
					arm.registers[rPC] = arm.read32bit(addr)
				}
			default:
				panic(fmt.Sprintf("load and store multiple: unimplemented op (%02b) l (%01b)", op, l))
			}
		} else {
			panic(fmt.Sprintf("load and store multiple: unimplemented op (%02b) l (%01b)", op, l))
		}
	case 0b10:
		if l == 0x1 {
			panic(fmt.Sprintf("load and store multiple: unimplemented op (%02b) l (%01b)", op, l))
		} else {
			switch WRn {
			case 0b11101:
				// "A7.7.101 PUSH" of "ARMv7-M"
				arm.fudge_thumb2disassemble32bit = "PUSH (stmdb)"

				regList := opcode & 0x5fff
				c := (uint32(bits.OnesCount16(regList))) * 4
				addr := arm.registers[rSP] - c

				// store each register in turn (from lowest to highest)
				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// write register if indicated by regList
					if regList&m == m {
						arm.write32bit(addr, arm.registers[i])
						addr += 4
					}
				}

				arm.registers[rSP] -= c
			default:
				panic(fmt.Sprintf("load and store multiple: unimplemented op (%02b) l (%01b)", op, l))
			}
		}
	case 0b11:
		panic("load and store multiple: illegal op (11)")
	}
}

func (arm *ARM) thumb2BranchesMiscControl(opcode uint16) {
	// "3.3.6 Branches, miscellaneous control instructions" of "Thumb-2 Supplement"

	if opcode&0xd000 == 0xd000 {
		arm.thumb2LongBranchWithLink(opcode)
	} else if opcode&0xd000 == 0x8000 {
		// "4.6.12 B" of "Thumb-2 Supplement"
		// T3 encoding
		arm.fudge_thumb2disassemble32bit = "B (cond)"

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.function32bitOpcode & 0x0400) >> 10)
		cond := (arm.function32bitOpcode & 0x03c0) >> 6
		imm6 := uint32(arm.function32bitOpcode & 0x003f)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x0800) >> 11)
		imm11 := uint32(opcode & 0x07ff)

		imm32 := (s << 20) | (j2 << 19) | (j1 << 18) | (imm6 << 12) | (imm11 << 1)

		if s == 0x01 {
			imm32 |= 0xfff00000
		}

		if arm.Status.condition(uint8(cond)) {
			arm.registers[rPC] += imm32
		}
	} else {
		panic("unimplemented branches, miscellaneous control instructions")
	}
}

func (arm *ARM) thumb2LongBranchWithLink(opcode uint16) {
	// details in "A7.7.18 BL" of "ARMv7-M"
	arm.fudge_thumb2disassemble32bit = "BL"

	arm.registers[rLR] = (arm.registers[rPC]-2)&0xfffffffe | 0x00000001

	// make sure we're working with 32bit immediate numbers so that we don't
	// drop bits when shifting
	s := uint32((arm.function32bitOpcode & 0x400) >> 10)
	j1 := uint32((opcode & 0x2000) >> 13)
	j2 := uint32((opcode & 0x800) >> 11)
	i1 := (^(j1 ^ s)) & 0x01
	i2 := (^(j2 ^ s)) & 0x01
	imm10 := uint32(arm.function32bitOpcode & 0x3ff)
	imm11 := uint32(opcode & 0x7ff)
	imm32 := (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)
	imm32 = imm32 | (s << 24) | (s << 25) | (s << 26) | (s << 27) | (s << 28) | (s << 29) | (s << 30) | (s << 31)
	arm.registers[rPC] += imm32
}
