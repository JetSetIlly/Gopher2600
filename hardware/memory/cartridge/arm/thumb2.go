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

// The "ARM Architecture Reference Manual Thumb-2 Supplement" referenced in
// this document can be found at:
//
// https://documentation-service.arm.com/static/5f1066ca0daa596235e7e90a
//
// And the "ARMv7-M Architecture Reference Manual" can be found at:
//
// https://documentation-service.arm.com/static/606dc36485368c4c2b1bf62f

package arm

func (arm *ARM) decodeThumb2(opcode uint16) func(uint16) {
	if opcode&0xe800 == 0xe800 || opcode&0xf000 == 0xf000 || opcode&0xf800 == 0xf800 {
		panic("undecoded 32-bit Thumb-2 instruction")
	}

	panic("undecoded 16-bit Thumb-2 instruction")
}
