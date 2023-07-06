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

import (
	"math"
)

func powInt(a, b int) int {
	return int(math.Pow(float64(a), float64(b)))
}

func (fpu *FPU) FPRound(value float64, N int, fpscr FPSCR) uint64 {
	// on pages A2-50 to A2-52 of "ARMv7-M"

	// comments in quotation marks are taken from the psuedo code for FPRound()

	if value == 0.0 {
		panic("FPRound() should never have been called with a value of 0.0")
	}

	// "The FPRound() function rounds and encodes a single-precision floating-point result value to a specified destination
	// format. This includes processing Overflow, Underflow and Inexact floating-point exceptions and performing
	// flush-to-zero processing on result values."

	// "Obtain format parameters - minimum exponent, number of exponent and fraction bits"
	var E int

	switch N {
	case 16:
		E = 5
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in FPRound()")
	}

	minExp := 2 - powInt(2, E-1)
	F := N - E - 1

	// "Split value into sign, unrounded mantissa and exponent"
	sign := value < 0.0
	mantissa := value
	if sign {
		mantissa = -mantissa
	}

	exponent := 0
	for mantissa < 1.0 {
		mantissa *= 2.0
		exponent--
	}
	for mantissa >= 2.0 {
		mantissa /= 2.0
		exponent++
	}

	// "Deal with flush-to-zero"
	if fpscr.FZ() && N != 16 && exponent < minExp {
		fpu.Status.SetUFC(true)
		return fpu.FPZero(sign, N)
	}

	// "Start creating the exponent value for the result. Start by biasing the actual exponent
	// so that the minimum exponent becomes 1, lower values 0 (indicating possible underflow)"

	biasedExp := exponent - minExp + 1
	if biasedExp < 0 {
		biasedExp = 0
	}
	if biasedExp == 0 {
		mantissa /= math.Pow(2, float64(minExp-exponent))
	}

	// 2 raised to the F power is a common calculation
	p2F := powInt(2, F)

	// "Get the unrounded mantissa as an integer, and the “units in last place” rounding error"
	intMant := int(mantissa * float64(p2F))
	roundingError := (mantissa * float64(p2F)) - float64(intMant)

	intMant &= ((0x1 << F) - 1)

	// "Underflow occurs if exponent is too small before rounding, and result is inexact or
	// the Underflow exception is trapped"
	//
	// bit 11 of FPSCR is reserved but even so, the pseudo-code says that that's
	// the bit to be tested
	if biasedExp == 0 && (roundingError != 0.0 || (fpscr.value>>11)&0x01 == 0x01) {
		fpu.FPProcessException(FPExc_Underflow, fpscr)
	}

	// "Round result according to rounding mode"
	var roundUp bool
	var overflowToInf bool
	switch fpscr.RMode() {
	case FPRoundNearest:
		roundUp = roundingError > 0.5 || (roundingError == 0.5 && intMant&0x01 == 0x01)
		overflowToInf = true
	case FPRoundPlusInf:
		roundUp = roundingError != 0.0 && !sign
		overflowToInf = !sign
	case FPRoundNegInf:
		roundUp = roundingError != 0.0 && sign
		overflowToInf = sign
	case FPRoundZero:
		roundUp = false
		overflowToInf = false
	}

	if roundUp {
		intMant++
		if intMant == p2F {
			// "rounded up from denormalised to normalised"
			biasedExp = 1
		}
		if intMant == powInt(2, (F+1)) {
			// rounded up to next exponent
			biasedExp++
			intMant /= 2
		}
	}

	// "Deal with overflow and generate result"

	var result uint64

	if N != 16 || !fpscr.AHP() {
		// "Single, double or IEEE half precision"
		if biasedExp >= powInt(2, E)-1 {
			if overflowToInf {
				result = fpu.FPInfinity(sign, N)
			} else {
				result = fpu.FPMaxNormal(sign, N)
			}
			fpu.FPProcessException(FPExc_Overflow, fpscr)

			// "ensure that an inexact exception occurs"
			roundingError = 1.0
		} else {
			if sign {
				result = uint64((1<<(N-1))-1) ^ 0xffffffffffffffff
			}
			result = result | uint64(biasedExp<<F) | uint64(intMant)
		}
	} else {
		// "Alternative half precision (with N==16)"
		if biasedExp >= powInt(2, E) {
			result = 0x7fff
			if sign {
				result |= 0xffffffffffff8000
			}
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)

			// "ensure that an inexact exception for no occur"
			roundingError = 0.0
		} else {
			if sign {
				result = uint64((1<<(N-1))-1) ^ 0xffffffffffffffff
			}
			result = result | uint64(biasedExp<<F) | uint64(intMant)
		}
	}

	// "Deal with inexact exception"
	if roundingError != 0.0 {
		fpu.FPProcessException(FPExc_Inexact, fpscr)
	}

	return result
}
