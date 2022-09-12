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

func (arm *ARM) thumb2Coprocessor(opcode uint16) {
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

	// op := (opcode & 0x0010) >> 4
	// op1 := (arm.state.function32bitOpcode & 0x03f0) >> 4
	// coproc := (opcode & 0xf00) >> 8
	// fmt.Printf("coproc: %04b op: %01b op1: %06b\n", coproc, op, op1)

	// temporary disasm tag
	arm.state.fudge_thumb2disassemble32bit = "coproc"
}
