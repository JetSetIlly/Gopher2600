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

func (fpu *FPU) FPDiv(op1 uint64, op2 uint64, N int, fpscrControlled bool) uint64 {
	// page A2-55 of "ARMv7-M"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPDiv()")
	}

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}

	typ1, sign1, value1 := fpu.FPUnpack(op1, N, fpscr)
	typ2, sign2, value2 := fpu.FPUnpack(op2, N, fpscr)
	done, result := fpu.FPProcessNaNs(typ1, typ2, N, op1, op2, fpscr)

	if !done {
		inf1 := typ1 == FPType_Infinity
		inf2 := typ2 == FPType_Infinity
		zero1 := typ1 == FPType_Zero
		zero2 := typ2 == FPType_Zero

		if (inf1 && inf2) || (zero1 && zero2) {
			result = fpu.FPDefaultNaN(N)
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)
		} else if inf1 || zero2 {
			resultSign := sign1 != sign2
			result = fpu.FPInfinity(resultSign, N)
			if !inf1 {
				fpu.FPProcessException(FPExc_DivideByZero, fpscr)
			}
		} else if zero1 || inf2 {
			resultSign := sign1 != sign2
			result = fpu.FPZero(resultSign, N)
		} else {
			result = fpu.FPRound(value1/value2, N, fpscr)
		}
	}

	return result
}

func (fpu *FPU) FPAdd(op1 uint64, op2 uint64, N int, fpscrControlled bool) uint64 {
	// page A2-54 of "ARMv7-M"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPDiv()")
	}

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}

	typ1, sign1, value1 := fpu.FPUnpack(op1, N, fpscr)
	typ2, sign2, value2 := fpu.FPUnpack(op2, N, fpscr)
	done, result := fpu.FPProcessNaNs(typ1, typ2, N, op1, op2, fpscr)

	if !done {
		inf1 := typ1 == FPType_Infinity
		inf2 := typ2 == FPType_Infinity
		zero1 := typ1 == FPType_Zero
		zero2 := typ2 == FPType_Zero

		if inf1 && inf2 && sign1 == !sign2 {
			result = fpu.FPDefaultNaN(N)
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)
		} else if (inf1 && !sign1) || (inf2 && !sign2) {
			result = fpu.FPInfinity(false, N)
		} else if (inf1 && sign1) || (inf2 && sign2) {
			result = fpu.FPInfinity(true, N)
		} else if zero1 && zero2 && sign1 == sign2 {
			result = fpu.FPZero(sign1, N)
		} else {
			resultValue := value1 + value2
			if resultValue == 0.0 {
				resultSign := fpu.Status.RMode() == FPRoundNegInf
				result = fpu.FPZero(resultSign, N)
			} else {
				result = fpu.FPRound(resultValue, N, fpscr)
			}
		}
	}

	return result
}
