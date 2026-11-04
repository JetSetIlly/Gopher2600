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

import "math"

func (fpu *FPU) FPSqrt(op uint64, N int) uint64 {
	// page A2-56 of "ARMv7-M"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPSqrt()")
	}

	typ, sign, value := fpu.FPUnpack(op, N, fpu.Status)

	var result uint64

	if typ == FPType_SNaN || typ == FPType_QNaN {
		result = fpu.FPProcessNaN(typ, N, op, fpu.Status)
	} else if typ == FPType_Zero {
		result = fpu.FPZero(sign, N)
	} else if typ == FPType_Infinity && !sign {
		result = fpu.FPInfinity(sign, N)
	} else if sign {
		result = fpu.FPDefaultNaN(N)
		fpu.FPProcessException(FPExc_InvalidOp, fpu.Status)
	} else {
		v := math.Sqrt(value)
		result = fpu.FPRound(v, N, fpu.Status)
	}

	return result
}
