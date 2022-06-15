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

func (arm *ARM) decodeThumb2Upper32bit(opcode uint16) func(uint16) {
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
		return func(_ uint16) {
			arm.function32bit = true
			arm.function32bitFunction = arm.thumb2BranchesORDataProcessing
			arm.function32bitOpcode = opcode
		}
	} else if opcode&0xfe40 == 0xe800 {
		// load and store multiple, RFE and SRS
		return func(_ uint16) {
			arm.function32bit = true
			arm.function32bitFunction = arm.thumb2LoadStoreMultiple
			arm.function32bitOpcode = opcode
		}
	} else if opcode&0xfe40 == 0xe840 {
		// load and store double and exclusive and table branch
		panic("load and store double and exclusive and table branch")
	} else if opcode&0xfe00 == 0xf800 {
		// load and store single data item, memory hints
		return func(_ uint16) {
			arm.function32bit = true
			arm.function32bitFunction = arm.thumb2LoadStoreSingle
			arm.function32bitOpcode = opcode
		}
	} else if opcode&0xee00 == 0xea00 {
		// data processing, no immediate operand
		panic("data processing, no immediate operand")
	}

	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (upper half-word) (%04x)", opcode))
}

func (arm *ARM) thumb2BranchesORDataProcessing(opcode uint16) {
	if opcode&0x8000 == 0x8000 {
		arm.thumb2LongBranchWithLink(opcode)
	} else {
		arm.thumb2DataProcessing(opcode)
	}
}

func (arm *ARM) thumb2DataProcessing(opcode uint16) {
	// "A5.3.1 Data processing (modified immediate)" of "ARMv7-M"

	if arm.function32bitOpcode&0xf200 == 0xf000 {
		// Data processing, modified 12-bit immediate

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
				// "A7.7.188 TST (immediate)" of "ARMv7-M"
				arm.fudge_thumb2Disassemble = "TST"

				result := arm.registers[Rn] & imm32
				if setFlags {
					arm.Status.isNegative(result)
					arm.Status.isZero(result)
					arm.Status.setCarry(carry)
				}
			} else {
				// "A7.7.8 AND (immediate)" of "ARMv7-M"
				arm.fudge_thumb2Disassemble = "AND"

				arm.registers[Rd] = arm.registers[Rn] & imm32
				if setFlags {
					arm.Status.isNegative(arm.registers[Rd])
					arm.Status.isZero(arm.registers[Rd])
					arm.Status.setCarry(carry)
				}
			}
		case 0b0010:
			if Rn == 0xf {
				// "4.6.76 MOV (immediate)" of "Thumb-2 Supplement"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "MOV (immediate)"

				if Rn != 0x000f {
					panic("Rn register must be 0b1111 for MOV immediate")
				}

				if Rd == rSP || Rd == rPC {
					panic(fmt.Sprintf("MOV using %d register and will be unpredictable", Rd))
				}

				imm3 := (opcode & 0x7000) >> 12
				imm8 := opcode & 0x00ff
				imm12 := (i << 11) | (imm3 << 8) | imm8
				imm32, carry := ThumbExpandImm_C(uint32(imm12), arm.Status.carry)
				arm.registers[Rd] = imm32

				if setFlags {
					arm.Status.isNegative(arm.registers[Rn])
					arm.Status.isZero(arm.registers[Rn])
					arm.Status.setCarry(carry)
				}

			} else {
				panic(fmt.Sprintf("unimplemented 'data processing instructions with modified 12-bit immediate' (%04b)", op))
			}

		case 0b1000:
			if arm.function32bitOpcode&0x100 == 0x100 {
				// "4.6.3 ADD (immediate)" of "Thumb-2 Supplement"
				// T3 encoding
				arm.fudge_thumb2Disassemble = "ADD (immediate)"

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
				// CMP
				// "4.6.29 CMP (immediate)" of "Thumb-2 Supplement"
				result, carry, overflow := AddWithCarry(arm.registers[Rn], ^imm32, 1)
				arm.Status.isNegative(result)
				arm.Status.isZero(result)
				arm.Status.setCarry(carry)
				arm.Status.setOverflow(overflow)
			} else {
				// SUB
				panic("sub")
			}

		default:
			fmt.Printf("%04x %04x\n", arm.function32bitOpcode, opcode)
			panic(fmt.Sprintf("unimplemented 'data processing instructions with modified 12-bit immediate' (%04b)", op))
		}
	} else {
		if arm.function32bitOpcode&0xf300 == 0xf300 {
			if arm.function32bitOpcode&0xf320 == 0xf320 {
				panic("reserved data processing instruction")
			}

			// bitfield operations
			op := (arm.function32bitOpcode & 0x00e0) >> 5
			switch op {
			case 0b0110:
				// // "4.6.197 UBFX" of "Thumb-2 Supplement"
				arm.fudge_thumb2Disassemble = "UBFX"

				Rn := arm.function32bitOpcode & 0x000f
				imm3 := (opcode & 0x7000) >> 12
				Rd := (opcode & 0x0f00) >> 8
				imm2 := (opcode & 0x00c0) >> 6
				widthm1 := opcode & 0x001f

				lsbit := (imm3 << 2) | imm2
				msbit := lsbit + widthm1
				if msbit <= 31 {
					v := arm.registers[Rn]
					w := v >> lsbit
					v = w << lsbit
					x := v << (31 - msbit)
					v = x >> (31 - msbit)
					arm.registers[Rd] = v
				}
			default:
				panic(fmt.Sprintf("unimplemented 'bitfield operation' (%03b)", op))
			}
		} else if arm.function32bitOpcode&0xf240 == 0xf240 {
			// "A7.7.76 MOV (immediate)" of "ARMv7-M"
			// T3 encoding
			arm.fudge_thumb2Disassemble = "MOV"

			i := (arm.function32bitOpcode & 0x0400) >> 10
			imm4 := arm.function32bitOpcode & 0x000f
			imm3 := (opcode & 0x7000) >> 12
			Rd := (opcode & 0x0f00) >> 8
			imm8 := opcode & 0x00ff

			imm32 := uint32((imm4 << 12) | (i << 11) | (imm3 << 8) | imm8)
			arm.registers[Rd] = imm32

		} else {
			panic("(2) unimplemented data processing instruction")
		}
	}
}

