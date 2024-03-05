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

func (arm *ARM) decode32bitThumb2(opcodeHi uint16, opcode uint16) decodeFunction {
	// Two tables for top level decoding of 32bit Thumb-2 instructions.
	//
	// "3.3 Instruction encoding for 32-bit Thumb instructions" of "Thumb-2 Supplement"
	//		and
	// "A5.3 32-bit Thumb instruction encoding" of "ARMv7-M"
	//
	// Both with different emphasis but the table in the "Thumb-2 Supplement"
	// was used.

	if opcodeHi&0xec00 == 0xec00 {
		// coprocessor
		return arm.decodeThumb2Coproc(opcode)
	} else if opcodeHi&0xf800 == 0xf000 {
		// branches, miscellaneous control
		//  OR
		// data processing: immediate, including bitfield and saturate
		return arm.decode32bitThumb2BranchesORDataProcessing(opcode)
	} else if opcodeHi&0xfe40 == 0xe800 {
		// load and store multiple, RFE and SRS
		return arm.decode32bitThumb2LoadStoreMultiple(opcode)
	} else if opcodeHi&0xfe40 == 0xe840 {
		// load and store double and exclusive and table branch
		return arm.decode32bitThumb2LoadStoreDoubleEtc(opcode)
	} else if opcodeHi&0xfe00 == 0xf800 {
		// load and store single data item, memory hints
		return arm.decode32bitThumb2LoadStoreSingle(opcode)
	} else if opcodeHi&0xee00 == 0xea00 {
		// data processing, no immediate operand
		return arm.decode32bitThumb2DataProcessingNonImmediate(opcode)
	}

	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (%04x)", opcodeHi))
}

