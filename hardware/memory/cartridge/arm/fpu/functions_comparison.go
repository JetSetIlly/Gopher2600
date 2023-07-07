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

func (fpu *FPU) FPCompare(op1 uint64, op2 uint64, N int, quietNaNexc bool, fpscrControlled bool) {
	// page A2-52 of "ARMv7-M"

	// unlike the reference function this implementation sets the FPU status
	// registers directly
	// (FPSCR.N, FPSCR.Z, FPSCR.C, FPSCR.V)

	var fpscr FPSCR
	if fpscrControlled {
		fpscr = fpu.Status
	} else {
		fpscr = fpu.StandardFPSCRValue()
	}

	typ1, _, value1 := fpu.FPUnpack(op1, N, fpscr)
	typ2, _, value2 := fpu.FPUnpack(op2, N, fpscr)

	if typ1 == FPType_SNaN || typ1 == FPType_QNaN || typ2 == FPType_SNaN || typ2 == FPType_QNaN {
		fpu.Status.SetNZCV(0b0011)
		if typ1 == FPType_SNaN || typ2 == FPType_SNaN || quietNaNexc {
			fpu.FPProcessException(FPExc_InvalidOp, fpscr)
		}
		return
	}

	if value1 == value2 {
		fpu.Status.SetNZCV(0b0110)
	} else if value1 < value2 {
		fpu.Status.SetNZCV(0b1000)
	} else { // value1 > value2
		fpu.Status.SetNZCV(0b0010)
	}
}
