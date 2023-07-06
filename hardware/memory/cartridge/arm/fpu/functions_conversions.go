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

func (fpu *FPU) FixedToFP(operand uint64, fractionBits int, unsigned bool, nearest bool, N int, fpscrControlled bool) uint64 {
	// page A2-59 of "ARMv7-M"

	// bits(N) FixedToFP(bits(M) operand, integer N, integer fraction_bits, boolean unsigned,
	//		boolean round_to_nearest, boolean fpscr_controlled)
	//
	//		assert N IN {32,64};
	//		fpscr_val = if fpscr_controlled then FPSCR else StandardFPSCRValue();
	//		if round_to_nearest then fpscr_val<23:22> = ‘00’;
	//		int_operand = if unsigned then UInt(operand) else SInt(operand);
	//		real_operand = int_operand / 2^fraction_bits;
	//		if real_operand == 0.0 then
	//			result = FPZero(‘0’, N);
	//		else
	//			result = FPRound(real_operand, N, fpscr_val);
	//		return result;

	if N != 32 && N != 64 {
		panic("unsupported number of bits in FixedToFP()")
	}

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

	if unsigned {
		intOperand := operand
		realOperand = float64(intOperand) / float64(2^uint64(fractionBits))
	} else {
		intOperand := int64(operand)
		realOperand = float64(intOperand) / float64(2^int64(fractionBits))
	}

	if realOperand == 0.0 {
		return fpu.FPZero(false, N)
	}
	return fpu.FPRound(realOperand, N, fpscr)
}
