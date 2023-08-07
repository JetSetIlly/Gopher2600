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

type FPException int

const (
	FPExc_InvalidOp FPException = iota
	FPExc_DivideByZero
	FPExc_Overflow
	FPExc_Underflow
	FPExc_Inexact
	FPExc_InputDenorm
)

func (fpu *FPU) FPProcessException(exception FPException, fpscr FPSCR) {
	// page A2-49 of "ARMv7-M"

	// we're taking advantage of the fact that the enable bits and the
	// "cumulative" bits are 8 bits apart in the FPU status register

	if fpscr.value>>uint32(exception+8) == 0x01 {
		panic("IMPLEMENTATION DEFINED floating-point trap handling")
	} else {
		fpu.Status.value |= (0x01 << exception)
	}
}

func (fpu *FPU) FPProcessNaN(typ FPType, N int, op uint64, fpscr FPSCR) uint64 {
	// page A2-49 of "ARMv7-M"

	var topfrac int

	switch N {
	case 32:
		topfrac = 22
	case 64:
		topfrac = 51
	default:
		panic("unsupported number of bits in FPProcessNaN()")
	}

	result := op

	if typ == FPType_SNaN {
		result = result | (0x01 << topfrac)
		fpu.FPProcessException(FPExc_InvalidOp, fpscr)
	}

	if fpscr.DN() {
		result = fpu.FPDefaultNaN(N)
	}

	return result
}

func (fpu *FPU) FPProcessNaNs(typ1 FPType, typ2 FPType, N int, op1 uint64, op2 uint64, fpscr FPSCR) (bool, uint64) {
	// page A2-49 to A2-50 of "ARMv7-M"

	var done bool
	var result uint64

	if typ1 == FPType_SNaN {
		done = true
		result = fpu.FPProcessNaN(typ1, N, op1, fpscr)
	} else if typ2 == FPType_SNaN {
		done = true
		result = fpu.FPProcessNaN(typ2, N, op2, fpscr)
	} else if typ1 == FPType_QNaN {
		done = true
		result = fpu.FPProcessNaN(typ1, N, op1, fpscr)
	} else if typ2 == FPType_QNaN {
		done = true
		result = fpu.FPProcessNaN(typ2, N, op2, fpscr)
	}

	return done, result
}

func (fpu *FPU) FPProcessNaNs3(typ1 FPType, typ2 FPType, typ3 FPType, N int,
	op1 uint64, op2 uint64, op3 uint64, fpscr FPSCR) (bool, uint64) {
	// page A2-50 of "ARMv7-M"

	var done bool
	var result uint64

	if typ1 == FPType_SNaN {
		done = true
		result = fpu.FPProcessNaN(typ1, N, op1, fpscr)
	} else if typ2 == FPType_SNaN {
		done = true
		result = fpu.FPProcessNaN(typ2, N, op2, fpscr)
	} else if typ3 == FPType_SNaN {
		done = true
		result = fpu.FPProcessNaN(typ3, N, op3, fpscr)
	} else if typ1 == FPType_QNaN {
		done = true
		result = fpu.FPProcessNaN(typ1, N, op1, fpscr)
	} else if typ2 == FPType_QNaN {
		done = true
		result = fpu.FPProcessNaN(typ2, N, op2, fpscr)
	} else if typ3 == FPType_QNaN {
		done = true
		result = fpu.FPProcessNaN(typ3, N, op3, fpscr)
	}

	return done, result
}
