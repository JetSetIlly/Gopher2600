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

import "fmt"

func decodeThumb2Coprocessor(arm *ARM, opcode uint16) *DisasmEntry {
	// the normal method for dividing the opcode for a thumb instruction is not
	// used for the coprocesor group of instructions.
	//
	// for reference, the following code block outlines the first layer of
	// triage if we did follow the same method. from "3.3.7 Coprocessor
	// instructions" of "Thumb-2 Supplement"
	//
	// if arm.state.function32bitOpcode&0xef00 == 0xef00 {
	// 	panic("reserved for AdvSIMD")
	// } else if arm.state.function32bitOpcode&0xef00 == 0xee00 {
	// 	if opcode&0x0010 == 0x0010 {
	// 		panic("MRC/MCR coproc register transfers")
	// 	} else {
	// 		panic("coproc data processing")
	// 	}
	// } else if arm.state.function32bitOpcode&0xee00 == 0xec00 {
	// 	panic("load/store coproc")
	// } else if arm.state.function32bitOpcode&0xefe0 == 0xec40 {
	// 	panic("MRRC/MCRR coproc register transfers")
	// } else {
	// 	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (coprocessor) (%04x)", opcode))
	// }

	// the reason we're eschewing the normal method is because it is not clear
	// how the opcodes are further subdivided. better instead to use the triage
	// as outlined in "A5.3.18 Coprocessor instructions" of "ARMv7-M".

	// before ploughing on though we should note the words of advice from the
	// notes in "3.3.7 Coprocessor instructions" of "Thumb-2 Supplement"
	//
	// • R15 reads as the address of the instruction plus four, rather than the
	// address of the instruction plus
	// eight.
	//
	// • Like all other 32-bit Thumb instructions, the instructions are stored in
	// memory in a different byte order from ARM instructions. See Instruction
	// alignment and byte ordering on page 2-13 for details.

	// only supporting the FPU for now. the remainder of this function
	// concentrates on the following text:
	//
	// "Chapter A6 The Floating-point Instruction Set Encoding" of "ARMv7-M"
	coproc := (opcode & 0xf00) >> 8

	// the coproc value for the FPU is either 1010 or 1011. panic for any other value
	if coproc&0b1110 != 0b1010 {
		panic(fmt.Sprintf("unsupported coproc (%04b)", coproc))
	}

	// T bit
	// T := (arm.state.function32bitOpcode & 0x1000) >> 12

	if arm.state.function32bitOpcodeHi&0x0fe0 == 0x0c44 {
		// "A6.7 64-bit transfers between Arm core and extension registers" of "ARMv7-M"
		panic("unimplemented FPU 64bit transfer instruction")
	} else if arm.state.function32bitOpcodeHi&0x0f00 == 0x0e00 && opcode&0x0010 == 0x0010 {
		// "A6.6 32-bit transfer between ARM core and extension registers" of "ARM-v7-M"
		L := (arm.state.function32bitOpcodeHi & 0x0010) >> 4
		A := (arm.state.function32bitOpcodeHi & 0x00e0) >> 5
		// C := (opcode & 0x100) >> 8

		if A == 0b111 {
			panic("unimplemented 32-bit transfer between ARM core and extension registers")
		} else {
			// "A7.7.243 VMOV (between Arm core register and single-precision register)" of "ARMv7-M"
			arm.state.fudge_thumb2disassemble32bit = "VMOV"

			// Vn := arm.state.function32bitOpcode & 0x000f
			Rt := (opcode & 0xf000) >> 12
			// N := (opcode && 0x0080) >> 7
			// Sn := (Vn << 1) | N

			// L is labelled op in the actual instruction defintion
			if L == 0b1 {
				// this should be copying the Sn "scalar" register to the
				// ARM register. for now we'll just copy a zero value
				arm.state.registers[Rt] = 0
			} else {
				// this should be copying the Rt ARM register to the Sn
				// "scalar" register. for now we'll do nothing
			}
		}

	} else if arm.state.function32bitOpcodeHi&0x0f00 == 0x0e00 && opcode&0x0010 != 0x0010 {
		// "A6.4 Floating-point data-processing instructions" of "ARMv7-M"

		// assume everything will just result in zero values for now
	} else if arm.state.function32bitOpcodeHi&0x0c00 == 0x0c00 {
		// "A6.5 Extension register load or store instructions" of "ARMv7-M"
		opcode := (arm.state.function32bitOpcodeHi & 0x01f0) >> 4
		Rn := arm.state.function32bitOpcodeHi & 0x000f

		switch opcode & 0b11011 {
		case 0b10010:
			switch Rn {
			case 0b1101:
				arm.state.fudge_thumb2disassemble32bit = "VPUSH"
			default:
				panic("unimplemented FPU register load/save instruction (VSTM)")
			}
		case 0b10001:
			fallthrough
		case 0b11001:
			arm.state.fudge_thumb2disassemble32bit = "VLDR"
		case 0b01011:
			switch Rn {
			case 0b1101:
				panic("unimplemented FPU register load/save instruction (VPOP)")
			default:
				arm.state.fudge_thumb2disassemble32bit = "VLDM"
			}
		default:
			panic("unimplemented FPU register load/save instruction")
		}
	} else {
		panic("undecoded FPU instruction")
	}

	return nil
}
