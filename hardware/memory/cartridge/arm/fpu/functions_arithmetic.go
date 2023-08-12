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
		panic("unsupported number of bits in FPAddr()")
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
				resultSign := fpscr.RMode() == FPRoundNegInf
				result = fpu.FPZero(resultSign, N)
			} else {
				result = fpu.FPRound(resultValue, N, fpscr)
			}
		}
	}

	return result
}

func (fpu *FPU) FPSub(op1 uint64, op2 uint64, N int, fpscrControlled bool) uint64 {
	// page A2-54 of "ARMv7-M"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPSub()")
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

		if inf1 && inf2 && sign1 == sign2 {
			result = fpu.FPDefaultNaN(N)
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)
		} else if (inf1 && !sign1) || (inf2 && sign2) {
			result = fpu.FPInfinity(false, N)
		} else if (inf1 && sign1) || (inf2 && !sign2) {
			result = fpu.FPInfinity(true, N)
		} else if zero1 && zero2 && sign1 != sign2 {
			result = fpu.FPZero(sign1, N)
		} else {
			resultValue := value1 - value2
			if resultValue == 0.0 {
				resultSign := fpscr.RMode() == FPRoundNegInf
				result = fpu.FPZero(resultSign, N)
			} else {
				result = fpu.FPRound(resultValue, N, fpscr)
			}
		}
	}

	return result
}

func (fpu *FPU) FPMul(op1 uint64, op2 uint64, N int, fpscrControlled bool) uint64 {
	// page A2-54 to A2-55 of "ARMv7-M"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPMul()")
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

		if (inf1 && zero2) || (zero1 && inf2) {
			result = fpu.FPDefaultNaN(N)
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)
		} else if inf1 || inf2 {
			resultSign := sign1 != sign2
			result = fpu.FPInfinity(resultSign, N)
		} else if zero1 || zero2 {
			resultSign := sign1 != sign2
			result = fpu.FPZero(resultSign, N)
		} else {
			result = fpu.FPRound(value1*value2, N, fpscr)
		}
	}

	return result
}

// The VNMLA, VNMLS, VNMUL group of FPU instructions use this to control operation
type VFPNegMul int

const (
	VFPNegMul_VNMLA VFPNegMul = iota
	VFPNegMul_VNMLS
	VFPNegMul_VNMNUL
)

func (fpu *FPU) FPMulAdd(addend uint64, op1 uint64, op2 uint64, N int, fpscrControlled bool) uint64 {
	// page A2-55 to A2-56 of "ARMv7-M"
	//
	// "The FPMulAdd() function performs the calculation A*B+C with only a single rounding step, and so provides greater
	// accuracy than performing the multiplication followed by an add"

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FPMulAdd()")
	}

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}

	typA, signA, valueA := fpu.FPUnpack(addend, N, fpscr)
	typ1, sign1, value1 := fpu.FPUnpack(op1, N, fpscr)
	typ2, sign2, value2 := fpu.FPUnpack(op2, N, fpscr)

	inf1 := typ1 == FPType_Infinity
	inf2 := typ2 == FPType_Infinity
	zero1 := typ1 == FPType_Zero
	zero2 := typ2 == FPType_Zero

	done, result := fpu.FPProcessNaNs3(typA, typ1, typ2, N, addend, op1, op2, fpscr)

	if typA == FPType_QNaN && ((inf1 && zero2) || (zero1 && inf2)) {
		result = fpu.FPDefaultNaN(N)
		fpu.FPProcessException(FPExc_InvalidOp, fpscr)
	}

	if !done {
		infA := typA == FPType_Infinity
		zeroA := typA == FPType_Zero

		// "Determine sign and type product will have if it does not cause an Invalid Operation"
		signP := sign1 == sign2
		infP := inf1 || inf2
		zeroP := zero1 || zero2

		// "Non SNaN-generated Invalid Operation cases are multiplies of zero by infinity and
		// additions of opposite-signed infinities"
		if (inf1 && zero2) || (zero1 && inf2) || (infA && infP && signA != signP) {
			result = fpu.FPDefaultNaN(N)
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)

			// "Other cases involving infinities produce an infinity of the same sign"
		} else if (infA && !signA) || (infP && !signP) {
			result = fpu.FPInfinity(false, 32)
		} else if (infA && signA) || (infP && signP) {
			result = fpu.FPInfinity(true, 32)

			// "Cases where the result is exactly zero and its sign is not determined by the
			// rounding mode are addition of the same-signed zeros"
		} else if zeroA && zeroP && signA == signP {
			result = fpu.FPZero(signA, N)

			// Otherwise calculate numerical result and round it
		} else {
			resultValue := value1 * value2
			resultValue += valueA
			if resultValue == 0.0 {
				resultSign := fpscr.RMode() == FPRoundNegInf
				result = fpu.FPZero(resultSign, N)
			} else {
				result = fpu.FPRound(resultValue, N, fpscr)
			}
		}
	}

	return result
}
