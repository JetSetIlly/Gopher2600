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

package fpu

func (fpu *FPU) VFPExpandImm(imm8 uint8, N int) uint64 {
	// on pages A6-166 of "ARMv7-M"

	var E int

	switch N {
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in VFPExpandImm()")
	}

	F := N - E - 1

	imm64 := uint64(imm8)

	sign := (imm64 & 0x80) >> 7
	bit6 := (imm64 & 0x40) >> 6
	expA := (^bit6) & 0b01
	expB := uint64(0)
	if bit6 == 0b01 {
		expB = (bit6 << (E - 3)) - 1
	}
	expC := (imm64 & 0x30) >> 4
	exp := (expA << (E - 1)) | (expB << 2) | expC
	frac := (imm64 & 0x0f) << (F - 4)

	return (sign << (E + F)) | (exp << F) | frac
}