func (arm *ARM) decode32bitThumb2DataProcessingNonImmediate(opcode uint16) decodeFunction {
	// "3.3.2 Data processing instructions, non-immediate" of "Thumb-2 Supplement"

	Rn := arm.state.instruction32bitOpcodeHi & 0x000f
	Rm := opcode & 0x000f
	Rd := (opcode & 0x0f00) >> 8

	if arm.state.instruction32bitOpcodeHi&0xfe00 == 0xea00 {
		// "Data processing instructions with constant shift"
		// page 3-18 of "Thumb-2 Supplement"
		op := (arm.state.instruction32bitOpcodeHi & 0x01e0) >> 5
		setFlags := arm.state.instruction32bitOpcodeHi&0x0010 == 0x0010
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

			return func() *DisasmEntry {
				// disassembly only
				if arm.decodeOnly {
					if tst {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "TST",
							Operand:  fmt.Sprintf("R%d, R%d, %s #%d", Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					} else {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("AND%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
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
					panic("impossible shift for TST/AND instruction")
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

				return nil
			}

		case 0b0001:
			// "4.6.16 BIC (register)" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("BIC%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
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

					// change status register
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

					// change status register
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

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}
				default:
					panic("impossible shift for BIC instruction")
				}

				return nil
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
						return func() *DisasmEntry {
							if arm.decodeOnly {
								return &DisasmEntry{
									Is32bit:  true,
									Operator: fmt.Sprintf("MOV%s", setFlagsMnemonic(setFlags)),
									Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
								}
							}

							// perform move
							arm.state.registers[Rd] = arm.state.registers[Rm]

							// change status register
							if setFlags {
								arm.state.status.isNegative(arm.state.registers[Rd])
								arm.state.status.isZero(arm.state.registers[Rd])

								// carry unchanged. there is a mistake in the
								// Thumb-2 Supplement but it is clear from the
								// ARMv7-M that carry is not affected by this
								// instruction

								// overflow unchanged
							}

							return nil
						}
					} else {
						// "4.6.68 LSL (immediate)" of "Thumb-2 Supplement"
						// T2 encoding
						return func() *DisasmEntry {
							if arm.decodeOnly {
								return &DisasmEntry{
									Is32bit:  true,
									Operator: fmt.Sprintf("LSL%s", setFlagsMnemonic(setFlags)),
									Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, imm5),
								}
							}

							// isolate carry bit information
							m := uint32(0x01) << (32 - imm5)
							carry := arm.state.registers[Rm]&m == m

							// perform shift
							arm.state.registers[Rd] = arm.state.registers[Rm] << imm5

							// change status register
							if setFlags {
								arm.state.status.isNegative(arm.state.registers[Rd])
								arm.state.status.isZero(arm.state.registers[Rd])
								arm.state.status.setCarry(carry)
								// overflow unchanged
							}

							return nil
						}
					}
				case 0b01:
					// "4.6.70 LSR (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: fmt.Sprintf("LSR%s", setFlagsMnemonic(setFlags)),
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, imm5),
							}
						}

						// isolate carry bit information
						m := uint32(0x01) << (imm5 - 1)
						carry := arm.state.registers[Rm]&m == m

						// perform shift
						arm.state.registers[Rd] = arm.state.registers[Rm] >> imm5

						// change status register
						if setFlags {
							arm.state.status.isNegative(arm.state.registers[Rd])
							arm.state.status.isZero(arm.state.registers[Rd])
							arm.state.status.setCarry(carry)
							// overflow unchanged
						}

						return nil
					}
				case 0b10:
					// "4.6.10 ASR (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: fmt.Sprintf("ASR%s", setFlagsMnemonic(setFlags)),
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, imm5),
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

						// change status register
						if setFlags {
							arm.state.status.isNegative(arm.state.registers[Rd])
							arm.state.status.isZero(arm.state.registers[Rd])
							arm.state.status.setCarry(carry)
							// overflow unchanged
						}

						return nil
					}
				case 0b11:
					if imm5 == 0b00000 {
						// 4.6.117 RRX Rotate Right with extend
						// T1 encoding
						return func() *DisasmEntry {
							if arm.decodeOnly {
								return &DisasmEntry{
									Is32bit:  true,
									Operator: fmt.Sprintf("RRX%s", setFlagsMnemonic(setFlags)),
									Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
								}
							}

							// perform rotation
							result, carry := RRX_C(arm.state.registers[Rm], arm.state.status.carry)
							arm.state.registers[Rd] = result

							// change status register
							if setFlags {
								arm.state.status.isNegative(arm.state.registers[Rd])
								arm.state.status.isZero(arm.state.registers[Rd])
								arm.state.status.setCarry(carry)
								// overflow unchanged
							}

							return nil
						}
					} else {
						// 4.6.115 ROR (immediate)
						// T1 encoding
						return func() *DisasmEntry {
							if arm.decodeOnly {
								return &DisasmEntry{
									Is32bit:  true,
									Operator: fmt.Sprintf("ROR%s", setFlagsMnemonic(setFlags)),
									Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, imm5),
								}
							}

							// perform rotation
							result, carry := ROR_C(arm.state.registers[Rm], uint32(imm5))
							arm.state.registers[Rd] = result

							// change status register
							if setFlags {
								arm.state.status.isNegative(arm.state.registers[Rd])
								arm.state.status.isZero(arm.state.registers[Rd])
								arm.state.status.setCarry(carry)
								// overflow unchanged
							}

							return nil
						}
					}
				default:
					panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (data processing, constant shift) (%04b) (%02b)", op, typ))
				}
			} else {
				// "4.6.92 ORR (register)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("ORR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, #%d", Rd, Rn, Rm, imm5),
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
					case 0b10:
						// with arithmetic right shift
						panic("unimplemented arithmetic right shift for ORR instruction")
					default:
						panic("impossible shift for ORR instruction")
					}

					// store result
					arm.state.registers[Rd] = result

					// change status register
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			}

		case 0b0011:
			if Rn == rPC {
				// "4.6.86 MVN (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("MVN%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, %s #%d", Rd, Rm, shiftTypeToMnemonic(typ), imm5),
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
						panic("impossible shift for MVN instruction")
					}

					// perform negation and store result
					result = ^result
					arm.state.registers[Rd] = result

					// change status register
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			} else {
				// "4.6.90 ORN (register)" of "Thumb-2 Supplement"
				// T1 encoding
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

			return func() *DisasmEntry {
				if Rd == rPC && setFlags {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "TEQ",
							Operand:  fmt.Sprintf("R%d, R%d, %s #%d", Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					}
				} else {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("EOR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					}
				}

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
				case 0b10:
					// with arithmetic right shift
					signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
					result = arm.state.registers[Rn] ^ (arm.state.registers[Rm] >> imm5)
					if signExtend == 0x01 {
						result |= ^uint32(0) << (32 - imm5)
					}
				case 0b11:
					if imm5 == 0b00000 {
						result, carry = RRX_C(arm.state.registers[Rm], arm.state.status.carry)
					} else {
						result, carry = ROR_C(arm.state.registers[Rm], uint32(imm5))
					}
				}

				// perform EOR or do nothing if this is just a TST
				if !(Rd == rPC && setFlags) {
					arm.state.registers[Rd] = result
				}

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

				return nil
			}

		case 0b1000:
			if Rd == rPC {
				// "4.6.28 CMN (register)" of "Thumb-2 Supplement"
				panic("unimplemented CMN (register) instruction")
			} else {
				// "4.6.4 ADD (register)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("ADD%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					}

					var shifted uint32

					switch typ {
					case 0b00:
						// with logical left shift
						shifted = arm.state.registers[Rm] << imm5
					case 0b01:
						// with logical right shift
						shifted = arm.state.registers[Rm] >> imm5
					case 0b10:
						// with arithmetic right shift
						signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
						shifted = arm.state.registers[Rm] >> imm5
						if signExtend == 0x01 {
							shifted |= ^uint32(0) << (32 - imm5)
						}
					default:
						panic("impossible shift for ADD (register) instruction")
					}

					// peform addition and store result
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, 0)
					arm.state.registers[Rd] = result

					// change status register
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}

					return nil
				}
			}

		case 0b1010:
			// "4.6.2 ADC (register)" of "Thumb-2 Supplement")
			// T2 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("ADD%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
					}
				}

				var shifted uint32

				switch typ {
				case 0b00:
					// with logical left shift
					shifted = arm.state.registers[Rm] << imm5
				case 0b01:
					panic("unimplemented logical right shift for ADC (register) instruction")
				case 0b10:
					panic("unimplemented arithmetic right shift for ADC (register) instruction")
				default:
					panic("impossible shift for ADC (register) instruction")
				}

				// carry value taken from carry bit in status register
				var c uint32
				if arm.state.status.carry {
					c = 1
				}

				// peform addition and store result
				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], shifted, c)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		case 0b1011:
			// "4.6.124 SBC (register) Subtract with Carry" of "Thumb-2 Supplement"
			// T2 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("SBC%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
					}
				}

				var c uint32
				if arm.state.status.carry {
					c = 1
				}

				var shifted uint32

				switch typ {
				case 0b00:
					// with logical left shift
					shifted = arm.state.registers[Rm] << imm5
				case 0b01:
					panic("unimplemented logical right shift for SBC (register) instruction")
				case 0b10:
					panic("unimplemented arithmetic right shift for SBC (register) instruction")
				default:
					panic("impossible shift for SBC (register) instruction")
				}

				// perform subtraction and store result
				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, c)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		case 0b1101:
			if Rd == rPC {
				// "4.6.30 CMP (register)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "CMP",
							Operand:  fmt.Sprintf("R%d, R%d, %s #%d", Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					}

					var shifted uint32

					switch typ {
					case 0b00:
						// with logical left shift
						shifted = arm.state.registers[Rm] << imm5
					case 0b01:
						// with logical right shift
						shifted = arm.state.registers[Rm] >> imm5
					case 0b10:
						panic("unimplemented arithmetic right shift for CMP (register) instruction")
					default:
						panic("impossible shift for CMP (register) instruction")
					}

					// perform comparison
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)

					// change status register
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}

					return nil
				}
			} else {
				// "4.6.177 SUB (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("SUB%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
						}
					}

					var shifted uint32

					switch typ {
					case 0b00:
						// with logical left shift
						shifted = arm.state.registers[Rm] << imm5
					case 0b01:
						// with logical right shift
						shifted = arm.state.registers[Rm] >> imm5
					case 0b10:
						// with arithmetic right shift
						signExtend := (arm.state.registers[Rm] & 0x80000000) >> 31
						shifted = arm.state.registers[Rm] >> imm5
						if signExtend == 0x01 {
							shifted |= ^uint32(0) << (32 - imm5)
						}
					default:
						panic("impossible shift for SUB (register) instruction")
					}

					// perform subtraction and store result
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^shifted, 1)
					arm.state.registers[Rd] = result

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}

					return nil
				}
			}

		case 0b1110:
			// "4.6.119 RSB (register)" of "Thumb-2 Supplement"
			// T1 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("RSB%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, R%d, %s #%d", Rd, Rn, Rm, shiftTypeToMnemonic(typ), imm5),
					}
				}

				var shifted uint32

				switch typ {
				case 0b00:
					// with logical left shift
					shifted = arm.state.registers[Rm] << imm5
				case 0b01:
					// with logical right shift
					shifted = arm.state.registers[Rm] >> imm5
				case 0b10:
					panic("unimplemented arithmetic right shift for RSB (register) instruction")
				default:
					panic("impossible shift for RSB (register) instruction")
				}

				// perform subtraction and store result
				result, carry, overflow := AddWithCarry(^arm.state.registers[Rn], shifted, 1)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		default:
			panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (data processing, constant shift) (%04b)", op))
		}
	} else if arm.state.instruction32bitOpcodeHi&0xff80 == 0xfa00 {
		if opcode&0x0080 == 0x0000 {
			// "Register-controlled shift instructions"
			// page 3-19 of "Thumb-2 Supplement"

			op := (arm.state.instruction32bitOpcodeHi & 0x0060) >> 5
			setFlags := (arm.state.instruction32bitOpcodeHi & 0x0010) == 0x0010

			switch op {
			case 0b00:
				// "4.6.69 LSL (register)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("LSL%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
						}
					}

					// whether to set carry bit
					shift := arm.state.registers[Rm] & 0x00ff
					m := uint32(0x01) << (32 - shift)
					carry := arm.state.registers[Rn]&m == m

					// perform actual shift
					arm.state.registers[Rd] = arm.state.registers[Rn] << shift

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}

			case 0b01:
				// "4.6.71 LSR (register)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("LSR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
						}
					}

					// whether to set carry bit
					shift := arm.state.registers[Rm] & 0x00ff
					m := uint32(0x01) << (shift - 1)
					carry := arm.state.registers[Rn]&m == m

					// perform actual shift
					arm.state.registers[Rd] = arm.state.registers[Rn] >> shift

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			case 0b10:
				// "4.6.11 ASR (register)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("ASR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
						}
					}

					// whether to set carry bit
					shift := arm.state.registers[Rm] & 0x00ff
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

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			default:
				panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (reg controlled shift) (%02b)", op))
			}
		} else {
			// "Signed and unsigned extend instructions with optional addition"
			// page 3-20 of "Thumb-2 Supplement"
			op := (arm.state.instruction32bitOpcodeHi & 0x0070) >> 4
			rot := (opcode & 0x0030) >> 4

			// rot is actually always used with a right shift of 3
			rot <<= 3

			switch op {
			case 0b000:
				if Rn == rPC {
					// "4.6.187 SXTH" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "SXTH",
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = v & 0x0000ffff
						if arm.state.registers[Rd]&0x8000 == 0x8000 {
							arm.state.registers[Rd] |= 0xffff0000
						}

						return nil
					}
				} else {
					// "4.6.184 SXTAH" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "SXTAH",
								Operand:  fmt.Sprintf("R%d, R%d, R%d, #%d", Rd, Rn, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x0000ffff)
						if arm.state.registers[Rd]&0x8000 == 0x8000 {
							arm.state.registers[Rd] |= 0xffff0000
						}

						return nil
					}
				}

			case 0b001:
				if Rn == rPC {
					// "4.6.226 UXTH" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "UXTH",
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = v & 0x0000ffff

						return nil
					}
				} else {
					// "4.6.223 UXTAH" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "UXTAH",
								Operand:  fmt.Sprintf("R%d, R%d, R%d, #%d", Rd, Rn, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x0000ffff)

						return nil
					}
				}
			case 0b101:
				if Rn == rPC {
					// "4.6.224 UXTB" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "UXTB",
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = v & 0x000000ff

						return nil
					}
				} else {
					// "4.6.221 UXTAB" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "UXTAB",
								Operand:  fmt.Sprintf("R%d, R%d, R%d, #%d", Rd, Rn, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x000000ff)

						return nil
					}
				}

			case 0b100:
				if Rn == 0b1111 {
					// "4.6.185 SXTB" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "SXTB",
								Operand:  fmt.Sprintf("R%d, R%d, #%d", Rd, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = v & 0x000000ff
						if arm.state.registers[Rd]&0x80 == 0x80 {
							arm.state.registers[Rd] |= 0xffffff00
						}

						return nil
					}
				} else {
					// "4.6.182 SXTAB" of "Thumb-2 Supplement"
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "SXTAB",
								Operand:  fmt.Sprintf("R%d, R%d, R%d, #%d", Rd, Rn, Rm, rot),
							}
						}

						v, _ := ROR_C(arm.state.registers[Rm], uint32(rot))
						arm.state.registers[Rd] = arm.state.registers[Rn] + (v & 0x000000ff)
						if arm.state.registers[Rd]&0x80 == 0x80 {
							arm.state.registers[Rd] |= 0xffffff00
						}

						return nil
					}
				}

			default:
				panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (sign or zero extension with opt addition) (%03b)", op))
			}
		}
	} else if arm.state.instruction32bitOpcodeHi&0xff80 == 0xfa80 {
		if opcode&0x0080 == 0x0000 {
			// "SIMD add and subtract"
			// page 3-21 of "Thumb-2 Supplement"
			panic("unimplemented SIMD add and subtract")
		} else {
			// "Other three-register data processing instructions"
			// page 3-23 of "Thumb-2 Supplement"
			op := (arm.state.instruction32bitOpcodeHi & 0x70) >> 4
			op2 := (opcode & 0x0070) >> 4
			if op == 0b011 && op2 == 0b000 {
				// "4.6.26 CLZ" of "Thumb-2 Supplement"
				// T1 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "CLZ",
							Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
						}
					}

					c := bits.LeadingZeros32(arm.state.registers[Rm])
					arm.state.registers[Rd] = uint32(c)

					return nil
				}
			} else {
				panic("unimplemented 'three-register data processing instruction'")
			}
		}
	} else if arm.state.instruction32bitOpcodeHi&0xff80 == 0xfb00 {
		// "32-bit multiplies and sum of absolute differences, with or without accumulate"
		// page 3-24 of "Thumb-2 Supplement"
		op := (arm.state.instruction32bitOpcodeHi & 0x0070) >> 4
		Ra := (opcode & 0xf000) >> 12
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b000 && op2 == 0b0000 {
			if Ra == rPC {
				// "4.6.84 MUL" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "MUL",
							Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
						}
					}

					// multiplication can be done on signed or unsigned value with
					// not change in functionality
					arm.state.registers[Rd] = arm.state.registers[Rn] * arm.state.registers[Rm]

					return nil
				}
			} else {
				// "4.6.74 MLA" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "MLA",
							Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", Rd, Rn, Rm, Ra),
						}
					}

					result := int(arm.state.registers[Rn]) * int(arm.state.registers[Rm])
					result += int(arm.state.registers[Ra])
					arm.state.registers[Rd] = uint32(result)

					return nil
				}
			}
		} else if op == 0b000 && op2 == 0b0001 {
			// "4.6.75 MLS" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MLS",
						Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", Rd, Rn, Rm, Ra),
					}
				}

				arm.state.registers[Rd] = uint32(int32(arm.state.registers[Ra]) - int32(arm.state.registers[Rn])*int32(arm.state.registers[Rm]))

				return nil
			}
		} else if op == 0b001 && op2 == 0b0000 {
			if Ra == 0b1111 {
				// "4.6.149 SMULBB, SMULBT, SMULTB, SMULTT" of "Thumb-2 Supplement"
				nHigh := opcode&0x0020 == 0x0020
				mHigh := opcode&0x0010 == 0x0010

				return func() *DisasmEntry {
					if arm.decodeOnly {
						x := 'B'
						y := 'B'
						if nHigh {
							x = 'T'
						}
						if mHigh {
							y = 'T'
						}
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("SMUL%x%x", x, y),
							Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
						}
					}

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

					return nil
				}
			} else {
				// "4.6.137 SMLABB, SMLABT, SMLATB, SMLATT" of "Thumb-2 Supplement"
				nHigh := opcode&0x0020 == 0x0020
				mHigh := opcode&0x0010 == 0x0010

				return func() *DisasmEntry {
					if arm.decodeOnly {
						x := 'B'
						y := 'B'
						if nHigh {
							x = 'T'
						}
						if mHigh {
							y = 'T'
						}
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("SMLA%x%x", x, y),
							Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", Rd, Rn, Rm, Ra),
						}
					}

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

					result := int64(operand1)*int64(operand2) + int64(arm.state.registers[Ra])
					arm.state.registers[Rd] = uint32(result)
					arm.state.status.saturation = result != result&0xffff

					return nil
				}
			}
		} else {
			panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (32bit multiplies) (%03b) (%04b)", op, op2))
		}
	} else if arm.state.instruction32bitOpcodeHi&0xff80 == 0xfb80 {
		op := (arm.state.instruction32bitOpcodeHi & 0x0070) >> 4
		op2 := (opcode & 0x00f0) >> 4

		if op == 0b010 && op2 == 0b0000 {
			// "4.6.207 UMULL" of "Thumb-2 Supplement"
			RdLo := (opcode & 0xf000) >> 12
			RdHi := Rd

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "UMULL",
						Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", RdLo, RdHi, Rn, Rm),
					}
				}

				result := uint64(arm.state.registers[Rn]) * uint64(arm.state.registers[Rm])
				arm.state.registers[RdHi] = uint32(result >> 32)
				arm.state.registers[RdLo] = uint32(result)

				return nil
			}
		} else if op == 0b011 && op2 == 0b1111 {
			// "4.6.198 UDIV" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "UDIV",
						Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
					}
				}

				// don't allow divide by zero
				if arm.state.registers[Rm] == 0 {
					arm.state.registers[Rd] = 0
				} else {
					arm.state.registers[Rd] = arm.state.registers[Rn] / arm.state.registers[Rm]
				}

				return nil
			}
		} else if op == 0b001 && op2 == 0b1111 {
			// "4.6.126 SDIV" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SDIV",
						Operand:  fmt.Sprintf("R%d, R%d, R%d", Rd, Rn, Rm),
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

				return nil
			}
		} else if op == 0b110 && op2 == 0b0000 {
			// "4.6.206 UMLAL" of "Thumb-2 Supplement"
			RdLo := (opcode & 0xf000) >> 12
			RdHi := Rd

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "UMLAL",
						Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", RdLo, RdHi, Rn, Rm),
					}
				}

				result := uint64(arm.state.registers[Rn]) * uint64(arm.state.registers[Rm])
				result += uint64(arm.state.registers[RdHi] + arm.state.registers[RdLo])
				arm.state.registers[RdHi] = uint32(result >> 32)
				arm.state.registers[RdLo] = uint32(result)

				return nil
			}
		} else if op == 0b000 && op2 == 0b0000 {
			// "4.6.150 SMULL" of "Thumb-2 Supplement"
			RdLo := (opcode & 0xf000) >> 12
			RdHi := Rd

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SMULL",
						Operand:  fmt.Sprintf("R%d, R%d, R%d, R%d", RdLo, RdHi, Rn, Rm),
					}
				}

				result := int64(arm.state.registers[Rn]) * int64(arm.state.registers[Rm])
				arm.state.registers[RdHi] = uint32(result >> 32)
				arm.state.registers[RdLo] = uint32(result)

				return nil
			}
		} else {
			panic(fmt.Sprintf("unimplemented data processing instructions, non immediate (64bit multiplies) (%03b) (%04b)", op, op2))
		}
	}

	panic("reserved data processing instructions, non-immediate")
}

