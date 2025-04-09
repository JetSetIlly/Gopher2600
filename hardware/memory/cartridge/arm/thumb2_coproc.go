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

// notes in "3.3.7 Coprocessor instructions" of "Thumb-2 Supplement"
//
// • R15 reads as the address of the instruction plus four, rather than the
// address of the instruction plus eight.
//
// • Like all other 32-bit Thumb instructions, the instructions are stored in
// memory in a different byte order from ARM instructions. See Instruction
// alignment and byte ordering on page 2-13 for details.

func (arm *ARM) decodeThumb2Coproc(opcode uint16) decodeFunction {
	// "A5.3.18 Coprocessor instructions" of "ARMv7-M" lists the instructions
	// that are common to all coprocessor types. we're ignoring those for the
	// moment

	// the coprocessor type is encoded in the lower 16 bits of the instruction
	coproc := (opcode & 0xf00) >> 8

	// only supporting the FPU for now
	switch coproc {
	case 0b1010:
		fallthrough
	case 0b1011:
		return arm.decodeThumb2FPU(opcode)
	}

	panic(fmt.Sprintf("unsupported coproc (%04b)", coproc))
}
