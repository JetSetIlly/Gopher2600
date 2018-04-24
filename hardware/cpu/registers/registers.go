package registers

import (
	"fmt"
	"log"
)

// implementing bit as a simple boolean
type bit bool

// Bits is an array of of type bit, used for register representation
type Bits []bit

func (r Bits) String() string {
	return fmt.Sprintf("%s (%d) [0x%04x]", r.ToString(), r.ToUint(), r.ToUint())
}

// Load value into register
func (r Bits) Load(v interface{}) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	copy(r, b)
}

// Add value to register. Returns carry and overflow states
func (r Bits) Add(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, len(r))
	if err != nil {
		log.Fatalln(err)
	}

	sign := r[0]

	i := len(b) - 1

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
func (r Bits) Subtract(v interface{}, carry bool) (bool, bool) {
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
func (r Bits) EOR(v interface{}) {
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
func (r Bits) ORA(v interface{}) {
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
func (r Bits) AND(v interface{}) {
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
func (r Bits) ROR(carry bool) bool {
	rcarry := bool(r[len(r)-1])
	copy(r[1:], r[:len(r)-1])
	r[0] = bit(carry)
	return rcarry
}

// ROL rotates register 1 bit to the left. Returns new carry status.
func (r Bits) ROL(carry bool) bool {
	rcarry := bool(r[0])
	copy(r[:len(r)-1], r[1:])
	r[len(r)-1] = bit(carry)
	return rcarry
}

// ASL (Arithmetic shift Left) shifts register one bit to the left. Returns
// the most significant bit as it was before the shift. If we think of the
// ASL operation as a multiply by two then the return value is the carry bit.
func (r Bits) ASL() bool {
	rcarry := bool(r[0])
	copy(r[:len(r)-1], r[1:])
	r[len(r)-1] = bit(false)
	return rcarry
}

// LSR (Logical Shift Right) shifts register one bit to the rigth.
// the least significant bit as it was before the shift. If we think of
// the ASL operation as a division by two then the return value is the carry bit.
func (r Bits) LSR() bool {
	rcarry := bool(r[len(r)-1])
	copy(r[1:], r[:len(r)-1])
	r[0] = bit(false)
	return rcarry
}

// IsNegative checks the sign bit of the register
func (r Bits) IsNegative() bool {
	return bool(r[0])
}

// IsZero checks if register is all zero bits
func (r Bits) IsZero() bool {
	for b := range r {
		if r[b] == true {
			return false
		}
	}
	return true
}
