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

import (
	"strings"
)

// the arm has a 32 bit status register but we only need the CSPR bits
// currently.
type status struct {
	// CPSR (current program status register) bits
	negative bool
	zero     bool
	overflow bool
	carry    bool

	// mask and firstcond bits of most recent IT instruction. rather than
	// maintaining a single itState value, the condition and mask are split
	// into two. this is for clarity and performance (checking itMask to see if
	// we're in an IT block is a simple comparison to zero)
	//
	// instruction is in an IT block when itMask != 0b0000
	//
	// updating of itCond is done in the main Run() function
	itCond uint8
	itMask uint8
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

func (sr *status) isOverflow(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d >>= 31
	e := (d & 0x01) + ((a >> 31) & 0x01) + ((b >> 31) & 0x01)
	e >>= 1
	sr.overflow = (d^e)&0x01 == 0x01
}

func (sr *status) isCarry(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d = (d >> 31) + (a >> 31) + (b >> 31)
	sr.carry = d&0x02 == 0x02
}

func (sr *status) setCarry(a bool) {
	sr.carry = a
}

func (sr *status) setOverflow(a bool) {
	sr.overflow = a
}

// conditional execution information from "A7.3 Conditional execution" in "ARMv7-M"
func (sr *status) condition(cond uint8) bool {
	b := false

	switch cond {
	case 0b0000:
		// equal
		b = sr.zero
	case 0b0001:
		// not equal
		b = !sr.zero
	case 0b0010:
		// carry set
		b = sr.carry
	case 0b0011:
		// carry clear
		b = !sr.carry
	case 0b0100:
		// minus
		b = sr.negative
	case 0b0101:
		// plus
		b = !sr.negative
	case 0b0110:
		// overflow
		b = sr.overflow
	case 0b0111:
		// no overflow
		b = !sr.overflow
	case 0b1000:
		// unsigned higer C==1 and Z==0
		b = sr.carry && !sr.zero
	case 0b1001:
		// unsigned lower C==0 and Z==1
		b = !sr.carry || sr.zero
	case 0b1010:
		// signed greater than N==V
		b = sr.negative == sr.overflow
	case 0b1011:
		// signed less than N!=V
		b = sr.negative != sr.overflow
	case 0b1100:
		// signed greater than Z==0 and N==V
		b = !sr.zero && sr.negative == sr.overflow
	case 0b1101:
		// signed less than or qual Z==1 or N!=V
		b = sr.zero || sr.negative != sr.overflow
	case 0b1110:
		b = true
	case 0b1111:
		panic("unpredictable condition")
	}

	return b
}
