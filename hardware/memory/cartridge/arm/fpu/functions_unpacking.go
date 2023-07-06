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
	"math/bits"
)

type FPType int

const (
	FPType_Nonzero FPType = iota
	FPType_Zero
	FPType_Infinity
	FPType_QNaN
	FPType_SNaN
)

func (fpu *FPU) FPUnpack(fpval uint64, N int, fpscr FPSCR) (FPType, bool, float64) {
	// page A2-47 to A2-49 of "ARMv7-M"

	// comments in quotation marks are taken from the psuedo code for FPUnpack()

	// this block is not in the pseudocode of FPUnpack() but it helps us with
	// our calls to bits.OnesCount() when figuring out if a number represents
	// infinity or a NaN
	var E int
	switch N {
	case 16:
		E = 5
	case 32:
		E = 8
	case 64:
		E = 11
	default:
		panic("unsupported number of bits in FPUnpack()")
	}

	// similarly, F helps us when testing the bit that differentiates between
	// QNaN and SNaN
	F := N - E - 1

	// note: for 32bit and 64bit numbers, when converting to a FPType_Nonzero
	// value, the math package is used

	// "Unpack a floating-point that it represents. The and infinities, is very NaNs.
	// (These values are and conversions.) number into its type, sign bit and real
	// number result has the correct large in magnitude for infinities, chosen to
	// simplify the description the real number sign for numbers and is 0.0 for of
	// comparisons The ‘fpscr_val’ argument supplies FPSCR control bits. Status
	// information is updated directly in the FPSCR where appropriate"

	var sign bool
	var typ FPType
	var value float64

	switch N {
	case 16:
		sign = fpval&0x8000 == 0x8000
		exp16 := (fpval & 0x7c00) >> 10
		frac16 := fpval & 0x3ff
		if bits.OnesCount64(exp16) == 0 {
			// "Produce zero if value is zero"
			if bits.OnesCount64(frac16) == 0 {
				typ = FPType_Zero
				value = 0.0
			} else {
				// "value = 2.0^-14 * (UInt(frac16) * 2.0^-10);"
				value = math.Pow(2, -14) * float64(frac16) * math.Pow(2, -10)
				typ = FPType_Nonzero
			}
		} else if bits.OnesCount64(exp16) == E && !fpscr.AHP() {
			if bits.OnesCount64(frac16) == 0 {
				typ = FPType_Infinity
				value = math.Pow(2, 1000000)
			} else {
				if frac16>>F == 0x01 {
					typ = FPType_QNaN
				} else {
					typ = FPType_SNaN
				}
				value = 0.0
			}
		} else {
			// "value = 2.0^(UInt(exp16)-15) * (1.0 + UInt(frac16) * 2.0^-10);"
			value = math.Pow(2, float64(exp16)-15) * (1.0 + float64(frac16)*math.Pow(2.0, -10))
			typ = FPType_Nonzero
		}
	case 32:
		sign = fpval&0x80000000 == 0x80000000
		exp32 := (fpval & 0x7f800000) >> 23
		frac32 := fpval & 0x007fffff
		if bits.OnesCount64(exp32) == 0 {
			// "Produce zero if value is zero of flush-to-zero is selected"
			if bits.OnesCount64(frac32) == 0 || fpscr.FZ() {
				typ = FPType_Zero
				value = 0.0
				if bits.OnesCount64(frac32) != 0 {
					// "denormalised input flushed to zero"
					fpu.FPProcessException(FPExc_InputDenorm, fpscr)
				}
			} else {
				// "value = 2.0^-126 * (UInt(frac32) * 2.0^-23);"
				value = math.Pow(2.0, -126) * (float64(frac32) * math.Pow(2.0, -23))
				typ = FPType_Nonzero
			}
		} else if bits.OnesCount64(exp32) == E {
			if bits.OnesCount64(frac32) == 0 {
				typ = FPType_Infinity
				value = math.Pow(2, 1000000)
			} else {
				if frac32>>F == 0x01 {
					typ = FPType_QNaN
				} else {
					typ = FPType_SNaN
				}
				value = 0.0
			}
		} else {
			// "value = 2.0^(UInt(exp32)-127) * (1.0 + UInt(frac32) * 2.0^-23);"
			value = math.Pow(2.0, float64(exp32)-127) * (1.0 + float64(frac32)*math.Pow(2.0, -23))
			typ = FPType_Nonzero
		}
	case 64:
		sign = fpval&0x8000000000000000 == 0x8000000000000000
		exp64 := (fpval & 0x7ff0000000000000) >> 52
		frac64 := fpval & 0x000fffffffffffff
		if bits.OnesCount64(exp64) == 0 {
			// "Produce zero if value is zero of flush-to-zero is selected"
			if bits.OnesCount64(frac64) == 0 || fpscr.FZ() {
				typ = FPType_Zero
				value = 0.0
				if bits.OnesCount64(frac64) != 0 {
					// "denormalised input flushed to zero"
					fpu.FPProcessException(FPExc_InputDenorm, fpscr)
				}
			} else {
				// "value = 2.0^-1022 * (UInt(frac64) * 2.0^-52);"
				value = math.Pow(2.0, -1022) * (float64(frac64) * math.Pow(2.0, -52))
				typ = FPType_Nonzero
			}
		} else if bits.OnesCount64(exp64) == E {
			if bits.OnesCount64(frac64) == 0 {
				typ = FPType_Infinity
				value = math.Pow(2, 1000000)
			} else {
				if frac64>>F == 0x01 {
					typ = FPType_QNaN
				} else {
					typ = FPType_SNaN
				}
				value = 0.0
			}
		} else {
			// "value = 2.0^(UInt(exp64)-1023) * (1.0 + UInt(frac64) * 2.0^-52);"
			value = math.Pow(2.0, float64(exp64)-1023) * (1.0 + float64(frac64)*math.Pow(2.0, -52))
			typ = FPType_Nonzero
		}
	default:
		panic("unsupported number of bits in FPUnpack()")
	}

	if sign {
		value *= -1
	}

	return typ, sign, value
}
