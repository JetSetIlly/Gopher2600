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

package leb128

// ULEB128 decoding algorithm taken from page 218 of "DWARF4 Standard", figure 46
//
// returns decoded value and the number of bytes consumed from the encoded array
func DecodeULEB128(encoded []uint8) (uint64, int) {
	var result uint64
	var shift uint64

	var n int
	for _, v := range encoded {
		n++
		result |= uint64(v&0x7f) << shift
		if v&0x80 == 0x00 {
			break
		}
		shift += 7
	}

	return result, n
}

// LEB128 decoding algorithm taken from page 218 of "DWARF4 Standard", figure 47
//
// returns decoded value and the number of bytes consumed from the encoded array
func DecodeSLEB128(encoded []uint8) (int64, int) {
	const size = 64

	var result int64
	var shift uint64

	var v uint8
	var n int
	for _, v = range encoded {
		n++
		result |= int64((int64(v) & 0x7f) << shift)
		shift += 7
		if v&0x80 == 0x00 {
			break
		}
	}

	// sign extend last byte from the encoded slice
	if shift < size && v&0x40 > 0 {
		result |= -(1 << shift)
	}

	return int64(result), n
}
