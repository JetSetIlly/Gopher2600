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

func (fpu *FPU) FPNeg(value uint64, N int) uint64 {
	switch N {
	case 32:
		return value ^ 0xffffffff80000000
	case 64:
		return value ^ 0x8000000000000000
	}
	panic("unsupported number of bits for FPNeg() function")
}

func (fpu *FPU) FPAbs(value uint64, N int) uint64 {
	switch N {
	case 32:
		return value & 0x7fffffff
	case 64:
		return value & 0x7fffffffffffffff
	}
	panic("unsupported number of bits for FPAbs() function")
}
