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

func is32BitThumb2(opcode uint16) bool {
	return opcode&0xf800 == 0xe800 || opcode&0xf000 == 0xf000
}

func (arm *ARM) decode32bitThumb2(opcode uint16) decodeFunction {
	// Two tables for top level decoding of 32bit Thumb-2 instructions.
	//
	// "3.3 Instruction encoding for 32-bit Thumb instructions" of "Thumb-2 Supplement"
	//		and
	// "A5.3 32-bit Thumb instruction encoding" of "ARMv7-M"
	//
	// Both with different emphasis but the table in the "Thumb-2 Supplement"
	// was used.

	if opcode&0xec00 == 0xec00 {
		// coprocessor
		return arm.decodeThumb2Coprocessor
	} else if opcode&0xf800 == 0xf000 {
		// branches, miscellaneous control
		//  OR
		// data processing: immediate, including bitfield and saturate
		return arm.decode32bitThumb2BranchesORDataProcessing
	} else if opcode&0xfe40 == 0xe800 {
		// load and store multiple, RFE and SRS
		return arm.decode32bitThumb2LoadStoreMultiple
	} else if opcode&0xfe40 == 0xe840 {
		// load and store double and exclusive and table branch
		return arm.decode32bitThumb2LoadStoreDoubleEtc
	} else if opcode&0xfe00 == 0xf800 {
		// load and store single data item, memory hints
		return arm.decode32bitThumb2LoadStoreSingle
	} else if opcode&0xee00 == 0xea00 {
		// data processing, no immediate operand
		return arm.decode32bitThumb2DataProcessingNonImmediate
	}

	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (%04x)", opcode))
}