func (arm *ARM) thumb2LoadStoreSingle(opcode uint16) {
	// "3.3.3 Load and store single data item, and memory hints" of "Thumb-2 Supplement"

	size := (arm.function32bitOpcode & 0x0060) >> 5
	s := arm.function32bitOpcode&0x0100 == 0x0100
	l := arm.function32bitOpcode&0x0010 == 0x0010
	Rn := arm.function32bitOpcode & 0x000f
	Rt := (opcode & 0xf000) >> 12
	imm12 := opcode & 0x0fff

	if s {
		panic("unhandled sign extend for 'load and store single data item, and memory hints'")
	}

	if arm.function32bitOpcode&0xfe1f == 0xf81f {
		// PC +/ imm12 (format 1 in the table)
		// further depends on size and L bit

		u := arm.function32bitOpcode&0x0080 == 0x0080
		addr := arm.registers[Rn]
		if u {
			addr += uint32(imm12)
		} else {
			addr -= uint32(imm12)
		}
		addr &= 0xfffffffe

		switch size {
		case 0b00:
			if l {
				// "A7.7.46 LDRB (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "LDRB (immediate - pc relative)"

				if Rt == rPC {
					panic("PC cannot be a destination register for this instruction")
				}
				arm.registers[Rt] = uint32(arm.read8bit(addr))
			} else {
				// "A7.7.163 STRB (immediate)" of "ARMv7-M
				// T2 encoding
				arm.fudge_thumb2Disassemble = "STRB (immediate - pc relative)"

				if Rt == rPC || Rt == rSP {
					panic("PC/SP cannot be a destination register for this instruction")
				}
				arm.write8bit(addr, uint8(arm.registers[Rt]))
			}
			return

		case 0b01:
		case 0b10:
			if l {
				// "A7.7.55 LDRH (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "LDRH (immediate - pc relative)"

				arm.registers[Rt] = uint32(arm.read16bit(addr))
			} else {
				// "A7.7.170 STRH (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "STRH (immediate - pc relative)"

				arm.write16bit(addr, uint16(arm.registers[Rt]))
			}
			return
		}
	} else if arm.function32bitOpcode&0xfe80 == 0xf880 {
		// Rn + imm12 (format 2 in the table)
		// further depends on size and L bit

		// U is always up for this format meaning that we add the index to
		// the base address
		addr := arm.registers[Rn] + uint32(imm12)

		switch size {
		case 0b00:
			if l {
				// "A7.7.46 LDRB (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "LDRB (immediate)"

				if Rt == rPC {
					panic("PC cannot be a destination register for this instruction")
				}
				arm.registers[Rt] = uint32(arm.read8bit(addr))
			} else {
				// "A7.7.163 STRB (immediate)" of "ARMv7-M
				// T2 encoding
				arm.fudge_thumb2Disassemble = "STRB (immediate)"

				if Rt == rPC || Rt == rSP {
					panic("PC/SP cannot be a destination register for this instruction")
				}
				arm.write8bit(addr, uint8(arm.registers[Rt]))
			}
			return

		case 0b01:
		case 0b10:
			if l {
				// "A7.7.55 LDRH (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "LDRH (immediate)"

				arm.registers[Rt] = uint32(arm.read16bit(addr))
			} else {
				// "A7.7.170 STRH (immediate)" of "ARMv7-M"
				// T2 encoding
				arm.fudge_thumb2Disassemble = "STRH (immediate)"

				arm.write16bit(addr, uint16(arm.registers[Rt]))
			}
			return
		}

		panic(fmt.Sprintf("unhandled size (%02b) for 'load and store single data item, and memory hints'", size))
	} else {
		// size := (arm.function32bitOpcode & 0x0060) >> 5

		if (opcode & 0x0f00) == 0x0c00 {
			panic("umimplemented Rn -imm8")
		} else if (opcode & 0x0f00) == 0x0e00 {
			panic("umimplemented Rn +imm8, user privilege")
		} else if (opcode & 0x0d00) == 0x0900 {
			panic("umimplemented Rn post-indexed by +/- imm8")
		} else if (opcode & 0x0d00) == 0x0d00 {
			panic("umimplemented Rn pre-indexes by +/- imm8")
		}
	}

	panic("reserved bit pattern in 'load and store single data item, and memory hints'")
}

