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
	"fmt"
	"strings"
)

// the arm has a 32 bit Status register but we only need the CSPR bits
// currently.
type Status struct {
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

func (sr Status) String() string {
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

	s.WriteString(fmt.Sprintf("   itMask: %04b", sr.itMask))

	return s.String()
}

func (sr *Status) reset() {
	sr.negative = false
	sr.zero = false
	sr.overflow = false
	sr.carry = false
}

func (sr *Status) isNegative(a uint32) {
	sr.negative = a&0x80000000 == 0x80000000
}

func (sr *Status) isZero(a uint32) {
	sr.zero = a == 0x00
}

func (sr *Status) isOverflow(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d >>= 31
	e := (d & 0x01) + ((a >> 31) & 0x01) + ((b >> 31) & 0x01)
	e >>= 1
	sr.overflow = (d^e)&0x01 == 0x01
}

func (sr *Status) isCarry(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d = (d >> 31) + (a >> 31) + (b >> 31)
	sr.carry = d&0x02 == 0x02
}

func (sr *Status) setCarry(a bool) {
	sr.carry = a
}

func (sr *Status) setOverflow(a bool) {
	sr.overflow = a
}

// conditional execution information from "A7.3 Conditional execution" in "ARMv7-M"
func (sr *Status) condition(cond uint8) bool {
	b := false

	switch cond {
	case 0b0000:
		// equal
		// BEQ
		b = sr.zero
	case 0b0001:
		// not equal
		// BNE
		b = !sr.zero
	case 0b0010:
		// carry set
		// BCS
		b = sr.carry
	case 0b0011:
		// carry clear
		// BCC
		b = !sr.carry
	case 0b0100:
		// minus
		// BMI
		b = sr.negative
	case 0b0101:
		// plus
		// BPL
		b = !sr.negative
	case 0b0110:
		// overflow
		// BVS
		b = sr.overflow
	case 0b0111:
		// no overflow
		// BVC
		b = !sr.overflow
	case 0b1000:
		// unsigned higer C==1 and Z==0
		// BHI
		b = sr.carry && !sr.zero
	case 0b1001:
		// unsigned lower C==0 and Z==1
		// BLS
		b = !sr.carry || sr.zero
	case 0b1010:
		// signed greater than N==V
		// BGE
		b = sr.negative == sr.overflow
	case 0b1011:
		// signed less than N!=V
		// BLT
		b = sr.negative != sr.overflow
	case 0b1100:
		// signed greater than Z==0 and N==V
		// BGT
		b = !sr.zero && sr.negative == sr.overflow
	case 0b1101:
		// signed less than or qual Z==1 or N!=V
		// BLE
		b = sr.zero || sr.negative != sr.overflow
	case 0b1110:
		b = true
	case 0b1111:
		panic("unpredictable condition")
	}

	return b
}
