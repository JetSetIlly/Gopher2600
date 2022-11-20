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

package leb128_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/leb128"
	"github.com/jetsetilly/gopher2600/test"
)

func TestDecodeULEB128(t *testing.T) {
	// tests from page 162 of the "DWARF4 Standard"
	v := []uint8{0x7f, 0x00}
	r, n := leb128.DecodeULEB128(v)
	test.Equate(t, n, 1)
	test.Equate(t, r, uint64(127))

	v = []uint8{0x80, 0x01, 0x00}
	r, n = leb128.DecodeULEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, uint64(128))

	v = []uint8{0x81, 0x01, 0x00}
	r, n = leb128.DecodeULEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, uint64(129))

	v = []uint8{0x82, 0x01, 0x00}
	r, n = leb128.DecodeULEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, uint64(130))

	v = []uint8{0xb9, 0x64, 0x00}
	r, n = leb128.DecodeULEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, uint64(12857))
}

func TestDecodeSLEB128(t *testing.T) {
	// tests from page 163 of the "DWARF4 Standard"
	v := []uint8{0x02, 0x00}
	r, n := leb128.DecodeSLEB128(v)
	test.Equate(t, n, 1)
	test.Equate(t, r, int64(2))

	v = []uint8{0x7e, 0x00}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 1)
	test.Equate(t, r, int64(-2))

	v = []uint8{0xff, 0x00}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(127))

	v = []uint8{0x81, 0x7f}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(-127))

	v = []uint8{0x80, 0x01}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(128))

	v = []uint8{0x80, 0x7f}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(-128))

	v = []uint8{0x81, 0x01}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(129))

	v = []uint8{0xff, 0x7e}
	r, n = leb128.DecodeSLEB128(v)
	test.Equate(t, n, 2)
	test.Equate(t, r, int64(-129))
}