func (arm *ARM) decode32bitThumb2DataProcessingNonImmediate(opcode uint16) *DisasmEntry {
	// "3.3.2 Data processing instructions, non-immediate" of "Thumb-2 Supplement"

	Rn := arm.state.function32bitOpcodeHi & 0x000f
	Rm := opcode & 0x000f
	Rd := (opcode & 0x0f00) >> 8

	if arm.state.function32bitOpcodeHi&0xfe00 == 0xea00 {
		// "Data processing instructions with constant shift"
		// page 3-18 of "Thumb-2 Supplement"
		op := (arm.state.function32bitOpcodeHi & 0x01e0) >> 5
		setFlags := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
		// sbz := (opcode & 0x8000) >> 15
		imm3 := (opcode & 0x7000) >> 12
		imm2 := (opcode & 0x00c0) >> 6
		typ := (opcode & 0x0030) >> 4
		imm5 := (imm3 << 2) | imm2

		switch op {
		case 0b0000:
			// "4.6.193 TST (register)" of "Thumb-2 Supplement"
			// and
			// "4.6.9 AND (register)" of "Thumb-2 Supplement"

			// whether this is a TST instruction or not
			tst := Rd == rPC && setFlags

			// disassembly only
			if arm.decodeOnly {
				if tst {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TST",
					}
				} else {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "AND",
					}
				}
			}

			var shifted uint32
			var carry bool

			switch typ {
			case 0b00:
				// with logical left shift

				// carry bit
				m := uint32(0x01) << (32 - imm5)
				carry = arm.state.registers[Rm]&m == m

				// perform shift
				shifted = arm.state.registers[Rm] << imm5
			case 0b01:
				// with logical right shift

				// carry bit
				m := uint32(0x01) << (imm5 - 1)
				carry = arm.state.registers[Rm]&m == m

				// perform shift
				shifted = (arm.state.registers[Rm] >> imm5)
			case 0b10:
				// with arithmetic right shift

				// carry bit
				m := uint32(0x01) << (imm5 - 1)
				carry = arm.state.registers[Rm]&m == m

				// perform shift (with sign extension)
				signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
				shifted = arm.state.registers[Rm] >> imm5
				if signExtend == 0x01 {
					shifted |= ^uint32(0) << (32 - imm5)
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
			}

			// perform AND operation
			result := arm.state.registers[Rn] & shifted

			// store result in register if this is the AND instruction
			if !tst {
				arm.state.registers[Rd] = result
			}

			// change status register
			if setFlags {
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				// overflow unchanged
			}

		case 0b0001:
			// "4.6.16 BIC (register)" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "BIC",
				}
			}

			switch typ {
			case 0b00:
				// with logical left shift

				// carry bit
				m := uint32(0x01) << (32 - imm5)
				carry := arm.state.registers[Rm]&m == m

				// perform shift
				shifted := arm.state.registers[Rm] << imm5

				// perform bit clear
				arm.state.registers[Rd] = arm.state.registers[Rn] & ^shifted

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

			case 0b01:
				// with logical right shift

				// carry bit
				m := uint32(0x01) << (32 - imm5)
				carry := arm.state.registers[Rm]&m == m

				// perform shift
				shifted := arm.state.registers[Rm] >> imm5

				// perform bit clear
				arm.state.registers[Rd] = arm.state.registers[Rn] & ^shifted

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			case 0b10:
				// with arithmetic right shift

				// carry bit
				m := uint32(0x01) << (imm5 - 1)
				carry := arm.state.registers[Rm]&m == m

				// perform shift (with sign extension)
				signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
				shifted := arm.state.registers[Rm] >> imm5
				if signExtend == 0x01 {
					shifted |= ^uint32(0) << (32 - imm5)
				}

				// perform bit clear
				arm.state.registers[Rd] = arm.state.registers[Rn] & ^shifted

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			default:
				panic(fmt.Sprintf("unhandled shift type (%02b) for BIC instruction ", typ))
			}

		case 0b0010:
			// "Move, and immediate shift instructions"
			// page 3-19 of "Thumb-2 Supplement"

			if Rn == rPC {
				switch typ {
				case 0b00:
					if imm5 == 0b00000 {
						// "4.6.77 MOV (register)" of "Thumb-2 Supplement"
						// T3 encoding
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "MOV",
								Operand:  "register",
							}
						}

						arm.state.registers[Rd] = arm.state.registers[Rm]
						if setFlags {
							arm.state.status.isNegative(arm.state.registers[Rd])
							arm.state.status.isZero(arm.state.registers[Rd])

							// carry unchanged. there is a mistake in the
							// Thumb-2 Supplement but it is clear from the
							// ARMv7-M that carry is not affected by this
							// instruction

							// overflow unchanged
						}
					} else {
						// "4.6.68 LSL (immediate)" of "Thumb-2 Supplement"
						// T2 encoding
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LSL",
								Operand:  "immediate",
							}
						}

						m := uint32(0x01) << (32 - imm5)
						carry := arm.state.registers[Rm]&m == m

						arm.state.registers[Rd] = arm.state.registers[Rm] << imm5
						if setFlags {
							arm.state.status.isNegative(arm.state.registers[Rd])
							arm.state.status.isZero(arm.state.registers[Rd])
							arm.state.status.setCarry(carry)
							// overflow unchanged
						}
					}
				case 0b01:
					// "4.6.70 LSR (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LSR",
							Operand:  "immediate",
						}
					}

					m := uint32(0x01) << (imm5 - 1)
					carry := arm.state.registers[Rm]&m == m

					arm.state.registers[Rd] = arm.state.registers[Rm] >> imm5
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}
				case 0b10:
					// "4.6.10 ASR (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "ASR",
							Operand:  "immediate",
						}
					}

					// whether to set carry bit
					m := uint32(0x01) << (imm5 - 1)
					carry := arm.state.registers[Rm]&m == m

					// extend sign, check for bit
					signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31

					// perform actual shift
					arm.state.registers[Rd] = arm.state.registers[Rm] >> imm5

					// perform sign extension
					if signExtend == 0x01 {
						arm.state.registers[Rd] |= ^uint32(0) << (32 - imm5)
					}

					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}
				case 0b11:
					if imm5 == 0b00000 {
						// 4.6.117 RRX Rotate Right
						// T1 encoding
						panic("unimplemented RRX")
					} else {
						// 4.6.115 ROR (immediate)
						// T1 encoding
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "ROR",
								Operand:  "immediate",
							}
						}

						result, carry := ROR_C(arm.state.registers[Rm], uint32(imm5))
						arm.state.registers[Rd] = result
						if setFlags {
							arm.state.status.isNegative(arm.state.registers[Rd])
							arm.state.status.isZero(arm.state.registers[Rd])
							arm.state.status.setCarry(carry)
							// overflow unchanged
						}
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}
			} else {
				// "4.6.92 ORR (register)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ORR",
						Operand:  "register",
					}
				}

				var carry bool
				var result uint32

				switch typ {
				case 0b00:
					// with logical left shift
					m := uint32(0x01) << (32 - imm5)
					carry = arm.state.registers[Rm]&m == m
					result = arm.state.registers[Rn] | (arm.state.registers[Rm] << imm5)
				case 0b01:
					// with logical right shift
					m := uint32(0x01) << (imm5 - 1)
					carry = arm.state.registers[Rm]&m == m
					result = arm.state.registers[Rn] | (arm.state.registers[Rm] >> imm5)
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}

				arm.state.registers[Rd] = result

				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0011:
			// two very similar instructions for this op
			//
			// "4.6.86 MVN (register)" of "Thumb-2 Supplement"
			// T2 encoding
			//
			// and
			//
			// "4.6.90 ORN (register)" of "Thumb-2 Supplement"
			// T1 encoding

			if Rn == rPC {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MVN",
						Operand:  "register",
					}
				}

				var carry bool
				var result uint32

				switch typ {
				case 0b00:
					// with logical left shift
					m := uint32(0x01) << (32 - imm5)
					carry = arm.state.registers[Rm]&m == m
					result = (arm.state.registers[Rm] << imm5)
				case 0b01:
					// with logical right shift
					m := uint32(0x01) << (imm5 - 1)
					carry = arm.state.registers[Rm]&m == m
					result = (arm.state.registers[Rm] >> imm5)
				case 0b10:
					// with arithmetic right shift
					signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
					result = arm.state.registers[Rm] >> imm5
					if signExtend == 0x01 {
						result |= ^uint32(0) << (32 - imm5)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}

				result = ^result
				arm.state.registers[Rd] = result

				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			} else {
				panic("ORN (register)")
			}

		case 0b0100:
			// two very similar instructions for this op
			//
			// "4.6.191 TEQ (register)" of "Thumb-2 Supplement"
			// T1 encoding
			//
			// and
			//
			// "4.6.37 EOR (register)" of "Thumb-2 Supplement
			// T2 encoding

			var carry bool
			var result uint32

			switch typ {
			case 0b00:
				// with logical left shift
				m := uint32(0x01) << (32 - imm5)
				carry = arm.state.registers[Rm]&m == m
				result = arm.state.registers[Rn] ^ (arm.state.registers[Rm] << imm5)
			case 0b01:
				// with logical right shift
				m := uint32(0x01) << (imm5 - 1)
				carry = arm.state.registers[Rm]&m == m
				result = arm.state.registers[Rn] ^ (arm.state.registers[Rm] >> imm5)
			case 0b11:
				// with arithmetic right shift
				signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
				result = arm.state.registers[Rn] ^ (arm.state.registers[Rm] >> imm5)
				if signExtend == 0x01 {
					result |= ^uint32(0) << (32 - imm5)
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
			}

			if Rd == rPC && setFlags {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TEQ",
						Operand:  "register",
					}
				}
			} else {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "EOR",
						Operand:  "register",
					}
				}
				arm.state.registers[Rd] = result
			}

			if setFlags {
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				// overflow unchanged
			}

		case 0b1000:
			if Rd == rPC {
				// "4.6.28 CMN (register)"
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) CMN", op, typ))
			} else {
				// "4.6.4 ADD (register)"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ADD",
						Operand:  "register",
					}
				}

				switch typ {
				case 0b00:
					// with logical left shift
					shifted := arm.state.registers[Rm] << imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, 0)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				case 0b01:
					// with logical right shift
					shifted := arm.state.registers[Rm] >> imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, 0)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				case 0b10:
					// with arithmetic right shift
					signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
					shifted := arm.state.registers[Rm] >> imm5
					if signExtend == 0x01 {
						shifted |= ^uint32(0) << (32 - imm5)
					}
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, 0)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) ADD", op, typ))
				}
			}

		case 0b1010:
			// "4.6.2 ADC (register)" of "Thumb-2 Supplement")
			// T2 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "ADC",
					Operand:  "register",
				}
			}

			switch typ {
			case 0b00:
				// with logical left shift
				shifted := arm.state.registers[Rm] << imm5

				// carry value taken from carry bit in status register
				var c uint32
				if arm.state.status.carry {
					c = 1
				}

				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, c)
				arm.state.registers[Rd] = result
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) ADC", op, typ))
			}

		case 0b1101:
			if Rd == rPC {
				// "4.6.30 CMP (register)" of "Thumb-2 Supplement"
				// T3 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "CMP",
					}
				}

				switch typ {
				case 0b00:
					// with logical left shift
					shifted := arm.state.registers[Rm] << imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				case 0b01:
					// with logical right shift
					shifted := arm.state.registers[Rm] >> imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) CMP", op, typ))
				}

			} else {
				// "4.6.177 SUB (register)" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SUB",
						Operand:  "register",
					}
				}

				switch typ {
				case 0b00:
					// with logical left shift
					shifted := arm.state.registers[Rm] << imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				case 0b01:
					// with logical right shift
					shifted := arm.state.registers[Rm] >> imm5
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				case 0b10:
					// with arithmetic right shift
					signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
					shifted := arm.state.registers[Rm] >> imm5
					if signExtend == 0x01 {
						shifted |= ^uint32(0) << (32 - imm5)
					}
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					arm.state.registers[Rd] = result
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				default:
					panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) SUB", op, typ))
				}
			}

		case 0b1110:
			// "4.6.119 RSB (register)" of "Thumb-2 Supplement"
			// T1 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "RSB",
					Operand:  "register",
				}
			}

			switch typ {
			case 0b00:
				// with logical left shift
				shifted := arm.state.registers[Rm] << imm5
				result, carry, overflow := AddWithCarry(^arm.state.registers[Rn], shifted, 1)
				arm.state.registers[Rd] = result
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

			case 0b01:
				// with logical right shift
				shifted := arm.state.registers[Rm] << imm5
				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, 1)
				arm.state.registers[Rd] = result
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}
			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b) RSB", op, typ))
			}

		default:
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (data processing, constant shift) (%04b)", op))
		}
	} else if arm.state.function32bitOpcodeHi&0xff80 == 0xfa00 {
		if opcode&0x0080 == 0x0000 {
			// "Register-controlled shift instructions"
			// page 3-19 of "Thumb-2 Supplement"

			op := (arm.state.function32bitOpcodeHi & 0x0060) >> 5
			setFlags := (arm.state.function32bitOpcodeHi & 0x0010) == 0x0010
			shift := arm.state.registers[Rm] & 0x00ff

			switch op {
			case 0b00:
				// "4.6.69 LSL (register)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LSL",
						Operand:  "register",
					}
				}

				// whether to set carry bit
				m := uint32(0x01) << (32 - shift)
				carry := arm.state.registers[Rn]&m == m

				// perform actual shift
				arm.state.registers[Rd] = arm.state.registers[Rn] << shift

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

			case 0b01:
				// "4.6.71 LSR (register)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LSR",
						Operand:  "register",
					}
				}

				// whether to set carry bit
				m := uint32(0x01) << (shift - 1)
				carry := arm.state.registers[Rn]&m == m

				// perform actual shift
				arm.state.registers[Rd] = arm.state.registers[Rn] >> shift

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			case 0b10:
				// "4.6.11 ASR (register)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ASR",
						Operand:  "register",
					}
				}

				// whether to set carry bit
				m := uint32(0x01) << (shift - 1)
				carry := arm.state.registers[Rn]&m == m

				// extend sign, check for bit
				signExtend := (arm.state.registers[Rn] & 0x80000000) >> 31

				// perform actual shift
				arm.state.registers[Rd] = arm.state.registers[Rn] >> shift

				// perform sign extension
				if signExtend == 0x01 {
					arm.state.registers[Rd] |= ^uint32(0) << (32 - shift)
				}

				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (reg controlled shift) (%02b)", op))
			}
		} else {
			// "Signed and unsigned extend instructions with optional addition"
			// page 3-20 of "Thumb-2 Supplement"
			op := (arm.state.function32bitOpcodeHi & 0x0070) >> 4
			rot := (opcode & 0x0030) >> 4

			switch op {
			case 0b000:
				if Rn == rPC {
					// "4.6.187 SXTH" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "SXTH",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = v & 0x0000ffff
					if arm.state.registers[Rd]&0x8000 == 0x8000 {
						arm.state.registers[Rd] |= 0xffff0000
					}
				} else {
					// "4.6.184 SXTAH" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "SXTAH",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x0000ffff)
					if arm.state.registers[Rd]&0x8000 == 0x8000 {
						arm.state.registers[Rd] |= 0xffff0000
					}
				}

			case 0b001:
				if Rn == rPC {
					// "4.6.226 UXTH" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "UXTH",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = v & 0x0000ffff
				} else {
					// "4.6.223 UXTAH" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "UXTAH",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x0000ffff)
				}
			case 0b101:
				if Rn == rPC {
					// "4.6.224 UXTB" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "UXTB",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = v & 0x000000ff
				} else {
					// "4.6.221 UXTAB" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "UXTAB",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x000000ff)
				}

			case 0b100:
				if Rn == 0b1111 {
					// "4.6.185 SXTB" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "SXTB",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = v & 0x000000ff
					if arm.state.registers[Rd]&0x80 == 0x80 {
						arm.state.registers[Rd] |= 0xffffff00
					}
				} else {
					// "4.6.182 SXTAB" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "SXTAB",
						}
					}

					v, _ := ROR_C(arm.state.registers[Rm], uint32(rot)<<3)
					arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x000000ff)
					if arm.state.registers[Rd]&0x80 == 0x80 {
						arm.state.registers[Rd] |= 0xffffff00
					}
				}

			default:
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (sign or zero extension with opt addition) (%03b)", op))
			}
		}
	} else if arm.state.function32bitOpcodeHi&0xff80 == 0xfa80 {
		if opcode&0x0080 == 0x0000 {
			// "SIMD add and subtract"
			// page 3-21 of "Thumb-2 Supplement"
			panic(fmt.Sprintf("unhandled SIMD add and subtract"))
		} else {
			// "Other three-register data processing instructions"
			// page 3-23 of "Thumb-2 Supplement"
			op := (arm.state.function32bitOpcodeHi & 0x70) >> 4
			op2 := (opcode & 0x0070) >> 4
			if op == 0b011 && op2 == 0b000 {
				// "4.6.26 CLZ" of "Thumb-2 Supplement"
				// T1 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "CLZ",
					}
				}
				c := bits.LeadingZeros32(arm.state.registers[Rm])
				arm.state.registers[Rd] = uint32(c)
			} else {
				panic(fmt.Sprintf("unhandled 'three-register data processing instruction'"))
			}
		}
	} else if arm.state.function32bitOpcodeHi&0xff80 == 0xfb00 {
		// "32-bit multiplies and sum of absolute differences, with or without accumulate"
		// page 3-24 of "Thumb-2 Supplement"
		op := (arm.state.function32bitOpcodeHi & 0x0070) >> 4
		Ra := (opcode & 0xf000) >> 12
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b000 && op2 == 0b0000 {
			if Ra == rPC {
				// "4.6.84 MUL" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MUL",
					}
				}

				// multiplication can be done on signed or unsigned value with
				// not change in functionality
				arm.state.registers[Rd] = arm.state.registers[Rn] * arm.state.registers[Rm]
			} else {
				// "4.6.74 MLA" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MLA",
					}
				}

				result := int(arm.state.registers[Rn]) * int(arm.state.registers[Rm])
				result += int(arm.state.registers[Ra])
				arm.state.registers[Rd] = uint32(result)
			}
		} else if op == 0b000 && op2 == 0b0001 {
			// "4.6.75 MLS" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "MLS",
				}
			}

			arm.state.registers[Rd] = uint32(int32(arm.state.registers[Ra]) - int32(arm.state.registers[Rn])*int32(arm.state.registers[Rm]))
		} else if op == 0b001 && op2 == 0b0000 {
			if Ra == 0b1111 {
				// "4.6.149 SMULBB, SMULBT, SMULTB, SMULTT" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SMUL",
						Operand:  "group of instructions",
					}
				}

				nHigh := opcode&0x0020 == 0x0020
				mHigh := opcode&0x0010 == 0x0010

				var operand1 uint16
				if nHigh {
					operand1 = uint16(arm.state.registers[Rn] >> 16)
				} else {
					operand1 = uint16(arm.state.registers[Rn])
				}

				var operand2 uint16
				if mHigh {
					operand2 = uint16(arm.state.registers[Rm] >> 16)
				} else {
					operand2 = uint16(arm.state.registers[Rm])
				}

				arm.state.registers[Rd] = uint32(int32(operand1) * int32(operand2))
			} else {
				panic(fmt.Sprintf("unhandled data processing instructions, non immediate (32bit multiplies) (%03b/%04b) Ra=%04b", op, op2, Ra))
			}
		} else {
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (32bit multiplies) (%03b/%04b)", op, op2))
		}
	} else if arm.state.function32bitOpcodeHi&0xff80 == 0xfb80 {
		// "64-bit multiply, multiply-accumulate, and divide instructions"
		// page 3-25 of "Thumb-2 Supplement"
		op := (arm.state.function32bitOpcodeHi & 0x0070) >> 4
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b010 && op2 == 0b0000 {
			// "4.6.207 UMULL" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "UMULL",
				}
			}

			RdLo := (opcode & 0xf000) >> 12
			RdHi := Rd

			result := uint64(arm.state.registers[Rn]) * uint64(arm.state.registers[Rm])
			arm.state.registers[RdHi] = uint32(result >> 32)
			arm.state.registers[RdLo] = uint32(result)
		} else if op == 0b011 && op2 == 0b1111 {
			// "4.6.198 UDIV" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "UDIV",
				}
			}

			// don't allow divide by zero
			if arm.state.registers[Rm] == 0 {
				arm.state.registers[Rd] = 0
			} else {
				arm.state.registers[Rd] = arm.state.registers[Rn] / arm.state.registers[Rm]
			}
		} else if op == 0b001 && op2 == 0b1111 {
			// "4.6.126 SDIV" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "SDIV",
				}
			}

			if arm.state.registers[Rm] == 0 {
				// don't allow divide by zero
				arm.state.registers[Rd] = 0
			} else if arm.state.registers[Rn] == 0x80000000 && arm.state.registers[Rm] == 0xffffffff {
				// "Overflow: If the signed integer division 0x80000000 / 0xFFFFFFFF is performed,
				// the pseudo-code produces the intermediate integer result +2 31 , which
				// overflows the 32-bit signed integer range. No indication of this overflow case
				// is produced, and the 32-bit result written to R[d] is required to be the bottom
				// 32 bits of the binary representation of +2 31 . So the result of the"
				arm.state.registers[Rd] = 0x80000000
			} else {
				arm.state.registers[Rd] = uint32(int32(arm.state.registers[Rn]) / int32(arm.state.registers[Rm]))
			}
		} else {
			panic(fmt.Sprintf("unhandled data processing instructions, non immediate (64bit multiplies) (%03b/%04b)", op, op2))
		}
	} else {
		panic("reserved data processing instructions, non-immediate")
	}

	return nil
}