func (arm *ARM) decode32bitThumb2LoadStoreDoubleEtc(opcode uint16) decodeFunction {
	// "3.3.4 Load/store double and exclusive, and table branch" of "Thumb-2 Supplement"

	p := (arm.state.instruction32bitOpcodeHi & 0x0100) == 0x0100
	u := (arm.state.instruction32bitOpcodeHi & 0x0080) == 0x0080
	w := (arm.state.instruction32bitOpcodeHi & 0x0020) == 0x0020
	l := (arm.state.instruction32bitOpcodeHi & 0x0010) == 0x0010

	Rn := arm.state.instruction32bitOpcodeHi & 0x000f
	Rt := (opcode & 0xf000) >> 12
	Rt2 := (opcode & 0x0f00) >> 8
	imm8 := opcode & 0x00ff
	imm32 := imm8 << 2

	if p || w {
		// this function has a lot of branching that could potentially be
		// smoothed out
		return func() *DisasmEntry {
			addr := arm.state.registers[Rn]
			if Rn == rPC {
				addr = AlignTo32bits(addr)
			}

			if p {
				// pre-index addressing
				if u {
					addr += uint32(imm32)
				} else {
					addr -= uint32(imm32)
				}
			}

			if arm.decodeOnly {
				var operator string
				if l {
					// "4.6.50 LDRD (immediate)" of "Thumb-2 Supplement"
					operator = "LDRD"
				} else {
					// "4.6.167 STRD (immediate)" of "Thumb-2 Supplement"
					operator = "STRD"
				}
				e := &DisasmEntry{
					Is32bit:  true,
					Operator: operator,
				}
				e.Operand = fmt.Sprintf("R%d, R%d, [R%d,", Rt, Rt2, Rn)
				if u {
					e.Operand = fmt.Sprintf("%s, #+%d]", e.Operand, imm32)
				} else {
					e.Operand = fmt.Sprintf("%s, #-%d]", e.Operand, imm32)
				}
				if p && w {
					e.Operand = fmt.Sprintf("%s!", e.Operand)
				}
				return e
			}

			if l {
				// "4.6.50 LDRD (immediate)" of "Thumb-2 Supplement"
				arm.state.registers[Rt] = arm.read32bit(addr, true)
				arm.state.registers[Rt2] = arm.read32bit(addr+4, true)
			} else {
				// "4.6.167 STRD (immediate)" of "Thumb-2 Supplement"
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
				// writeback
				arm.state.registers[Rn] = addr
			}

			return nil
		}

	} else if arm.state.instruction32bitOpcodeHi&0x0080 == 0x0080 {
		// "Load and Store Exclusive Byte Halfword, Doubleword, and Table Branch"

		op := (opcode & 0x00f0) >> 4
		Rm := opcode & 0x000f

		switch op {
		case 0b0000:
			// "4.6.188 TBB" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TBB",
						Operand:  fmt.Sprintf("R%d, R%d", Rn, Rm),
					}
				}

				idx := arm.state.registers[Rn] + arm.state.registers[Rm]
				if Rn == rPC || Rm == rPC {
					idx -= 2
				}
				halfwords := arm.read8bit(idx)
				arm.state.registers[rPC] += uint32(halfwords) << 1

				return nil
			}
		case 0b0001:
			// "4.6.189 TBH" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "TBH",
						Operand:  fmt.Sprintf("R%d, R%d, LSL #1", Rn, Rm),
					}
				}

				idx := arm.state.registers[Rn] + (arm.state.registers[Rm] << 1)
				if Rn == rPC || Rm == rPC {
					idx -= 2
				}
				halfwords := arm.read16bit(idx, false)
				arm.state.registers[rPC] += uint32(halfwords) << 1
				return nil
			}
		default:
			panic(fmt.Sprintf("unimplemented load and store double and exclusive and table branch (load and store exclusive byte etc.) (%04b)", op))
		}
	} else {
		// "Load and Store Exclusive"
		panic("unimplemented load and store double and exclusive and table branch (load and store exclusive)")
	}
}

