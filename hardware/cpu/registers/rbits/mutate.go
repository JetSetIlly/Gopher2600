package rbits

import "log"

// this module of the rbits packagae contains all the methods that mutate the register

// Load value into register
func (r *Register) Load(v interface{}) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("Load: %s", err.Error())
	}

	copy(r.value, b.value)
}

// Add value to register. Returns carry and overflow states
func (r *Register) Add(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("Add: %s", err.Error())
	}

	sign := r.value[0]

	i := len(b.value) - 1

	// ripple adder
	for i >= 0 {
		if r.value[i] == false && b.value[i] == false && carry == false { // 0 0 0
			r.value[i] = false
			carry = false
		} else if r.value[i] == false && b.value[i] == false && carry == true { // 0 0 1
			r.value[i] = true
			carry = false
		} else if r.value[i] == false && b.value[i] == true && carry == false { // 0 1 0
			r.value[i] = true
			carry = false
		} else if r.value[i] == false && b.value[i] == true && carry == true { // 0 1 1
			r.value[i] = false
			carry = true
		} else if r.value[i] == true && b.value[i] == false && carry == false { // 1 0 0
			r.value[i] = true
			carry = false
		} else if r.value[i] == true && b.value[i] == false && carry == true { // 1 0 1
			r.value[i] = false
			carry = true
		} else if r.value[i] == true && b.value[i] == true && carry == false { // 1 1 0
			r.value[i] = false
			carry = true
		} else if r.value[i] == true && b.value[i] == true && carry == true { // 1 1 1
			r.value[i] = true
			carry = true
		}

		i--
	}

	overflow := sign == true && b.value[0] == true && r.value[0] == false

	return carry, overflow
}

// Subtract value from register. Returns carry and overflow states
//
// Note that carry flag is opposite of what you might expect when subtracting
// on the 6502/6507
func (r *Register) Subtract(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("Subtract: %s", err.Error())
	}

	// generate two's complement
	i := 0
	for i < len(b.value) {
		b.value[i] = !b.value[i]
		i++
	}
	b.Add(1, false)

	return r.Add(b, !carry)
}

// EOR - XOR Register with value
func (r *Register) EOR(v interface{}) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("EOR: %s", err.Error())
	}

	i := 0
	for i < len(r.value) {
		r.value[i] = (r.value[i] || b.value[i]) && r.value[i] != b.value[i]
		i++
	}
}

// ORA - OR Register with value
func (r *Register) ORA(v interface{}) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("ORA: %s", err.Error())
	}

	i := 0
	for i < len(r.value) {
		r.value[i] = r.value[i] || b.value[i]
		i++
	}
}

// AND register with value
func (r *Register) AND(v interface{}) {
	b, err := Generate(v, len(r.value), "")
	if err != nil {
		log.Fatalf("AND: %s", err.Error())
	}

	i := 0
	for i < len(r.value) {
		r.value[i] = r.value[i] && b.value[i]
		i++
	}
}

// ROR rotates register 1 bit to the right. Returns new carry status.
func (r *Register) ROR(carry bool) bool {
	rcarry := bool(r.value[len(r.value)-1])
	copy(r.value[1:], r.value[:len(r.value)-1])
	r.value[0] = bit(carry)
	return rcarry
}

// ROL rotates register 1 bit to the left. Returns new carry status.
func (r *Register) ROL(carry bool) bool {
	rcarry := bool(r.value[0])
	copy(r.value[:len(r.value)-1], r.value[1:])
	r.value[len(r.value)-1] = bit(carry)
	return rcarry
}

// ASL (Arithmetic shift Left) shifts register one bit to the left. Returns
// the most significant bit as it was before the shift. If we think of the
// ASL operation as a multiply by two then the return value is the carry bit.
func (r *Register) ASL() bool {
	rcarry := bool(r.value[0])
	copy(r.value[:len(r.value)-1], r.value[1:])
	r.value[len(r.value)-1] = bit(false)
	return rcarry
}

// LSR (Logical Shift Right) shifts register one bit to the rigth.
// the least significant bit as it was before the shift. If we think of
// the ASL operation as a division by two then the return value is the carry bit.
func (r *Register) LSR() bool {
	rcarry := bool(r.value[len(r.value)-1])
	copy(r.value[1:], r.value[:len(r.value)-1])
	r.value[0] = bit(false)
	return rcarry
}