func (arm *ARM) decode32bitThumb2LoadStoreDoubleEtc(opcode uint16) *DisasmEntry {
	// "3.3.4 Load/store double and exclusive, and table branch" of "Thumb-2 Supplement"

	p := (arm.state.function32bitOpcodeHi & 0x0100) == 0x0100
	u := (arm.state.function32bitOpcodeHi & 0x0080) == 0x0080
	w := (arm.state.function32bitOpcodeHi & 0x0020) == 0x0020
	l := (arm.state.function32bitOpcodeHi & 0x0010) == 0x0010

	Rn := arm.state.function32bitOpcodeHi & 0x000f
	Rt := (opcode & 0xf000) >> 12
	Rt2 := (opcode & 0x0f00) >> 8
	imm8 := opcode & 0x00ff
	imm32 := imm8 << 2

	if p || w {
		// "Load and Store Double"
		addr := arm.state.registers[Rn]

		if p {
			// pre-index addressing
			if u {
				addr += uint32(imm32)
			} else {
				addr -= uint32(imm32)
			}
		}

		if l {
			// "4.6.50 LDRD (immediate)" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "LDRD",
					Operand:  "immediate",
				}
			}

			arm.state.registers[Rt] = arm.read32bit(addr, true)
			arm.state.registers[Rt2] = arm.read32bit(addr+4, true)
		} else {
			// "4.6.167 STRD (immediate)" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "STRD",
					Operand:  "immediate",
				}
			}

			arm.write32bit(addr, arm.state.registers[Rt], true)
			arm.write32bit(addr+4, arm.state.registers[Rt2], true)
		}

		if !p {
			// post-index addressing
			if u {
				addr += uint32(imm32)
			} else {
				addr -= uint32(imm32)
			}
		}

		if w {
			arm.state.registers[Rn] = addr
		}

	} else if arm.state.function32bitOpcodeHi&0x0080 == 0x0080 {
		// "Load and Store Exclusive Byte Halfword, Doubleword, and Table Branch"

		op := (opcode & 0x00f0) >> 4

		switch op {
		case 0b0000:
			// "4.6.188 TBB" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "TBB",
				}
			}

			Rm := opcode & 0x000f
			idx := arm.state.registers[Rn] + arm.state.registers[Rm]
			if Rn == rPC || Rm == rPC {
				idx -= 2
			}
			halfwords := arm.read8bit(idx)
			arm.state.registers[rPC] += uint32(halfwords) << 1
		case 0b0001:
			// "4.6.189 TBH" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "TBH",
				}
			}

			Rm := opcode & 0x000f
			idx := arm.state.registers[Rn] + (arm.state.registers[Rm] << 1)
			if Rn == rPC || Rm == rPC {
				idx -= 2
			}
			halfwords := arm.read16bit(idx, true)
			arm.state.registers[rPC] += uint32(halfwords) << 1
		default:
			panic(fmt.Sprintf("unhandled load and store double and exclusive and table branch (load and store exclusive byte etc.) (%04b)", op))
		}
	} else {
		// "Load and Store Exclusive"
		panic("unhandled load and store double and exclusive and table branch (load and store exclusive)")
	}

	return nil
}