func (arm *ARM) decode32bitThumb2BranchesORDataProcessing(opcode uint16) decodeFunction {
	if opcode&0x8000 == 0x8000 {
		return arm.decode32bitThumb2BranchesORMiscControl(opcode)
	}
	return arm.decode32bitThumb2DataProcessing(opcode)
}

func (arm *ARM) decode32bitThumb2DataProcessing(opcode uint16) decodeFunction {
	// "3.3.1 Data processing instructions: immediate, including bitfield and saturate" of "Thumb-2 Supplement"

	if arm.state.instruction32bitOpcodeHi&0xfa00 == 0xf000 {
		// "Data processing instructions with modified 12-bit immediate"
		// page 3-14 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.instruction32bitOpcodeHi & 0x01e0) >> 5
		setFlags := (arm.state.instruction32bitOpcodeHi & 0x0010) == 0x0010

		Rn := arm.state.instruction32bitOpcodeHi & 0x000f
		Rd := (opcode & 0x0f00) >> 8

		i := (arm.state.instruction32bitOpcodeHi & 0x0400) >> 10
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
				return func() *DisasmEntry {
					// "4.6.192 TST (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "TST",
							Operand:  fmt.Sprintf("R%d, #$%08x", Rn, imm32),
						}
					}

					// always change status register for TST instruction
					result := arm.state.registers[Rn] & imm32
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					// overflow unchanged

					return nil
				}
			} else {
				// "4.6.8 AND (immediate)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("AND%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					// perform AND operation
					arm.state.registers[Rd] = arm.state.registers[Rn] & imm32

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			}

		case 0b0001:
			// "4.6.15 BIC (immediate)" of "Thumb-2 Supplement"
			// T1 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("BIC%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
					}
				}

				// clear bits
				arm.state.registers[Rd] = arm.state.registers[Rn] & ^imm32

				// change status register
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

				return nil
			}

		case 0b0010:
			if Rn == 0xf {
				// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("MOV%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, #$%08x", Rd, imm32),
						}
					}

					// perform mov
					arm.state.registers[Rd] = imm32

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			} else {
				// "4.6.91 ORR (immediate)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("ORR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					// perform OR operation
					arm.state.registers[Rd] = arm.state.registers[Rn] | imm32

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			}

		case 0b0011:
			if Rn == 0b1111 {
				// "4.6.85 MVN (immediate)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("MVN%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, #$%08x", Rd, imm32),
						}
					}

					// perform move
					arm.state.registers[Rd] = ^imm32

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			} else {
				// "4.6.89 ORN (immediate)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("ORN%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					// perform or operation
					arm.state.registers[Rd] = arm.state.registers[Rn] | ^imm32

					// change status register
					if setFlags {
						arm.state.status.isNegative(arm.state.registers[Rd])
						arm.state.status.isZero(arm.state.registers[Rd])
						arm.state.status.setCarry(carry)
						// overflow unchanged
					}

					return nil
				}
			}

		case 0b0100:
			return func() *DisasmEntry {
				result := arm.state.registers[Rn] ^ imm32

				if Rd == 0b1111 && setFlags {
					// "4.6.190 TEQ (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "TEQ",
							Operand:  fmt.Sprintf("R%d, #$%08x", Rn, imm32),
						}
					}
				} else {
					// "4.6.36 EOR (immediate)" of "Thumb-2 Supplement"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("EOR%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					// perform EOR operation
					arm.state.registers[Rd] = result
				}

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					// overflow unchanged
				}

				return nil
			}

		case 0b1000:
			if Rd == 0b1111 {
				// "4.6.27 CMN (immediate)" of "Thumb-2 Supplement"
				// T1 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "CMN",
							Operand:  fmt.Sprintf("R%d, #$%08x", Rn, imm32),
						}
					}

					// perform comparison and change stateus register
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, 0)
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)

					return nil
				}
			} else {
				if arm.state.instruction32bitOpcodeHi&0x100 == 0x100 {
					// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: fmt.Sprintf("ADD%s", setFlagsMnemonic(setFlags)),
								Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
							}
						}

						// perform addition and store result
						result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, 0)
						arm.state.registers[Rd] = result

						// change status register
						if setFlags {
							arm.state.status.isNegative(result)
							arm.state.status.isZero(result)
							arm.state.status.setCarry(carry)
							arm.state.status.setOverflow(overflow)
						}

						return nil
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
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("ADC%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
					}
				}

				// carry value taken from carry bit in status register
				var c uint32
				if arm.state.status.carry {
					c = 1
				}

				// perform addition and store result
				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], imm32, c)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		case 0b1011:
			// "4.6.123 SBC (immediate)" of "Thumb-2 Supplement"
			// T1 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("SBC%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
					}
				}

				// carry value taken from carry bit in status register
				var c uint32
				if arm.state.status.carry {
					c = 1
				}

				// perform subtraction and store result
				result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, c)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		case 0b1101:
			if Rd == 0b1111 {
				// "4.6.29 CMP (immediate)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "CMP",
							Operand:  fmt.Sprintf("R%d, #$%08x", Rn, imm32),
						}
					}

					// perform comparison and store result
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
					arm.state.status.isNegative(result)
					arm.state.status.isZero(result)
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)

					return nil
				}
			} else {
				// "4.6.176 SUB (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: fmt.Sprintf("SUB%s", setFlagsMnemonic(setFlags)),
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					// perform subtraction and store result
					result, carry, overflow := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
					arm.state.registers[Rd] = result

					// change status register
					if setFlags {
						arm.state.status.isNegative(result)
						arm.state.status.isZero(result)
						arm.state.status.setCarry(carry)
						arm.state.status.setOverflow(overflow)
					}

					return nil
				}
			}

		case 0b1110:
			// "4.6.118 RSB (immediate)" of "Thumb-2 Supplement"
			// T2 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: fmt.Sprintf("RSB%s", setFlagsMnemonic(setFlags)),
						Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
					}
				}

				// perform subtraction and store result
				result, carry, overflow := AddWithCarry(^arm.state.registers[Rn], imm32, 1)
				arm.state.registers[Rd] = result

				// change status register
				if setFlags {
					arm.state.status.isNegative(arm.state.registers[Rd])
					arm.state.status.isZero(arm.state.registers[Rd])
					arm.state.status.setCarry(carry)
					arm.state.status.setOverflow(overflow)
				}

				return nil
			}

		default:
			panic(fmt.Sprintf("unimplemented 'data processing instructions with modified 12bit immediate' (%04b)", op))
		}
	} else if arm.state.instruction32bitOpcodeHi&0xfb40 == 0xf200 {
		// "Data processing instructions with plain 12-bit immediate"
		// page 3-15 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.instruction32bitOpcodeHi & 0x0080) >> 7
		op2 := (arm.state.instruction32bitOpcodeHi & 0x0030) >> 4

		Rn := arm.state.instruction32bitOpcodeHi & 0x000f
		Rd := (opcode & 0x0f00) >> 8

		i := (arm.state.instruction32bitOpcodeHi & 0x0400) >> 10
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
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "ADDW",
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					result, _, _ := AddWithCarry(arm.state.registers[Rn], imm32, 0)
					arm.state.registers[Rd] = result
					return nil
				}

			default:
				panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 12bit immediate (%01b) (%02b)'", op, op2))
			}
		case 0b1:
			switch op2 {
			case 0b10:
				// "4.6.176 SUB (immediate)" of "Thumb-2 Supplement"
				// T4 encoding (immediate)
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "SUBW",
							Operand:  fmt.Sprintf("R%d, R%d, #$%08x", Rd, Rn, imm32),
						}
					}

					result, _, _ := AddWithCarry(arm.state.registers[Rn], ^imm32, 1)
					arm.state.registers[Rd] = result
					return nil
				}
			default:
				panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 12bit immediate (%01b) (%02b)'", op, op2))
			}
		}

	} else if arm.state.instruction32bitOpcodeHi&0xfb40 == 0xf240 {
		// "Data processing instructions with plain 16-bit immediate"
		// page 3-15 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.instruction32bitOpcodeHi & 0x0080) >> 7
		op2 := (arm.state.instruction32bitOpcodeHi & 0x0030) >> 4

		i := (arm.state.instruction32bitOpcodeHi & 0x0400) >> 10
		imm4 := arm.state.instruction32bitOpcodeHi & 0x000f
		imm3 := (opcode & 0x7000) >> 12
		Rd := (opcode & 0x0f00) >> 8
		imm8 := opcode & 0x00ff

		imm32 := uint32((imm4 << 12) | (i << 11) | (imm3 << 8) | imm8)

		if op == 0b0 && op2 == 0b00 {
			// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
			// T3 encoding
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "MOVW",
						Operand:  fmt.Sprintf("R%d, #$%08x", Rd, imm32),
					}
				}

				arm.state.registers[Rd] = imm32
				return nil
			}
		} else if op == 0b1 && op2 == 0b00 {
			panic("unimplemented MOVT")
		} else {
			panic(fmt.Sprintf("unimplemented 'data processing instructions with plain 16bit immediate (%01b) (%02b)'", op, op2))
		}

	} else if arm.state.instruction32bitOpcodeHi&0xfb10 == 0xf300 {
		// "Data processing instructions, bitfield and saturate"
		// page 3-16 of "Thumb-2 Supplement" (part of section 3.3.1)

		op := (arm.state.instruction32bitOpcodeHi & 0x00e0) >> 5
		Rn := arm.state.instruction32bitOpcodeHi & 0x000f
		imm3 := (opcode & 0x7000) >> 12
		Rd := (opcode & 0x0f00) >> 8
		imm2 := (opcode & 0x00c0) >> 6

		switch op {
		case 0b010:
			// "4.6.125 SBFX" of "Thumb-2 Supplement"
			widthm1 := opcode & 0x001f
			lsbit := (imm3 << 2) | imm2
			msbit := lsbit + widthm1
			width := widthm1 + 1

			if msbit > 31 {
				panic("invalid SBFX")
			}

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "SBFX",
						Operand:  fmt.Sprintf("R%d, R%d, #%d, #%d", Rd, Rn, lsbit, width),
					}
				}

				arm.state.registers[Rd] = (arm.state.registers[Rn] >> uint32(lsbit)) & ((1 << width) - 1)
				if arm.state.registers[Rd]>>widthm1 == 0x01 {
					arm.state.registers[Rd] = arm.state.registers[Rd] | ^((1 << width) - 1)
				}

				return nil
			}
		case 0b011:
			// "4.6.14 BFI" of "Thumb-2 Supplement"
			msbit := opcode & 0x001f // labelled msb in the instruction specification
			lsbit := (imm3 << 2) | imm2
			width := msbit - lsbit + 1
			maskRemove := uint32(^(((1 << msbit) - 1) << lsbit))
			maskInsert := uint32(((1 << width) - 1) << 1)

			if msbit < lsbit {
				panic("invalid BFI")
			}

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "BFI",
						Operand:  fmt.Sprintf("R%d, R%d, #%d, #%d", Rd, Rn, lsbit, width),
					}
				}

				// remove bits from destination register
				arm.state.registers[Rd] = arm.state.registers[Rd] & maskRemove

				// insert bits from source register
				v := arm.state.registers[Rn] & maskInsert
				arm.state.registers[Rd] = arm.state.registers[Rd] | (v << lsbit)

				return nil
			}

		case 0b110:
			// "4.6.197 UBFX" of "Thumb-2 Supplement"
			Rn := arm.state.instruction32bitOpcodeHi & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm2 := (opcode & 0x00c0) >> 6
			widthm1 := opcode & 0x001f

			lsbit := (imm3 << 2) | imm2
			msbit := lsbit + widthm1
			width := widthm1 + 1

			if msbit > 31 {
				panic("invalid UBFX")
			}

			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "UBFX",
						Operand:  fmt.Sprintf("R%d, R%d, #%d, #%d", Rd, Rn, lsbit, width),
					}
				}

				arm.state.registers[Rd] = (arm.state.registers[Rn] >> uint32(lsbit)) & ((1 << width) - 1)
				return nil
			}
		default:
			panic(fmt.Sprintf("unimplemented 'bitfield operation' (%03b)", op))
		}
	}

	panic("reserved data processing instructions: immediate, including bitfield and saturate")
}

