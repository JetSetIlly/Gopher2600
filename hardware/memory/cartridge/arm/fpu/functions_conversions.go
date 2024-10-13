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

func (fpu *FPU) FixedToFP(operand uint64, N int, fractionBits int, unsigned bool, nearest bool, fpscrControlled bool) uint64 {
	// page A2-59 of "ARMv7-M"

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}
	if nearest {
		fpscr.SetRMode(FPRoundNearest)
	}

	var realOperand float64

	switch N {
	case 32:
		if unsigned {
			realOperand = float64(uint32(operand) / uint32(math.Pow(2, float64(fractionBits))))
		} else {
			realOperand = float64(int32(operand) / int32(math.Pow(2, float64(fractionBits))))
		}
	case 64:
		if unsigned {
			realOperand = float64(operand / uint64(math.Pow(2, float64(fractionBits))))
		} else {
			realOperand = float64(int64(operand) / int64(math.Pow(2, float64(fractionBits))))
		}
	default:
		panic("unsupported number of bits in FixedToFP()")
	}

	if realOperand == 0.0 {
		return fpu.FPZero(false, N)
	}
	return fpu.FPRound(realOperand, N, fpscr)
}

func (fpu *FPU) FPToFixed(operand uint64, N int, fractionBits int, unsigned bool, roundZero bool, fpscrControlled bool) uint64 {
	// page A2-58 to A2-59 of "ARMv7-M"

	// comments in quotation marks are taken from the psuedo code for FPToFixed()

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}
	if roundZero {
		fpscr.SetRMode(FPRoundZero)
	}

	typ, _, val := fpu.FPUnpack(operand, N, fpscr)

	// "For NaNs and infinities, FPUnpack() has produced a value that will round to the
	// required result of the conversion. Also, the value produced for infinities will
	// cause the conversion to overflow and signal an Invalid Operation floating-point
	// exception as required. NaNs must also generate such a floating-point exception"
	if typ == FPType_SNaN || typ == FPType_QNaN {
		fpu.FPProcessException(FPExc_InvalidOp, fpscr)
	}

	// "Scale value by specified number of fraction bits, then start rounding to an integer
	// and determine the rounding error"
	val = val * math.Pow(2, float64(fractionBits))
	intResult := int(val)
	roundingError := val - float64(intResult)

	// "Apply the specified rounding mode"
	var roundUp bool

	switch fpscr.RMode() {
	case FPRoundNearest:
		roundUp = (roundingError > 0.5) || (roundingError == 0.5 && intResult&0x01 == 0x01)
	case FPRoundPlusInf:
		roundUp = roundingError != 0.0
	case FPRoundNegInf:
		roundUp = false
	case FPRoundZero:
		roundUp = (roundingError != 0.0 && intResult < 0)
	}

	if roundUp {
		intResult++
	}

	var result uint64
	var overflow bool

	if unsigned {
		result, overflow = fpu.UnsignedSatQ(intResult, N)
	} else {
		result, overflow = fpu.SignedSatQ(intResult, N)
	}

	if overflow {
		fpu.FPProcessException(FPExc_InvalidOp, fpscr)
	} else if roundingError != 0.0 {
		fpu.FPProcessException(FPExc_Inexact, fpscr)
	}

	return result
}
