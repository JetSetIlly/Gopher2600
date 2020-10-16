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

package registers

import (
	"fmt"
)

// Register is an array of of type bit, used for register representation.
type Register struct {
	label string
	value uint8
}

// NewRegister creates a new register of a givel size and name, and initialises
// the value.
func NewRegister(val uint8, label string) Register {
	return Register{
		value: val,
		label: label,
	}
}

// NewAnonRegister initialises a new register without a name.
func NewAnonRegister(val uint8) Register {
	return NewRegister(val, "")
}

// Label returns the registers label (or ID).
func (r Register) Label() string {
	return r.label
}

// returns value as a string in hexadecimal notation.
func (r Register) String() string {
	return fmt.Sprintf("%02x", r.value)
}

// Value returns the current value of the register.
func (r Register) Value() uint8 {
	return r.value
}

// BitWidth returns the number of bits used to store the register value.
func (r Register) BitWidth() int {
	return 8
}

// Address returns the current value of the register /as a uint16/. this is
// useful when you want to use the register value in an address context.
//
// for example, the stack pointer stores page zero addresses - which can be
// stored in just 8bits but which are always interpreted as 16bit value.
func (r Register) Address() uint16 {
	return uint16(r.value)
}

// IsNegative checks the sign bit of the register.
func (r Register) IsNegative() bool {
	return r.value&0x80 == 0x80
}

// IsZero checks if register is zero.
func (r Register) IsZero() bool {
	return r.value == 0
}

// IsBitV returns the state of the second MSB.
func (r Register) IsBitV() bool {
	return r.value&0x40 == 0x40
}

// Load value into register.
func (r *Register) Load(val uint8) {
	r.value = val
}

// Add value to register. Returns carry and overflow states.
func (r *Register) Add(val uint8, carry bool) (rcarry bool, overflow bool) {
	// note value of register before we change it
	v := r.value

	r.value += val
	if carry {
		r.value++
	}

	// overflow detection from Ken Shirriff's blog: "The 6502 overflow flag
	// explained mathematically"
	overflow = ((v ^ r.value) & (val ^ r.value) & 0x80) != 0

	// carry detection
	if v == r.value {
		rcarry = carry
	} else {
		rcarry = r.value < v
	}

	return rcarry, overflow
}

// Subtract value from register. Returns carry and overflow states.
func (r *Register) Subtract(val uint8, carry bool) (rcarry bool, overflow bool) {
	return r.Add(^val, carry)
}

// AND value with register.
func (r *Register) AND(val uint8) {
	r.value &= val
}

// ASL (arithmetic shift left) shifts register one bit to the left. Returns
// the most significant bit as it was before the shift. If we think of the
// ASL operation as a multiply by two then the return value is the carry bit.
func (r *Register) ASL() bool {
	carry := r.IsNegative()
	r.value <<= 1
	return carry
}

// EOR (exclusive or) value with register.
func (r *Register) EOR(val uint8) {
	r.value ^= val
}

// LSR (logical shift right) shifts register one bit to the right.
// the least significant bit as it was before the shift. If we think of
// the ASL operation as a division by two then the return value is the carry bit.
func (r *Register) LSR() bool {
	carry := r.value&1 == 1
	r.value >>= 1
	return carry
}

// ORA (non-exclusive or) value with register.
func (r *Register) ORA(val uint8) {
	r.value |= val
}

// ROL rotates register 1 bit to the left. Returns new carry status.
func (r *Register) ROL(carry bool) bool {
	rcarry := r.IsNegative()
	r.value <<= 1
	if carry {
		r.value |= 1
	}
	return rcarry
}

// ROR rotates register 1 bit to the right. Returns new carry status.
func (r *Register) ROR(carry bool) bool {
	rcarry := r.value&1 == 1
	r.value >>= 1
	if carry {
		r.value |= 0x80
	}
	return rcarry
}