func (arm *ARM) decode32bitThumb2LoadStoreSingle(opcode uint16) decodeFunction {
	// "3.3.3 Load and store single data item, and memory hints" of "Thumb-2 Supplement"

	// Addressing mode discussed in "A4.6.5 Addressing modes" of "ARMv7-M"

	size := (arm.state.instruction32bitOpcodeHi & 0x0060) >> 5
	s := arm.state.instruction32bitOpcodeHi&0x0100 == 0x0100
	l := arm.state.instruction32bitOpcodeHi&0x0010 == 0x0010
	Rn := arm.state.instruction32bitOpcodeHi & 0x000f
	Rt := (opcode & 0xf000) >> 12

	// memmory hints are unimplemented. they occur for load instructions
	// whenever the target register is a PC and the size is 8bit or 16bit
	if Rt == rPC && l && (size == 0b00 || size == 0b01) {
		// panic for now. when we come to implement hints properly, we need to
		// further consider the size, sign extension and also the format group
		// the instruction appears in
		panic("unimplemented memory hint")
	}

	if arm.state.instruction32bitOpcodeHi&0xfe1f == 0xf81f {
		// PC +/ imm12 (format 1 in the table)
		// further depends on size. l is always true

		u := arm.state.instruction32bitOpcodeHi&0x0080 == 0x0080
		imm12 := opcode & 0x0fff

		// all addresses are pre-indexed relative to the PC register and there is no write-back
		preIndex := func() uint32 {
			addr := AlignTo32bits(arm.state.registers[rPC] - 2)
			if u {
				addr += uint32(imm12)
			} else {
				addr -= uint32(imm12)
			}
			return addr
		}

		// indexingSign used in disassembly
		indexingSign := '-'
		if u {
			indexingSign = '+'
		}

		switch size {
		case 0b00:
			if s {
				// "4.6.60 LDRSB (literal)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSB",
							Operand:  fmt.Sprintf("R%d, [PC, #%c%d]", Rt, indexingSign, imm12),
						}
					}

					addr := preIndex()
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					if arm.state.registers[Rt]&0x80 == 0x80 {
						arm.state.registers[Rt] |= 0xffffff00
					}
					return nil
				}
			} else {
				// "4.6.47 LDRB (literal)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRB",
							Operand:  fmt.Sprintf("R%d, [PC, #%c%d]", Rt, indexingSign, imm12),
						}
					}

					addr := preIndex()
					arm.state.registers[Rt] = uint32(arm.read8bit(addr))
					return nil
				}
			}
		case 0b01:
			if s {
				// "4.6.64 LDRSH (literal)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRSH",
							Operand:  fmt.Sprintf("R%d, [PC, #%c%d]", Rt, indexingSign, imm12),
						}
					}

					addr := preIndex()
					if u {
						addr += uint32(imm12)
					} else {
						addr -= uint32(imm12)
					}

					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					if arm.state.registers[Rt]&0x8000 == 0x8000 {
						arm.state.registers[Rt] |= 0xffff0000
					}
					return nil
				}
			} else {
				// "4.6.56 LDRH (literal)" of "Thumb-2 Supplement"
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDRH",
							Operand:  fmt.Sprintf("R%d, [PC, #%c%d]", Rt, indexingSign, imm12),
						}
					}

					addr := preIndex()
					arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
					return nil
				}
			}
		case 0b10:
			// "4.6.44 LDR (literal)" of "Thumb-2 Supplement"
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDR",
						Operand:  fmt.Sprintf("R%d, [PC, #%c%d]", Rt, indexingSign, imm12),
					}
				}

				addr := preIndex()
				arm.state.registers[Rt] = arm.read32bit(addr, false)
				return nil
			}
		default:
			panic(fmt.Sprintf("unimplemented size (%02b) for 'PC +/- imm12'", size))
		}
	} else if arm.state.instruction32bitOpcodeHi&0xfe80 == 0xf880 {
		// Rn + imm12 (format 2 in the table)
		//
		// immediate offset
		//
		// further depends on size and L bit

		// U is always up for this format meaning that we add the index to
		// the base address
		imm12 := opcode & 0x0fff

		// all addresses are pre-indexed and there is no write-back

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSB",
								Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
							}
						}
						addr := arm.state.registers[Rn] + uint32(imm12)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						if arm.state.registers[Rt]&0x80 == 0x80 {
							arm.state.registers[Rt] |= 0xffffff00
						}
						return nil
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRB",
								Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
							}
						}
						addr := arm.state.registers[Rn] + uint32(imm12)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						return nil
					}
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRB",
							Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
						}
					}
					addr := arm.state.registers[Rn] + uint32(imm12)
					arm.write8bit(addr, uint8(arm.state.registers[Rt]))
					return nil
				}
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					// T1 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDSH",
								Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
							}
						}
						addr := arm.state.registers[Rn] + uint32(imm12)
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						if arm.state.registers[Rt]&0x8000 == 0x8000 {
							arm.state.registers[Rt] |= 0xffff0000
						}
						return nil
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRH",
								Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
							}
						}
						addr := arm.state.registers[Rn] + uint32(imm12)
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						return nil
					}
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				// T1 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRH",
							Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
						}
					}
					addr := arm.state.registers[Rn] + uint32(imm12)
					arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
					return nil
				}
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDR",
							Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
						}
					}
					addr := arm.state.registers[Rn] + uint32(imm12)
					arm.state.registers[Rt] = arm.read32bit(addr, false)
					return nil
				}
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STR",
							Operand:  fmt.Sprintf("R%d, [R%d, #+%d]", Rt, Rn, imm12),
						}
					}
					addr := arm.state.registers[Rn] + uint32(imm12)
					arm.write32bit(addr, arm.state.registers[Rt], false)
					return nil
				}
			}
		default:
			panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn + imm12'", size))
		}

	} else if (opcode & 0x0f00) == 0x0c00 {
		// Rn - imm8 (format 3 in the table)
		//
		// negative immediate offset
		//
		// further depends on size and L bit

		imm8 := opcode & 0x00ff

		// all addresses are pre-indexed and there is no write-back

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					// T2 encdoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSB",
								Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
							}
						}
						addr := arm.state.registers[Rn] - uint32(imm8)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						if arm.state.registers[Rt]&0x80 == 0x80 {
							arm.state.registers[Rt] |= 0xffffff00
						}
						return nil
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					// T3 encdoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRB",
								Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
							}
						}
						addr := arm.state.registers[Rn] - uint32(imm8)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						return nil
					}
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				// T3 encdoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRB",
							Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
						}
					}
					addr := arm.state.registers[Rn] - uint32(imm8)
					arm.write8bit(addr, uint8(arm.state.registers[Rt]))
					return nil
				}
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					// T2 encdoding
					return func() *DisasmEntry {
						addr := arm.state.registers[Rn] - uint32(imm8)
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSH",
								Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
							}
						}
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						if arm.state.registers[Rt]&0x8000 == 0x8000 {
							arm.state.registers[Rt] |= 0xffff0000
						}
						return nil
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					// T3 encdoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRH",
								Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
							}
						}
						addr := arm.state.registers[Rn] - uint32(imm8)
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						return nil
					}

				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				// T4 encdoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRH",
							Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
						}
					}
					addr := arm.state.registers[Rn] - uint32(imm8)
					arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
					return nil
				}
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDR",
							Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
						}
					}
					addr := arm.state.registers[Rn] - uint32(imm8)
					arm.state.registers[Rt] = arm.read32bit(addr, false)
					return nil
				}
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STR",
							Operand:  fmt.Sprintf("R%d, [R%d, #-%d]", Rt, Rn, imm8),
						}
					}
					addr := arm.state.registers[Rn] - uint32(imm8)
					arm.write32bit(addr, arm.state.registers[Rt], false)
					return nil
				}
			}
		default:
			panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn - imm8'", size))
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

		postIndex := func(addr uint32) uint32 {
			if u {
				addr += uint32(imm8)
			} else {
				addr -= uint32(imm8)
			}
			return addr
		}

		// indexingSign used in disassembly
		indexingSign := '-'
		if u {
			indexingSign = '+'
		}

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSB",
								Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := arm.state.registers[Rn]
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						if arm.state.registers[Rt]&0x80 == 0x80 {
							arm.state.registers[Rt] |= 0xffffff00
						}
						arm.state.registers[Rn] = postIndex(addr)
						return nil
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRB",
								Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := arm.state.registers[Rn]
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						arm.state.registers[Rn] = postIndex(addr)
						return nil
					}
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRB",
							Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := arm.state.registers[Rn]
					arm.write8bit(addr, uint8(arm.state.registers[Rt]))
					arm.state.registers[Rn] = postIndex(addr)
					return nil
				}
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSH",
								Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := arm.state.registers[Rn]
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						if arm.state.registers[Rt]&0x8000 == 0x8000 {
							arm.state.registers[Rt] |= 0xffff0000
						}
						arm.state.registers[Rn] = postIndex(addr)
						return nil
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRH",
								Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := arm.state.registers[Rn]
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						arm.state.registers[Rn] = postIndex(addr)
						return nil
					}
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRH",
							Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := arm.state.registers[Rn]
					arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
					arm.state.registers[Rn] = postIndex(addr)
					return nil
				}
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDR",
							Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := arm.state.registers[Rn]
					if Rt == rPC {
						arm.state.registers[Rt] = arm.read32bit(addr, false) + 1
					} else {
						arm.state.registers[Rt] = arm.read32bit(addr, false)
					}
					arm.state.registers[Rn] = postIndex(addr)
					return nil
				}
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STR",
							Operand:  fmt.Sprintf("R%d, [R%d], #%c%d", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := arm.state.registers[Rn]
					arm.write32bit(addr, arm.state.registers[Rt], false)
					arm.state.registers[Rn] = postIndex(addr)
					return nil
				}
			}
		default:
			panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn post-index +/- imm8'", size))
		}

	} else if (opcode & 0x0d00) == 0x0d00 {
		// Rn pre-indexed by +/- imm8 (format 6 in the table)
		imm8 := opcode & 0x00ff
		u := (opcode & 0x0200) == 0x0200

		// all addresses are pre-indexed and there is write-back

		preIndex := func(addr uint32) uint32 {
			if u {
				addr += uint32(imm8)
			} else {
				addr -= uint32(imm8)
			}
			return addr
		}

		// indexingSign used in disassembly
		indexingSign := '-'
		if u {
			indexingSign = '+'
		}

		switch size {
		case 0b00:
			if l {
				if s {
					// "4.6.59 LDRSB (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSB",
								Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := preIndex(arm.state.registers[Rn])
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						if arm.state.registers[Rt]&0x80 == 0x80 {
							arm.state.registers[Rt] |= 0xffffff00
						}
						arm.state.registers[Rn] = addr
						return nil
					}
				} else {
					// "4.6.46 LDRB (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRB",
								Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := preIndex(arm.state.registers[Rn])
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						arm.state.registers[Rn] = addr
						return nil
					}
				}
			} else {
				// "4.6.164 STRB (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRB",
							Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := preIndex(arm.state.registers[Rn])
					arm.write8bit(addr, uint8(arm.state.registers[Rt]))
					arm.state.registers[Rn] = addr
					return nil
				}
			}
		case 0b01:
			if l {
				if s {
					// "4.6.63 LDRSH (immediate)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSH",
								Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := preIndex(arm.state.registers[Rn])
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						if arm.state.registers[Rt]&0x8000 == 0x8000 {
							arm.state.registers[Rt] |= 0xffff0000
						}
						arm.state.registers[Rn] = addr
						return nil
					}
				} else {
					// "4.6.55 LDRH (immediate)" of "Thumb-2 Supplement"
					// T3 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRH",
								Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
							}
						}
						addr := preIndex(arm.state.registers[Rn])
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						arm.state.registers[Rn] = addr
						return nil
					}
				}
			} else {
				// "4.6.172 STRH (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRH",
							Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := preIndex(arm.state.registers[Rn])
					arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
					arm.state.registers[Rn] = addr
					return nil
				}
			}
		case 0b10:
			if l {
				// "4.6.43 LDR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDR",
							Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := preIndex(arm.state.registers[Rn])
					arm.state.registers[Rt] = arm.read32bit(addr, false)
					arm.state.registers[Rn] = addr
					return nil
				}
			} else {
				// "4.6.162 STR (immediate)" of "Thumb-2 Supplement"
				// T4 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STR",
							Operand:  fmt.Sprintf("R%d, [R%d, #%c%d]!", Rt, Rn, indexingSign, imm8),
						}
					}
					addr := preIndex(arm.state.registers[Rn])
					arm.write32bit(addr, arm.state.registers[Rt], false)
					arm.state.registers[Rn] = addr
					return nil
				}
			}
		default:
			panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn +/- imm8'", size))
		}

	} else if (opcode & 0x0fc0) == 0x0000 {
		// Rn + shifted register (format 7 in the table)
		shift := (opcode & 0x0030) >> 4
		Rm := opcode & 0x000f

		// all addresses are pre-indexed by a shifted register and there is no write-back

		if l {
			switch size {
			case 0b00:
				if s {
					// "4.6.61 LDRSB (register)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSB",
								Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
							}
						}
						addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						if arm.state.registers[Rt]&0x80 == 0x80 {
							arm.state.registers[Rt] |= 0xffffff00
						}
						return nil
					}
				} else {
					// "4.6.48 LDRB (register)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRB",
								Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
							}
						}
						addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
						arm.state.registers[Rt] = uint32(arm.read8bit(addr))
						return nil
					}
				}
			case 0b01:
				if s {
					// "4.6.65 LDRSH (register)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRSH",
								Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
							}
						}
						addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						if arm.state.registers[Rt]&0x8000 == 0x8000 {
							arm.state.registers[Rt] |= 0xffff0000
						}
						return nil
					}
				} else {
					// "4.6.57 LDRH (register)" of "Thumb-2 Supplement"
					// T2 encoding
					return func() *DisasmEntry {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "LDRH",
								Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
							}
						}
						addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
						arm.state.registers[Rt] = uint32(arm.read16bit(addr, false))
						return nil
					}
				}
			case 0b10:
				// "4.6.45 LDR (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDR",
							Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
						}
					}
					addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
					if Rt == rPC {
						arm.state.registers[rPC] = (arm.read32bit(addr, true) + 2) & 0xfffffffe
					} else {
						arm.state.registers[Rt] = arm.read32bit(addr, false)
					}
					return nil
				}
			default:
				panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn + shifted register' (load)", size))
			}
		} else {
			switch size {
			case 0b00:
				// "4.6.165 STRB (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRB",
							Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
						}
					}
					addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
					arm.write8bit(addr, uint8(arm.state.registers[Rt]))
					return nil
				}
			case 0b01:
				// "4.6.173 STRH (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STRH",
							Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
						}
					}
					addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
					arm.write16bit(addr, uint16(arm.state.registers[Rt]), false)
					return nil
				}
			case 0b10:
				// "4.6.163 STR (register)" of "Thumb-2 Supplement"
				// T2 encoding
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STR",
							Operand:  fmt.Sprintf("R%d, [R%d, R%d, LSL #%d]", Rt, Rn, Rm, shift),
						}
					}
					addr := arm.state.registers[Rn] + (arm.state.registers[Rm] << shift)
					arm.write32bit(addr, arm.state.registers[Rt], false)
					return nil
				}
			default:
				panic(fmt.Sprintf("unimplemented size (%02b) for 'Rn + shifted register' (save)", size))
			}
		}
	}

	panic("unimplemented bit pattern in 'load and store single data item, and memory hints'")
}