func (arm *ARM) thumb2LoadStoreSingle_(opcode uint16) {
	// "3.3.3 Load and store single data item, and memory hints" of "Thumb-2 Supplement"
	//
	// the equivalent tables in "ARMv7-M" are more plentiful but ulimately, include the
	// same information. The "Thumb-2 Supplement" was used.

	// load (1) or store (0)
	l := (arm.function32bitOpcode & 0x0010) >> 4

	// sign-extended (1) or zero extended (0)
	// s := (arm.function32bitOpcode & 0x0100) >> 8

	// size of transfer
	size := (arm.function32bitOpcode & 0x0060) >> 5

	if arm.function32bitOpcode&0x001f == 0x001f {
		// PC +/- imm12

		// indexing upwards (1) or downwards (0)
		// u := (arm.function32bitOpcode & 0x0080) >> 7

		panic("load and store single: unimplemented: PC +/- imm12")
	} else if arm.function32bitOpcode&0x0080 == 0x0080 {
		// "A7.7.43 LDR (immediate)" of "ARMv7-M"
		// T3 Encoding

		// Rn +/- imm12

		// Rn cannot be 15. if it was it would be "PC +/- imm 12" version of
		// the instruction
		Rn := arm.function32bitOpcode & 0x000f
		Rt := (opcode & 0xf000) >> 12
		Imm32 := uint32(opcode & 0x0fff)

		// if Rt is PC and we're in IT block but this isn't the last
		// instruction in the block then the results are unpredictable

		if l == 1 {
			// load
			switch size {
			case 0b00:
				panic("load and store single: unimplemented: Rn +/- imm12: load (size 00)")
			case 0b01:
				panic("load and store single: unimplemented: Rn +/- imm12: load (size 01)")
			case 0b10:
				// "A7.7.43 LDR (immediate)" of "ARMv7-M"
				arm.fudge_thumb2Disassemble = "LDR (immediate)"

				addr := arm.registers[Rn] + Imm32
				data := arm.read32bit(addr)
				arm.registers[Rt] = data

			case 0b11:
				panic("load and store single: unimplemented: Rn +/- imm12: load (size 11)")
			}
		} else {
			// store
			panic("load and store single: unimplemented: Rn +/- imm12: store")
		}
	} else {
		if opcode&0x0f00 == 0xc00 {
			// Rn -imm8
			panic("load and store single: unimplemented: Rn -imm8")
		} else if opcode&0x0f00 == 0x0e00 {
			// Rn + imm8, user privilege
			panic("load and store single: unimplemented: Rn + imm8, user privilege")
		} else if opcode&0x0d00 == 0x0900 {
			// Rn post-indexed by += imm8
			panic("load and store single: unimplemented: Rn post-indexed by += imm8")
		} else if opcode&0x0d00 == 0x0d00 {
			// Rn pre-indexed by += imm8
			panic("load and store single: unimplemented: Rn pre-indexed by += imm8")
		} else if opcode&0x0fc0 == 0x0000 {
			// Rn + shifted reister
			panic("load and store single: unimplemented: Rn + shifted reister")
		} else {
			panic("reserved operation in 'load and store single'")
		}
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
				arm.fudge_thumb2Disassemble = "POP (ldmia)"

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
				arm.fudge_thumb2Disassemble = "PUSH (stmdb)"

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

func (arm *ARM) thumb2LongBranchWithLink(opcode uint16) {
	// details in "A7.7.18 BL" of "ARMv7-M"

	arm.registers[rLR] = (arm.registers[rPC]-2)&0xfffffffe | 0x00000001

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

	arm.fudge_thumb2Disassemble = "BL"
}