func (arm *ARM) decode32bitThumb2BranchesORDataProcessing(opcode uint16) *DisasmEntry {
	if opcode&0x8000 == 0x8000 {
		return arm.decode32bitThumb2BranchesORMiscControl(opcode)
	}
	return arm.decode32bitThumb2DataProcessing(opcode)
}

func (arm *ARM) decode32bitThumb2DataProcessing(opcode uint16) *DisasmEntry {
	// "3.3.1 Data processing instructions: immediate, including bitfield and saturate" of "Thumb-2 Supplement"

	if arm.state.function32bitOpcodeHi&0xfa00 == 0xf000 {
		// "Data processing instructions with modified 12-bit immediate"
		// page 3-14 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.function32bitOpcodeHi & 0x01e0) >> 5
		setFlags := (arm.state.function32bitOpcodeHi & 0x0010) == 0x0010

		Rn := arm.state.function32bitOpcodeHi & 0x000f
		Rd := (opcode & 0x0f00) >> 8

		i := (arm.state.function32bitOpcodeHi & 0x0400) >> 10
		imm3 := (opcode & 0x7000) >> 12
		imm8 := opcode & 0x00ff
		imm12 := (i << 11) | (imm3 << 8) | imm8

		// some of the instructions in this group (ADD, CMP, etc.) are not
		// interested in the output carry from the ThumbExandImm_c() function.
		// in those cases, the carry flag is obtained by other mean
		imm32, carry := ThumbExpandImm_C(uint32(imm12), arm.state.status.carry)

		switch op {
		case 0b0000:
			if Rd == 0b1111 && setFlags {
				// "4.6.192 TST (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TST",
						Operand:  "immediate",
					}
				}

				result := arm.state.registers[Rn] & imm32
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				// overflow unchanged
			} else {
				// "4.6.8 AND (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "AND",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = arm.state.registers[Rn] & imm32
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0001:
			// "4.6.15 BIC (immediate)" of "Thumb-2 Supplement"
			// T1 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "BIC",
					Operand:  "immediate",
				}
			}

			arm.state.registers[Rd] = arm.state.registers[Rn] & ^imm32
			if setFlags {
				arm.state.status.isNegative(arm.state.registers[Rd])
				arm.state.status.isZero(arm.state.registers[Rd])
				arm.state.status.setCarry(carry)
				// overflow unchanged
			}

		case 0b0010:
			if Rn == 0xf {
				// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MOV",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = imm32
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			} else {
				// "4.6.91 ORR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ORR",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = arm.state.registers[Rn] | imm32
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0011:
			if Rn == 0b1111 {
				// "4.6.85 MVN (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MVN",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = ^imm32
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			} else {
				// "4.6.89 ORN (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ORN",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = arm.state.registers[Rn] | ^imm32
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}
			}

		case 0b0100:
			result := arm.state.registers[Rn] ^ imm32

			if Rd == 0b1111 && setFlags {
				// "4.6.190 TEQ (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TEQ",
						Operand:  "immediate",
					}
				}
			} else {
				// "4.6.36 EOR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "EOR",
						Operand:  "immediate",
					}
				}

				arm.state.registers[Rd] = result
			}

			if setFlags {
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				// overflow unchanged
			}

		case 0b1000:
			if Rd == 0b1111 {
				// "4.6.27 CMN (immediate)" of "Thumb-2 Supplement"
				// T1 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "CMN",
						Operand:  "immediate",
					}
				}

				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, 0)
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				arm.state.status.setOverflow(overflow)
			} else {
				if arm.state.function32bitOpcodeHi&0x100 == 0x100 {
					// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "ADD",
							Operand:  "immediate",
						}
					}

					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, 0)
					arm.state.registers[Rd] = result

					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}
				} else {
					// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
					// T4 encoding
					panic("unimplemented 'ADD (immediate)' T4 encoding")
				}
			}

		case 0b1010:
			// "4.6.1 ADC (immediate)" of "Thumb-2 Supplement"
			// T1 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "ADC",
					Operand:  "immediate",
				}
			}

			var c uint32
			if arm.state.status.carry {
				c = 1
			}

			result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, c)
			arm.state.registers[Rd] = result

			if setFlags {
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				arm.state.status.setOverflow(overflow)
			}

		case 0b1011:
			// "4.6.123 SBC (immediate)" of "Thumb-2 Supplement"
			// T1 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "SBC",
					Operand:  "immediate",
				}
			}

			var c uint32
			if arm.state.status.carry {
				c = 1
			}

			result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, c)
			arm.state.registers[Rd] = result

			if setFlags {
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				arm.state.status.setOverflow(overflow)
			}

		case 0b1101:
			if Rd == 0b1111 {
				// "4.6.29 CMP (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "CMP",
						Operand:  "immediate",
					}
				}

				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
				arm.state.status.isNegative(result)
				arm.state.status.isZero(result)
				arm.state.status.setCarry(carry)
				arm.state.status.setOverflow(overflow)
			} else {
				// "4.6.176 SUB (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SUB",
						Operand:  "immediate",
					}
				}

				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
				arm.state.registers[Rd] = result
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}
			}

		case 0b1110:
			// "4.6.118 RSB (immediate)" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "RSB",
					Operand:  "immediate",
				}
			}

			result, carry, overflow := AddWithCarry(^arm.state.registers[Rn], imm32, 1)
			arm.state.registers[Rd] = result
			if setFlags {
				arm.state.status.isNegative(arm.state.registers[Rd])
				arm.state.status.isZero(arm.state.registers[Rd])
				arm.state.status.setCarry(carry)
				arm.state.status.setOverflow(overflow)
			}

		default:
			panic(fmt.Sprintf("unimplemented 'data processing instructions with modified 12bit immediate' (%04b)", op))
		}
	} else if arm.state.function32bitOpcodeHi&0xfb40 == 0xf200 {
		// "Data processing instructions with plain 12-bit immediate"
		// page 3-15 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.function32bitOpcodeHi & 0x0080) >> 7
		op2 := (arm.state.function32bitOpcodeHi & 0x0030) >> 4

		Rn := arm.state.function32bitOpcodeHi & 0x000f
		Rd := (opcode & 0x0f00) >> 8

		i := (arm.state.function32bitOpcodeHi & 0x0400) >> 10
		imm3 := (opcode & 0x7000) >> 12
		imm8 := opcode & 0x00ff
		imm12 := (i << 11) | (imm3 << 8) | imm8
		imm32 := uint32(imm12)

		// status register doesn't change with these instructions

		switch op {
		case 0b0:
			switch op2 {
			case 0b00:
				// "4.6.3 ADD (immediate) " of "Thumb-2 Supplement"
				// T4 encoding (wide addition)
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "ADDW",
						Operand:  "immediate",
					}
				}

				result, _, _ := AddWithCarry(arm.state.registers[Rn], imm32, 0)
				arm.state.registers[Rd] = result

			default:
				panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 12bit immediate (op=%01b op2=%02b)'", op, op2))
			}
		case 0b1:
			switch op2 {
			case 0b10:
				// "4.6.176 SUB (immediate)" of "Thumb-2 Supplement"
				// T4 encoding (immediate)
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SUBW",
						Operand:  "immediate",
					}
				}

				result, _, _ := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
				arm.state.registers[Rd] = result
			default:
				panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 12bit immediate (op=%01b op2=%02b)'", op, op2))
			}
		}

	} else if arm.state.function32bitOpcodeHi&0xfb40 == 0xf240 {
		// "Data processing instructions with plain 16-bit immediate"
		// page 3-15 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.function32bitOpcodeHi & 0x0080) >> 7
		op2 := (arm.state.function32bitOpcodeHi & 0x0030) >> 4

		if op == 0b0 && op2 == 0b00 {
			// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
			// T3 encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "MOV",
					Operand:  "immediate",
				}
			}

			i := (arm.state.function32bitOpcodeHi & 0x0400) >> 10
			imm4 := arm.state.function32bitOpcodeHi & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm8 := opcode & 0x00ff

			imm32 := uint32((imm4 << 12) | (i << 11) | (imm3 << 8) | imm8)
			arm.state.registers[Rd] = imm32
		} else if op == 0b1 && op2 == 0b00 {
			panic("unimplemented MOVT")
		} else {
			panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 16bit immediate (op=%01b op2=%02b)'", op, op2))
		}

	} else if arm.state.function32bitOpcodeHi&0xfb10 == 0xf300 {
		// "Data processing instructions, bitfield and saturate"
		// page 3-16 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.function32bitOpcodeHi & 0x00e0) >> 5
		switch op {
		case 0b010:
			// "4.6.125 SBFX" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "SBFX",
				}
			}

			Rn := arm.state.function32bitOpcodeHi & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm2 := (opcode & 0x00c0) >> 6
			widthm1 := opcode & 0x001f

			lsbit := (imm3 << 2) | imm2
			msbit := lsbit + widthm1
			width := widthm1 + 1
			if msbit <= 31 {
				arm.state.registers[Rd] = (arm.state.registers[Rn] >> uint32(lsbit)) & ((1 << width) - 1)
				if arm.state.registers[Rd]>>widthm1 == 0x01 {
					arm.state.registers[Rd] = arm.state.registers[Rd] | ^((1 << width) - 1)
				}
			}
		case 0b011:
			// "4.6.14 BFI" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "BFI",
				}
			}

			Rn := arm.state.function32bitOpcodeHi & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm2 := (opcode & 0x00c0) >> 6
			msbit := opcode & 0x001f // labelled msb in the instruction specification

			lsbit := (imm3 << 2) | imm2
			width := msbit - lsbit + 1

			if msbit >= lsbit {
				var mask uint32
				var v uint32

				// remove bits from destination register
				mask = ^(((1 << msbit) - 1) << lsbit)
				arm.state.registers[Rd] = arm.state.registers[Rd] & mask

				// insert bits from source register
				mask = ((1 << width) - 1) << 1
				v = arm.state.registers[Rn] & mask
				arm.state.registers[Rd] = arm.state.registers[Rd] | (v << lsbit)
			} else {
				// unpredictable behaviour
			}

		case 0b110:
			// "4.6.197 UBFX" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "UBFX",
				}
			}

			Rn := arm.state.function32bitOpcodeHi & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm2 := (opcode & 0x00c0) >> 6
			widthm1 := opcode & 0x001f

			lsbit := (imm3 << 2) | imm2
			msbit := lsbit + widthm1
			width := widthm1 + 1
			if msbit <= 31 {
				arm.state.registers[Rd] = (arm.state.registers[Rn] >> uint32(lsbit)) & ((1 << width) - 1)
			}
		default:
			panic(fmt.Sprintf("unimplemented 'bitfield operation' (%03b)", op))
		}
	} else {
		panic("reserved data processing instructions: immediate, including bitfield and saturate")
	}

	return nil
}

