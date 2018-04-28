package rbits

import (
	"fmt"
	"log"
)

// implementing bit as a simple boolean
type bit bool

// Register is an array of of type bit, used for register representation
type Register []bit

// Size returns the number of bits in register
func (r Register) Size() int {
	return len(r)
}

func (r Register) String() string {
	return fmt.Sprintf("%s (%d) [0x%04x]", r.ToBits(), r.ToUint(), r.ToUint())
}

// IsNegative checks the sign bit of the register
func (r Register) IsNegative() bool {
	return bool(r[0])
}

// IsZero checks if register is all zero bits
func (r Register) IsZero() bool {
	for b := range r {
		if r[b] == true {
			return false
		}
	}
	return true
}

// Load value into register
func (r Register) Load(v interface{}) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	copy(r, b)
}

// Add value to register. Returns carry and overflow states
func (r Register) Add(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	sign := r[0]

	i := len(b) - 1

	// ripple adder
	for i >= 0 {
		if r[i] == false && b[i] == false && carry == false { // 0 0 0
			r[i] = false
			carry = false
		} else if r[i] == false && b[i] == false && carry == true { // 0 0 1
			r[i] = true
			carry = false
		} else if r[i] == false && b[i] == true && carry == false { // 0 1 0
			r[i] = true
			carry = false
		} else if r[i] == false && b[i] == true && carry == true { // 0 1 1
			r[i] = false
			carry = true
		} else if r[i] == true && b[i] == false && carry == false { // 1 0 0
			r[i] = true
			carry = false
		} else if r[i] == true && b[i] == false && carry == true { // 1 0 1
			r[i] = false
			carry = true
		} else if r[i] == true && b[i] == true && carry == false { // 1 1 0
			r[i] = false
			carry = true
		} else if r[i] == true && b[i] == true && carry == true { // 1 1 1
			r[i] = true
			carry = true
		}

		i--
	}

	overflow := sign == true && b[0] == true && r[0] == false

	return carry, overflow
}

// Subtract value from register. Returns carry and overflow states
//
// Note that carry flag is opposite of what you might expect when subtracting
// on the 6502/6507
func (r Register) Subtract(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	// generate two's complement
	i := 0
	for i < len(b) {
		b[i] = !b[i]
		i++
	}
	b.Add(1, false)

	return r.Add(b, !carry)
}

// EOR - XOR Register with value
func (r Register) EOR(v interface{}) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	i := 0
	for i < len(r) {
		r[i] = (r[i] || b[i]) && r[i] != b[i]
		i++
	}
}

// ORA - OR Register with value
func (r Register) ORA(v interface{}) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	i := 0
	for i < len(r) {
		r[i] = r[i] || b[i]
		i++
	}
}

// AND register with value
func (r Register) AND(v interface{}) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	i := 0
	for i < len(r) {
		r[i] = r[i] && b[i]
		i++
	}
}

// ROR rotates register 1 bit to the right. Returns new carry status.
func (r Register) ROR(carry bool) bool {
	rcarry := bool(r[len(r)-1])
	copy(r[1:], r[:len(r)-1])
	r[0] = bit(carry)
	return rcarry
}

// ROL rotates register 1 bit to the left. Returns new carry status.
func (r Register) ROL(carry bool) bool {
	rcarry := bool(r[0])
	copy(r[:len(r)-1], r[1:])
	r[len(r)-1] = bit(carry)
	return rcarry
}

// ASL (Arithmetic shift Left) shifts register one bit to the left. Returns
// the most significant bit as it was before the shift. If we think of the
// ASL operation as a multiply by two then the return value is the carry bit.
func (r Register) ASL() bool {
	rcarry := bool(r[0])
	copy(r[:len(r)-1], r[1:])
	r[len(r)-1] = bit(false)
	return rcarry
}

// LSR (Logical Shift Right) shifts register one bit to the rigth.
// the least significant bit as it was before the shift. If we think of
// the ASL operation as a division by two then the return value is the carry bit.
func (r Register) LSR() bool {
	rcarry := bool(r[len(r)-1])
	copy(r[1:], r[:len(r)-1])
	r[0] = bit(false)
	return rcarry
}