func (arm *ARM) decode32bitThumb2LoadStoreMultiple(opcode uint16) decodeFunction {
	// "3.3.5 Load and store multiple, RFE, and SRS" of "Thumb-2 Supplement"
	//		and
	// "A5.3.5 Load Multiple and Store Multiple" of "ARMv7-M"

	op := (arm.state.instruction32bitOpcodeHi & 0x0180) >> 7
	l := (arm.state.instruction32bitOpcodeHi & 0x0010) == 0x0010
	w := (arm.state.instruction32bitOpcodeHi & 0x0020) == 0x0020
	Rn := arm.state.instruction32bitOpcodeHi & 0x000f

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
				regList := opcode & 0xdfff

				return func() *DisasmEntry {
					// Pop multiple registers from the stack
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "POP",
							Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', uint8(regList), "")),
						}
					}

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
					return nil
				}
			default:
				// "4.6.42 LDMIA / LDMFD" of "Thumb-2 Supplement"
				// T2 encoding

				regList := opcode & 0xdfff
				var writebackSign rune
				if w {
					writebackSign = '!'
				}

				// Load multiple (increment after, full descending)
				return func() *DisasmEntry {
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "LDMIA",
							Operand:  fmt.Sprintf("R%d%c, {%s}", Rn, writebackSign, reglistToMnemonic('R', uint8(regList), "")),
						}
					}

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
								panic("LDMIA writeback register is being loaded")
							}
							arm.state.registers[i] = arm.read32bit(addr, true)
							addr += 4
						}
					}

					// write PC
					if regList&0x8000 == 0x8000 {
						arm.state.registers[rPC] = arm.read32bit(addr, true)
					}
					return nil
				}
			}
		} else {
			// "4.6.161 STMIA / STMEA" of "Thumb-2 Supplement"
			// T2 encoding

			regList := opcode & 0x5fff
			var writebackSign rune
			if w {
				writebackSign = '!'
			}

			// Store multiple (increment after, empty ascending)
			return func() *DisasmEntry {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "STMIA",
						Operand:  fmt.Sprintf("R%d%c, {%s}", Rn, writebackSign, reglistToMnemonic('R', uint8(regList), "")),
					}
				}

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
							panic("STMIA writeback register is being stored")
						}

						// there is a branch in the pseudocode that applies to T1
						// encoding only. ommitted here
						arm.write32bit(addr, arm.state.registers[i], true)
						addr += 4
					}
				}

				return nil
			}
		}
	case 0b10:
		if l {
			// "4.6.41 LDMDB / LDMEA" of "Thumb-2 Supplement"

			regList := opcode & 0xdfff
			var writebackSign rune
			if w {
				writebackSign = '!'
			}

			return func() *DisasmEntry {
				// Load multiple (decrement before, empty ascending)
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "LDMDB",
						Operand:  fmt.Sprintf("R%d%c, {%s}", Rn, writebackSign, reglistToMnemonic('R', uint8(regList), "")),
					}
				}

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
							panic("LDMDB writeback register is being loaded")
						}
						arm.state.registers[i] = arm.read32bit(addr, true)
						addr += 4
					}
				}

				// write PC
				if regList&0x8000 == 0x8000 {
					arm.state.registers[rPC] = arm.read32bit(addr, true)
				}
				return nil
			}

		} else {
			switch WRn {
			case 0b11101:
				// "4.6.99 PUSH" of "Thumb-2 Supplement"
				// T2 encoding
				regList := opcode & 0x5fff

				return func() *DisasmEntry {
					// Push multiple registers to the stack
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "PUSH",
							Operand:  fmt.Sprintf("{%s}", reglistToMnemonic('R', uint8(regList), "")),
						}
					}

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
					return nil
				}
			default:
				// "4.6.160 STMDB / STMFD" of "Thumb-2 Supplement"

				regList := opcode & 0x5fff
				var writebackSign rune
				if w {
					writebackSign = '!'
				}

				return func() *DisasmEntry {
					// Store multiple (decrement before, full descending)
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "STMDB",
							Operand:  fmt.Sprintf("R%d%c, {%s}", Rn, writebackSign, reglistToMnemonic('R', uint8(regList), "")),
						}
					}

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
								panic("STMDB writeback register is being stored")
							}
							arm.write32bit(addr, arm.state.registers[i], true)
							addr += 4
						}
					}
					return nil
				}
			}
		}
	}

	panic(fmt.Sprintf("load and store multiple: illegal op (%02b)", op))
}