func (arm *ARM) decode32bitThumb2LoadStoreSingle(opcode uint16) *DisasmEntry {
	// "3.3.3 Load and store single data item, and memory hints" of "Thumb-2 Supplement"

	// Addressing mode discussed in "A4.6.5 Addressing modes" of "ARMv7-M"

	size := (arm.state.function32bitOpcodeHi & 0x0060) >> 5
	s := arm.state.function32bitOpcodeHi&0x0100 == 0x0100
	l := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
	Rn := arm.state.function32bitOpcodeHi & 0x000f
	Rt := (opcode & 0xf000) >> 12

	// memmory hints are unimplemented. they occur for load instructions
	// whenever the target register is a PC and the size is 8bit or 16bit
	if Rt == rPC && l && (size == 0b00 || size == 0b01) {
		// panic for now. when we come to implement hints properly, we need to
		// further consider the size, sign extension and also the format group
		// the instruction appears in
		panic("unimplemented memory hint")
	}

	if arm.state.function32bitOpcodeHi&0xfe1f == 0xf81f {
		// PC +/ imm12 (format 1 in the table)
		// further depends on size. l is always true

		u := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		imm12 := opcode & 0x0fff

		// Rn is always the PC for this instruction class
		addr := (arm.state.registers[rPC] - 2) & 0xfffffffc

		// all addresses are pre-indexed and there is no write-back
		if u {
			addr += uint32(imm12)
		} else {
			addr -= uint32(imm12)
		}

		switch size {
		case 0b00:
			if s {
				// "4.6.60 LDRSB (literal)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDRSB",
						Operand:  "literal PC relative",
					}
				}
				arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				if arm.state.registers[Rt]&0x80 == 0x80 {
					arm.state.registers[Rt] |= 0xffffff00
				}
			} else {
				// "4.6.47 LDRB (literal)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDRB",
						Operand:  "literal PC relative",
					}
				}
				arm.state.registers[Rt] = uint32(arm.read8bit(addr))
			}
		case 0b01:
			if s {
				// "4.6.64 LDRSH (literal)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDRSH",
						Operand:  "literal PC relative",
					}
				}
				arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				if arm.state.registers[Rt]&0x8000 == 0x8000 {
					arm.state.registers[Rt] |= 0xffff0000
				}
			} else {
				// "4.6.56 LDRH (literal)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDRH",
						Operand:  "literal PC relative",
					}
				}
				arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
			}
		case 0b10:
			// "4.6.44 LDR (literal)" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "LDR",
					Operand:  "literal PC relative",
				}
			}

			arm.state.registers[Rt] = arm.read32bit(addr, false)
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'PC +/- imm12'", size))
		}
	} else if arm.state.function32bitOpcodeHi&0xfe80 == 0xf880 {
		// Rn + imm12 (format 2 in the table)
		//
		// immediate offset
		//
		// further depends on size and L bit

		// U is always up for this format meaning that we add the index to
		// the base address
		imm12 := opcode & 0x0fff

		// all addresses are pre-indexed and there is no write-back
		addr := arm.state.registers[Rn] + uint32(imm12)

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSB",
							Operand:  "immediate offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "immediate offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRB",
						Operand:  "immediate offset",
					}
				}
				arm.write8bit(addr, uint8(arm.state.registers[Rt]))
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDSH",
							Operand:  "immediate offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  "immediate offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRH",
						Operand:  "immediate offset",
					}
				}
				arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  "immediate offset",
					}
				}
				arm.state.registers[Rt] = arm.read32bit(addr, false)
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STR",
						Operand:  "immediate offset",
					}
				}
				arm.write32bit(addr, arm.state.registers[Rt], false)
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + imm12'", size))
		}

	} else if (opcode & 0x0f00) == 0x0c00 {
		// Rn - imm8 (format 3 in the table)
		//
		// negative immediate offset
		//
		// further depends on size and L bit

		imm8 := opcode & 0x00ff

		// all addresses are pre-indexed and there is no write-back
		addr := arm.state.registers[Rn] - uint32(imm8)

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSB",
							Operand:  "immediate negative offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "immediate negative offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRB",
						Operand:  "immediate negative offset",
					}
				}
				arm.write8bit(addr, uint8(arm.state.registers[Rt]))
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSH",
							Operand:  "immediate negative offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  "immediate negative offset",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRH",
						Operand:  "immediate negative offset",
					}
				}
				arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  "immediate negative offset",
					}
				}
				arm.state.registers[Rt] = arm.read32bit(addr, false)
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STR",
						Operand:  "immediate negative offset",
					}
				}
				arm.write32bit(addr, arm.state.registers[Rt], false)
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn - imm8'", size))
		}

	} else if (opcode & 0x0f00) == 0x0e00 {
		// Rn + imm8, user privilege (format 4 in the table)
		// imm8 := opcode & 0x00ff
		panic("unimplemented Rn + imm8, user privilege")

	} else if (opcode & 0x0d00) == 0x0900 {
		// Rn post-index by +/- imm8 (format 5 in the table)
		imm8 := opcode & 0x00ff
		u := (opcode & 0x0200) == 0x0200

		// all addresses are post-indexed and there is write-back
		addr := arm.state.registers[Rn]

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "immediate post-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "immediate post-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRB",
						Operand:  "immediate post-index",
					}
				}
				arm.write8bit(addr, uint8(arm.state.registers[Rt]))
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSH",
							Operand:  "immediate post-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  "immediate post-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRH",
						Operand:  "immediate post-index",
					}
				}
				arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  "immediate post-index",
					}
				}
				if Rt == rPC {
					arm.state.registers[Rt] = arm.read32bit(addr, false) + 1
				} else {
					arm.state.registers[Rt] = arm.read32bit(addr, false)
				}
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STR",
						Operand:  "immediate post-index",
					}
				}
				arm.write32bit(addr, arm.state.registers[Rt], false)
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn post-index +/- imm8'", size))
		}

		// post-index
		if u {
			addr += uint32(imm8)
		} else {
			addr -= uint32(imm8)
		}

		// write-back
		arm.state.registers[Rn] = addr

	} else if (opcode & 0x0d00) == 0x0d00 {
		// Rn pre-indexed by +/- imm8 (format 6 in the table)
		imm8 := opcode & 0x00ff
		u := (opcode & 0x0200) == 0x0200

		// all addresses are pre-indexed and there is write-back
		addr := arm.state.registers[Rn]
		if u {
			addr += uint32(imm8)
		} else {
			addr -= uint32(imm8)
		}

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSB",
							Operand:  "immediate pre-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "immediate pre-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRB",
						Operand:  "immediate pre-index",
					}
				}
				arm.write8bit(addr, uint8(arm.state.registers[Rt]))
			}
		case 0b01:
			if l {
				arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSH",
							Operand:  "immediate pre-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  "immediate pre-index",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRH",
						Operand:  "immediate pre-index",
					}
				}
				arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  "immediate offset",
					}
				}
				arm.state.registers[Rt] = arm.read32bit(addr, false)
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STR",
						Operand:  "immediate offset",
					}
				}
				arm.write32bit(addr, arm.state.registers[Rt], false)
			}
		default:
			panic(fmt.Sprintf("unhandled size (%02b) for 'Rn +/- imm8'", size))
		}

		// write-back
		arm.state.registers[Rn] = addr

	} else if (opcode & 0x0fc0) == 0x0000 {
		// Rn + shifted register (format 7 in the table)
		shift := (opcode & 0x0030) >> 4
		Rm := opcode & 0x000f

		// all addresses are pre-indexed by a shifted register and there is no write-back
		addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)

		if l {
			switch size {
			case 0b00:
				if s {
					// "4.6.61 LDRSB (register)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSB",
							Operand:  "register shifted",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
				} else {
					// "4.6.48 LDRB (register)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  "register shifted",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
				}
			case 0b01:
				if s {
					// "4.6.65 LDRSH (register)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSH",
							Operand:  "register shifted",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
				} else {
					// "4.6.57 LDRH (register)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  "register shifted",
						}
					}
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
				}
			case 0b10:
				// "4.6.45 LDR (register)" of "Thumb-2 Supplement"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  "register shifted",
					}
				}
				arm.state.registers[Rt] = arm.read32bit(addr, false)
				if Rt == rPC {
					arm.state.registers[rPC] += 2
					arm.state.registers[rPC] &= 0xfffffffe
				}
			default:
				panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + shifted register' (load)", size))
			}
		} else {
			switch size {
			case 0b00:
				// "4.6.165 STRB (register)" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRB",
						Operand:  "register shifted",
					}
				}
				arm.write8bit(addr, uint8(arm.state.registers[Rt]))
			case 0b01:
				// "4.6.173 STRH (register)" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STRH",
						Operand:  "register shifted",
					}
				}
				arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
			case 0b10:
				// "4.6.163 STR (register)" of "Thumb-2 Supplement"
				// T2 encoding
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STR",
						Operand:  "register shifted",
					}
				}
				arm.write32bit(addr, arm.state.registers[Rt], false)
			default:
				panic(fmt.Sprintf("unhandled size (%02b) for 'Rn + shifted register' (save)", size))
			}
		}

	} else {
		panic("unhandled bit pattern in 'load and store single data item, and memory hints'")
	}

	return nil
}

