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

	// number of exponent bits
	var E int
	switch N {
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in VFPExpandImm()")
	}

	// number of fraction bits
	F := N - E - 1

	// NOT(imm8<6>):Replicate(imm8<6>, E-3):imm8<5:4>
	bit6 := (imm8 & 0x40) >> 6
	exp := uint64(^bit6) << E
	for i := 0; i < E-3; i++ {
		exp |= uint64(bit6) << (E - 2 - i)
	}
	exp |= uint64((imm8 & 0x30) >> 4)

	// imm8<3:0>:Zeros(F-4)
	frac := uint64(imm8&0x0f) << (F - 4)

	sign := (imm8 >> 7) & 0x1
	return (uint64(sign) << (N - 1)) | (exp << F) | frac
}
