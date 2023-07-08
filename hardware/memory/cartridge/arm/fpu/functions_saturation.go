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

func (fpu *FPU) SignedSatQ(i int, N int, unsigned bool) (uint64, bool) {
	var result uint64
	var saturated bool

	powN := int(math.Pow(2, float64(N)-1))

	if i > powN-1 {
		result = uint64(powN) - 1
		saturated = true
	} else if i < -(powN - 1) {
		result = uint64(i)
		saturated = true
	} else {
		result = uint64(i)
	}

	return result & ((1 << N) - 1), saturated
}

func (fpu *FPU) UnsignedSatQ(i int, N int, unsigned bool) (uint64, bool) {
	var result uint64
	var saturated bool

	powN := int(math.Pow(2, float64(N)-1))

	if i > powN-1 {
		result = uint64(powN) - 1
		saturated = true
	} else {
		result = uint64(i)
	}

	return result & ((1 << N) - 1), saturated
}