func (arm *ARM) decode32bitThumb2BranchesORMiscControl(opcode uint16) decodeFunction {
	// "3.3.6 Branches, miscellaneous control instructions" of "Thumb-2 Supplement"

	if arm.state.instruction32bitOpcodeHi&0xffe0 == 0xf3e0 {
		panic("move to register from status")
	} else if arm.state.instruction32bitOpcodeHi&0xfff0 == 0xf3d0 {
		panic("exception return")
	} else if arm.state.instruction32bitOpcodeHi&0xfff0 == 0xf3c0 {
		panic("branch, change to java")
	} else if arm.state.instruction32bitOpcodeHi&0xfff0 == 0xf3b0 {
		panic("special control operations")
	} else if arm.state.instruction32bitOpcodeHi&0xfff0 == 0xf3a0 {
		imodM := (opcode & 0x0700) >> 8
		if imodM == 0b000 {
			panic("NOP, hints")
		} else {
			panic("change processor state")
		}
	} else if arm.state.instruction32bitOpcodeHi&0xffe0 == 0xf380 {
		panic("move to status from register")
	} else if arm.state.instruction32bitOpcodeHi&0xf800 == 0xf000 {
		return arm.decode32bitThumb2Branches(opcode)
	}

	panic("unimplemented branches, miscellaneous control instructions")
}

