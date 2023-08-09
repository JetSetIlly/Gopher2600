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

// the status register is an incomplete implementation or CPSR/APSR register
//
// the structure also contains the IT state fields (itCond and itMask) which are
// technically part of the EPSR register in 32bit architectures
//
// this makeshift type will suffice for now but should be replaced with a more
// flexible and more accurate system in the future
type status struct {
	// basic CPSR bits. present in the APSR too
	negative   bool
	zero       bool
	carry      bool
	overflow   bool
	saturation bool

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

func (sr status) String() string {
	s := strings.Builder{}
	s.WriteString("Status: ")

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
	if sr.carry {
		s.WriteRune('C')
	} else {
		s.WriteRune('c')
	}
	if sr.overflow {
		s.WriteRune('V')
	} else {
		s.WriteRune('v')
	}
	if sr.saturation {
		s.WriteRune('Q')
	} else {
		s.WriteRune('q')
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
func (sr *status) condition(cond uint8) (bool, string) {
	var mnemonic string
	var b bool

	switch cond {
	case 0b0000:
		// equal
		mnemonic = "BEQ"
		b = sr.zero
	case 0b0001:
		// not equal
		mnemonic = "BNE"
		b = !sr.zero
	case 0b0010:
		// carry set
		mnemonic = "BCS"
		b = sr.carry
	case 0b0011:
		// carry clear
		mnemonic = "BCC"
		b = !sr.carry
	case 0b0100:
		// minus
		mnemonic = "BMI"
		b = sr.negative
	case 0b0101:
		// plus
		mnemonic = "BPL"
		b = !sr.negative
	case 0b0110:
		// overflow
		mnemonic = "BVS"
		b = sr.overflow
	case 0b0111:
		// no overflow
		mnemonic = "BVC"
		b = !sr.overflow
	case 0b1000:
		// unsigned higer C==1 and Z==0
		mnemonic = "BHI"
		b = sr.carry && !sr.zero
	case 0b1001:
		// unsigned lower C==0 and Z==1
		mnemonic = "BLS"
		b = !sr.carry || sr.zero
	case 0b1010:
		// signed greater than N==V
		mnemonic = "BGE"
		b = sr.negative == sr.overflow
	case 0b1011:
		// signed less than N!=V
		mnemonic = "BLT"
		b = sr.negative != sr.overflow
	case 0b1100:
		// signed greater than Z==0 and N==V
		mnemonic = "BGT"
		b = !sr.zero && sr.negative == sr.overflow
	case 0b1101:
		// signed less than or qual Z==1 or N!=V
		mnemonic = "BLE"
		b = sr.zero || sr.negative != sr.overflow
	case 0b1110:
		mnemonic = "B"
		b = true
	case 0b1111:
		panic("unpredictable condition")
	}

	return b, mnemonic
}
