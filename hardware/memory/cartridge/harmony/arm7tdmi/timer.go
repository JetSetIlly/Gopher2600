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

package arm7tdmi

type timer struct {
	active bool

	control uint32
	counter uint32
}

func (t *timer) step() {
	if !t.active {
		return
	}

	// this figure isn't accurate
	t.counter += 0x3a
}

func (t *timer) write(addr uint32, val uint32) bool {
	switch addr {
	case 0xe0008004:
		t.control = val
		t.active = t.control&0x01 == 0x01
	case 0xe0008008:
		t.counter = val
	case 0xe0008010:
	case 0xe0008014:
	case 0xe0008018:
	case 0xe000801c:
	default:
		return false
	}

	return true
}

func (t *timer) read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case 0xe0008004:
		val = t.control
	case 0xe0008008:
		val = t.counter
	case 0xe0008010:
	case 0xe0008014:
	case 0xe0008018:
	case 0xe000801c:
	default:
		return 0, false
	}

	return val, true
}