func (arm *ARM) decode32bitThumb2LoadStoreMultiple(opcode uint16) *DisasmEntry {
	// "3.3.5 Load and store multiple, RFE, and SRS" of "Thumb-2 Supplement"
	//		and
	// "A5.3.5 Load Multiple and Store Multiple" of "ARMv7-M"

	op := (arm.state.function32bitOpcodeHi & 0x0180) >> 7
	l := (arm.state.function32bitOpcodeHi & 0x0010) == 0x0010
	w := (arm.state.function32bitOpcodeHi & 0x0020) == 0x0020
	Rn := arm.state.function32bitOpcodeHi & 0x000f

	WRn := Rn
	if w {
		WRn |= 0x0010
	}

	switch op {
	case 0b01:
		if l {
			switch WRn {
			case 0b11101:
				// "4.6.98 POP" of "Thumb-2 Supplement"
				// T2 encoding

				// Pop multiple registers from the stack
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "POP",
					}
				}

				regList := opcode & 0xdfff
				c := uint32(bits.OnesCount16(regList) * 4)
				addr := arm.state.registers[rSP]
				arm.state.registers[rSP] += c

				// read each register in turn (from lower to highest)
				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// read register if indicated by regList
					if regList&m == m {
						arm.state.registers[i] = arm.read32bit(addr, true)
						addr += 4
					}
				}

				// write PC
				if regList&0x8000 == 0x8000 {
					arm.state.registers[rPC] = (arm.read32bit(addr, true) + 2) & 0xfffffffe
				}
			default:
				// "4.6.42 LDMIA / LDMFD" of "Thumb-2 Supplement"
				// T2 encoding

				// Load multiple (increment after, full descending)
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDMIA/LDMFD",
					}
				}

				regList := opcode & 0xdfff
				c := uint32(bits.OnesCount16(regList) * 4)
				addr := arm.state.registers[Rn]

				// update register if W bit is set
				if w {
					arm.state.registers[Rn] += c
				}

				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// read register if indicated by regList
					if regList&m == m {
						if i == int(Rn) {
							panic("LDMIA/LDMFD writeback register is being loaded")
						}
						arm.state.registers[i] = arm.read32bit(addr, true)
						addr += 4
					}
				}

				// write PC
				if regList&0x8000 == 0x8000 {
					arm.state.registers[rPC] = arm.read32bit(addr, true)
				}

			}
		} else {
			// "4.6.161 STMIA / STMEA" of "Thumb-2 Supplement"
			// T2 encoding

			// Store multiple (increment after, empty ascending)
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "STMIA/STMEA",
				}
			}

			regList := opcode & 0x5fff
			c := uint32(bits.OnesCount16(regList) * 4)
			addr := arm.state.registers[Rn]

			// update register if W bit is set
			if w {
				arm.state.registers[Rn] += c
			}

			for i := 0; i <= 14; i++ {
				// shift single-bit mask
				m := uint16(0x01 << i)

				// write register if indicated by regList
				if regList&m == m {
					if i == int(Rn) {
						panic("STMIA/STMEA writeback register is being stored")
					}

					// there is a branch in the pseudocode that applies to T1
					// encoding only. ommitted here
					arm.write32bit(addr, arm.state.registers[i], true)
					addr += 4
				}
			}

		}
	case 0b10:
		if l {
			// "4.6.41 LDMDB / LDMEA" of "Thumb-2 Supplement"

			// Load multiple (decrement before, empty ascending)
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "LDMDB/LDMEA",
				}
			}

			regList := opcode & 0xdfff
			c := uint32(bits.OnesCount16(regList) * 4)
			addr := arm.state.registers[Rn] - c

			// update register if W bit is set
			if w {
				arm.state.registers[Rn] -= c
			}

			// read each register in turn (from lower to highest)
			for i := 0; i <= 14; i++ {
				// shift single-bit mask
				m := uint16(0x01 << i)

				// read register if indicated by regList
				if regList&m == m {
					if i == int(Rn) {
						panic("LDMDB/LDMEA writeback register is being loaded")
					}
					arm.state.registers[i] = arm.read32bit(addr, true)
					addr += 4
				}
			}

			// write PC
			if regList&0x8000 == 0x8000 {
				arm.state.registers[rPC] = arm.read32bit(addr, true)
			}

		} else {
			switch WRn {
			case 0b11101:
				// "4.6.99 PUSH" of "Thumb-2 Supplement"
				// T2 encoding

				// Push multiple registers to the stack
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "PUSH",
					}
				}

				regList := opcode & 0x5fff
				c := (uint32(bits.OnesCount16(regList))) * 4
				addr := arm.state.registers[rSP] - c

				// store each register in turn (from lowest to highest)
				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// write register if indicated by regList
					if regList&m == m {
						arm.write32bit(addr, arm.state.registers[i], true)
						addr += 4
					}
				}

				arm.state.registers[rSP] -= c
			default:
				// "4.6.160 STMDB / STMFD" of "Thumb-2 Supplement"

				// Store multiple (decrement before, full descending)
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SRMDB/STMFD",
					}
				}

				regList := opcode & 0x5fff
				c := (uint32(bits.OnesCount16(regList))) * 4
				addr := arm.state.registers[Rn] - c

				// update register if W bit is set
				if w {
					arm.state.registers[Rn] -= c
				}

				// store each register in turn (from lowest to highest)
				for i := 0; i <= 14; i++ {
					// shift single-bit mask
					m := uint16(0x01 << i)

					// write register if indicated by regList
					if regList&m == m {
						if i == int(Rn) {
							panic("STMDB/STMDF writeback register is being stored")
						}
						arm.write32bit(addr, arm.state.registers[i], true)
						addr += 4
					}
				}
			}
		}
	default:
		panic(fmt.Sprintf("load and store multiple: illegal op (%02b)", op))
	}

	return nil
}