func (arm *ARM) decode32bitThumb2Branches(opcode uint16) decodeFunction {
	// "3.3.6 Branches, miscellaneous control instructions" of "Thumb-2 Supplement"
	//
	// branches are in the top half of the table and are differentiated by the
	// second half of the instruction (ie. the opcode argument to this
	// function)

	if opcode&0xd000 == 0x8000 {
		// "4.6.12 B" of "Thumb-2 Supplement"
		// T3 encoding
		// Conditional Branch

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.instruction32bitOpcodeHi & 0x0400) >> 10)
		imm6 := uint32(arm.state.instruction32bitOpcodeHi & 0x003f)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x0800) >> 11)
		imm11 := uint32(opcode & 0x07ff)
		imm32 := (s << 20) | (j2 << 19) | (j1 << 18) | (imm6 << 12) | (imm11 << 1)

		// decide on branch offset direction
		if s == 0x01 {
			imm32 |= 0xfff00000
		}

		// condition that must be met before the branch can take place
		cond := (arm.state.instruction32bitOpcodeHi & 0x03c0) >> 6

		// branch target as a string
		operand := arm.branchTargetOffsetFromPC(int64(imm32))

		return func() *DisasmEntry {
			passed, mnemonic := arm.state.status.condition(uint8(cond))

			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: mnemonic,
					Operand:  operand,
				}
			}

			// adjust PC if condition has been met
			if passed {
				arm.state.registers[rPC] += imm32
			}

			return nil
		}

	} else if opcode&0xc000 == 0xc000 {
		// "4.6.18 BL, BLX (immediate)" of "Thumb-2 Supplment"
		// T1 encoding
		// Long Branch With link

		// BL and BLX instructions differ by one bit
		blx := opcode&0x1000 != 0x1000

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.instruction32bitOpcodeHi & 0x400) >> 10)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x800) >> 11)
		i1 := (^(j1 ^ s)) & 0x01
		i2 := (^(j2 ^ s)) & 0x01

		var operator string
		var imm32 uint32

		if blx {
			operator = "BLX"

			imm10H := uint32(arm.state.instruction32bitOpcodeHi & 0x3ff)
			imm10L := uint32(opcode & 0x7ff)

			// immediate 32bit value is sign extended
			imm32 = (s << 23) | (i1 << 22) | (i2 << 21) | (imm10H << 11) | (imm10L << 2)

			// decide on branch offset direction
			if s == 0x01 {
				imm32 |= 0xff000000
			}
		} else {
			operator = "BL"

			imm10 := uint32(arm.state.instruction32bitOpcodeHi & 0x3ff)
			imm11 := uint32(opcode & 0x7ff)

			// immediate 32bit value is sign extended
			imm32 = (s << 24) | (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)

			// decide on branch offset direction
			if s == 0x01 {
				imm32 |= 0xff000000
			}
		}

		// branch target as a string
		operand := arm.branchTargetOffsetFromPC(int64(imm32))

		return func() *DisasmEntry {
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: operator,
					Operand:  operand,
				}
			}

			// record PC in link register
			arm.state.registers[rLR] = (arm.state.registers[rPC]-2)&0xfffffffe | 0x00000001

			// adjust PC
			arm.state.registers[rPC] += imm32
			return nil
		}

	} else if opcode&0xd000 == 0x9000 {
		// "4.6.12 B" of "Thumb-2 Supplement"
		// T4 encoding

		// make sure we're working with 32bit immediate numbers so that we don't
		// drop bits when shifting
		s := uint32((arm.state.instruction32bitOpcodeHi & 0x400) >> 10)
		j1 := uint32((opcode & 0x2000) >> 13)
		j2 := uint32((opcode & 0x800) >> 11)
		i1 := (^(j1 ^ s)) & 0x01
		i2 := (^(j2 ^ s)) & 0x01
		imm10 := uint32(arm.state.instruction32bitOpcodeHi & 0x3ff)
		imm11 := uint32(opcode & 0x7ff)

		// immediate 32bit value is sign extended
		imm32 := (s << 24) | (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)

		// decide on branch offset direction
		if s == 0x01 {
			imm32 |= 0xff000000
		}

		// branch target as a string
		operand := arm.branchTargetOffsetFromPC(int64(imm32))

		return func() *DisasmEntry {
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "B",
					Operand:  operand,
				}
			}

			// adjust PC
			arm.state.registers[rPC] += imm32

			return nil
		}
	}

	panic("unimplemented branches, miscellaneous control instructions")
}
