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

// Generation of specific floating-point values
//
// The following functions generate specific floating-point values. The sign
// argument of FPZero() , FPMaxNormal() , and FPInfinity() is '0' for the
// positive version and '1' for the negative version.

func (fpu *FPU) FPZero(sign bool, N int) uint64 {
	// page A2-46 of "ARMv7-M"

	// bits(N) FPZero(bit sign, integer N)
	//		assert N IN {16,32,64};
	//		if N == 16 then
	//			E = 5;
	//		elsif N == 32 then
	//			E = 8;
	//		else E = 11;
	//		F = N - E - 1;
	//		exp = Zeros(E);
	//		frac = Zeros(F);
	//		return sign:exp:frac;

	if !sign {
		return 0
	}

	switch N {
	case 16:
		return 0xfffffffffff8000
	case 32:
		return 0xfffffff80000000
	case 64:
		return 0x800000000000000
	}

	panic("unsupported number of bits in FPZero()")
}

func (fpu *FPU) FPInfinity(sign bool, N int) uint64 {
	// page A2-46 of "ARMv7-M"

	var E int

	switch N {
	case 16:
		E = 5
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in FPInfinity()")
	}

	F := N - E - 1
	exp := uint64((1 << E) - 1)
	var S uint64
	if sign {
		S = uint64(1<<(E+F)-1) ^ 0xffffffffffffffff
	}
	return S | (exp << F)
}

func (fpu *FPU) FPMaxNormal(sign bool, N int) uint64 {
	// page A2-47 of "ARMv7-M"

	var E int

	switch N {
	case 16:
		E = 5
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in FPMaxNormal()")
	}

	F := N - E - 1
	exp := uint64((1<<(E-1))-1) << 1
	frac := uint64((1 << F) - 1)
	var S uint64
	if sign {
		S = uint64(1<<(E+F)-1) ^ 0xffffffffffffffff
	}
	return S | (exp << F) | frac
}

func (fpu *FPU) FPDefaultNaN(N int) uint64 {
	// page A2-47 of "ARMv7-M"

	var E int

	switch N {
	case 16:
		E = 5
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported numbers of bits in FPDefaultNaN()")
	}

	F := N - E - 1

	exp := uint64((1 << E) - 1)
	frac := uint64(1 << (F - 1))
	return (exp << F) | frac
}