func (arm *ARM) decode32bitThumb2BranchesORMiscControl(opcode uint16) *DisasmEntry {
	// "3.3.6 Branches, miscellaneous control instructions" of "Thumb-2 Supplement"

	if arm.state.function32bitOpcodeHi&0xffe0 == 0xf3e0 {
		panic("move to register from status")
	} else if arm.state.function32bitOpcodeHi&0xfff0 == 0xf3d0 {
		panic("exception return")
	} else if arm.state.function32bitOpcodeHi&0xfff0 == 0xf3c0 {
		panic("branch, change to java")
	} else if arm.state.function32bitOpcodeHi&0xfff0 == 0xf3b0 {
		panic("special control operations")
	} else if arm.state.function32bitOpcodeHi&0xfff0 == 0xf3a0 {
		imodM := (opcode & 0x0700) >> 8
		if imodM == 0b000 {
			panic("NOP, hints")
		} else {
			panic("change processor state")
		}
	} else if arm.state.function32bitOpcodeHi&0xffe0 == 0xf380 {
		panic("move to status from register")
	} else if arm.state.function32bitOpcodeHi&0xf800 == 0xf000 {
		return arm.decode32bitThumb2Branches(opcode)
	} else {
		panic(fmt.Sprintf("unimplemented branches, miscellaneous control instructions"))
	}

	return nil
}

