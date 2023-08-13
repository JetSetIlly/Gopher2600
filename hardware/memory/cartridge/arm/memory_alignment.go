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

package arm

// align address to 32bits. some instructions will align the PC to the 64bit boundary
func AlignTo64bits(v uint32) uint32 {
	return v & 0xfffffff8
}

// checks whether address is aligned to 64bits
func IsAlignedTo64bits(v uint32) bool {
	return v&0xfffffff8 == v
}

// align address to 32bits. some instructions will align the PC to the 32bit boundary
func AlignTo32bits(v uint32) uint32 {
	return v & 0xfffffffc
}

// checks whether address is aligned to 32bits
func IsAlignedTo32bits(v uint32) bool {
	return v&0xfffffffc == v
}

// align address to 16bits
func AlignTo16bits(v uint32) uint32 {
	return v & 0xfffffffe
}

// checks whether address is aligned to 16bits
func IsAlignedTo16bits(v uint32) bool {
	return v&0xfffffffe == v
}
