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

import "strings"

// the arm7tdmi has a 32 bit status register but we only need the CSPR bits
// currently.
type status struct {
	// CPSR (current program status register) bits
	negative bool
	zero     bool
	overflow bool
	carry    bool
}

func (sr *status) String() string {
	s := strings.Builder{}
	if sr.negative {
		s.WriteRune('N')
	} else {
		s.WriteRune('n')
	}
	if sr.zero {
		s.WriteRune('Z')
	} else {
		s.WriteRune('z')
	}
	if sr.overflow {
		s.WriteRune('V')
	} else {
		s.WriteRune('v')
	}
	if sr.carry {
		s.WriteRune('C')
	} else {
		s.WriteRune('c')
	}
	return s.String()
}

func (sr *status) reset() {
	sr.negative = false
	sr.zero = false
	sr.overflow = false
	sr.carry = false
}

func (sr *status) isNegative(a uint32) {
	sr.negative = a&0x80000000 == 0x80000000
}

func (sr *status) isZero(a uint32) {
	sr.zero = a == 0x00
}

func (sr *status) setOverflow(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d >>= 31
	e := (d & 0x01) + ((a >> 31) & 0x01) + ((b >> 31) & 0x01)
	e >>= 1
	sr.overflow = (d^e)&0x01 == 0x01
}

func (sr *status) setCarry(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d = (d >> 31) + (a >> 31) + (b >> 31)
	sr.carry = d&0x02 == 0x02
}