func (arm *ARM) decode32bitThumb2Branches(opcode uint16) *DisasmEntry {
	// "3.3.6 Branches, miscellaneous control instructions" of "Thumb-2 Supplement"
	//
	// branches are in the top half of the table and are differentiated by the
	// second half of the instruction (ie. the opcode argument to this
	// function)

	if opcode&0xd000 == 0x8000 {
		// "4.6.12 B" of "Thumb-2 Supplement"
		// T3 encoding
		// Conditional Branch
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "B",
				Operand:  "conditional",
			}
		}

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.function32bitOpcodeHi & 0x0400) >> 10)
		cond := (arm.state.function32bitOpcodeHi & 0x03c0) >> 6
		imm6 := uint32(arm.state.function32bitOpcodeHi & 0x003f)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x0800) >> 11)
		imm11 := uint32(opcode & 0x07ff)

		imm32 := (s << 20) | (j2 << 19) | (j1 << 18) | (imm6 << 12) | (imm11 << 1)

		if s == 0x01 {
			imm32 |= 0xfff00000
		}

		if arm.state.status.condition(uint8(cond)) {
			arm.state.registers[rPC] += imm32
		}

	} else if opcode&0xc000 == 0xc000 {
		// BL and BLX instructions differ by one bit
		blx := opcode&0x1000 != 0x1000

		// "4.6.18 BL, BLX (immediate)" of "Thumb-2 Supplment"
		// T1 encoding
		// Long Branch With link
		if arm.decodeOnly {
			if blx {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "BLX",
				}
			}
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "BL",
			}
		}

		// record PC in link register
		arm.state.registers[rLR] = (arm.state.registers[rPC]-2)&0xfffffffe | 0x00000001

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.function32bitOpcodeHi & 0x400) >> 10)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x800) >> 11)
		i1 := (^(j1 ^ s)) & 0x01
		i2 := (^(j2 ^ s)) & 0x01

		var imm32 uint32

		if blx {
			imm10H := uint32(arm.state.function32bitOpcodeHi & 0x3ff)
			imm10L := uint32(opcode & 0x7ff)

			// immediate 32bit value is sign extended
			imm32 = (s << 23) | (i1 << 22) | (i2 << 21) | (imm10H << 11) | (imm10L << 2)
			if s == 0x01 {
				imm32 |= 0xff000000
			}
		} else {
			imm10 := uint32(arm.state.function32bitOpcodeHi & 0x3ff)
			imm11 := uint32(opcode & 0x7ff)

			// immediate 32bit value is sign extended
			imm32 = (s << 24) | (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)
			if s == 0x01 {
				imm32 |= 0xff000000
			}
		}

		// adjust PC
		arm.state.registers[rPC] += imm32

	} else if opcode&0xd000 == 0x9000 {
		// "4.6.12 B" of "Thumb-2 Supplement"
		// T4 encoding
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "B",
				Operand:  "non-conditional",
			}
		}

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.function32bitOpcodeHi & 0x400) >> 10)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x800) >> 11)
		i1 := (^(j1 ^ s)) & 0x01
		i2 := (^(j2 ^ s)) & 0x01
		imm10 := uint32(arm.state.function32bitOpcodeHi & 0x3ff)
		imm11 := uint32(opcode & 0x7ff)

		// immediate 32bit value is sign extended
		imm32 := (s << 24) | (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)
		if s == 0x01 {
			imm32 |= 0xff000000
		}

		// adjust PC
		arm.state.registers[rPC] += imm32

	} else {
		panic(fmt.Sprintf("unimplemented branches, miscellaneous control instructions"))
	}

	return nil
}
