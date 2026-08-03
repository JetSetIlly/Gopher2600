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

// page A-29 of "ARMv7-M"
//
// "Some instructions perform saturating arithmetic, that is, if the result of
// the arithmetic overflows the destination signed or unsigned N-bit integer
// range, the result produced is the largest or smallest value in that range,
// rather than wrapping around modulo 2 N . This is supported in pseudocode by
// the SignedSatQ() and UnsignedSatQ() functions when a boolean result is wanted
// saying whether saturation occurred, and by the SignedSat() and UnsignedSat()
// functions when only the saturated result is wanted"

func UnsignedSatQ(i int, N int) (uint64, bool) {
	mask := (uint64(1) << N) - 1

	switch {
	case i < 0:
		return 0, true
	case uint64(i) > mask:
		return mask, true
	default:
		return uint64(i) & mask, false
	}
}

func SignedSatQ(i int, N int) (uint64, bool) {
	mask := (uint64(1) << N) - 1
	min := -(1 << (N - 1))
	max := (1 << (N - 1)) - 1

	switch {
	case i < min:
		return uint64(min) & mask, true
	case i > max:
		return uint64(max) & mask, true
	default:
		return uint64(i) & mask, false
	}
}
