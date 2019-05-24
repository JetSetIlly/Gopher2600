package register

import (
	"fmt"
)

// note that none of these values fail if value is too big for regsuter. the
// value is simply masked and stored

// Load value into register
func (r *Register) Load(v interface{}) {
	switch v := v.(type) {
	case *Register:
		r.value = v.value & r.mask
	case int:
		r.value = uint32(v) & r.mask
	case uint8:
		r.value = uint32(v) & r.mask
	case uint16:
		r.value = uint32(v) & r.mask
	case uint32:
		r.value = uint32(v) & r.mask
	case uint64:
		r.value = uint32(v) & r.mask
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}
}

// Add value to register. Returns carry and overflow states
func (r *Register) Add(v interface{}, carry bool) (bool, bool) {
	var rval, vval uint32

	// note value of register before we change it
	rval = r.value

	switch v := v.(type) {
	case *Register:
		vval = uint32(v.value)
		r.value += vval
		if carry {
			r.value++
		}
	case int:
		vval = uint32(v)
		r.value += vval
		if carry {
			r.value++
		}
	case uint8:
		vval = uint32(v)
		r.value += vval
		if carry {
			r.value++
		}
	case uint16:
		vval = uint32(v)
		r.value += vval
		if carry {
			r.value++
		}
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}

	// decide on overflow flag
	// -- notes from Ken Shirriff's blog: "The 6502 overflow flag explained
	// mathematically"
	overflow := ((rval ^ r.value) & (vval ^ r.value) & 0x80) != 0

	// decide on carry flag
	carry = ^r.mask&r.value != 0
	if carry {
		r.value &= r.mask
	}

	return carry, overflow
}

// Subtract value from register. Returns carry and overflow states
func (r *Register) Subtract(v interface{}, carry bool) (bool, bool) {
	var vval uint16

	switch v := v.(type) {
	case *Register:
		vval = uint16(v.value)
	case int:
		vval = uint16(v)
	case uint8:
		vval = uint16(v)
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}

	// one's complement
	vval = ^vval
	vval &= uint16(r.mask)

	return r.Add(vval, carry)
}

// AND value with register
func (r *Register) AND(v interface{}) {
	switch v := v.(type) {
	case *Register:
		r.value &= v.value
	case int:
		r.value &= uint32(v)
	case uint8:
		r.value &= uint32(v)
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}
	r.value &= r.mask
}

// ASL (arithmetic shift left) shifts register one bit to the left. Returns
// the most significant bit as it was before the shift. If we think of the
// ASL operation as a multiply by two then the return value is the carry bit.
func (r *Register) ASL() bool {
	carry := r.IsNegative()
	r.value <<= 1
	r.value &= r.mask
	return carry
}

// EOR (exclusive or) value with register
func (r *Register) EOR(v interface{}) {
	switch v := v.(type) {
	case *Register:
		r.value ^= v.value
	case int:
		r.value ^= uint32(v)
	case uint8:
		r.value ^= uint32(v)
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}
	r.value &= r.mask
}

// LSR (logical shift right) shifts register one bit to the right.
// the least significant bit as it was before the shift. If we think of
// the ASL operation as a division by two then the return value is the carry bit.
func (r *Register) LSR() bool {
	carry := r.value&1 == 1
	r.value >>= 1
	r.value &= r.mask
	return carry
}

// ORA (non-exclusive or) value with register
func (r *Register) ORA(v interface{}) {
	switch v := v.(type) {
	case *Register:
		r.value |= v.value
	case int:
		r.value |= uint32(v)
	case uint8:
		r.value |= uint32(v)
	default:
		panic(fmt.Sprintf("unsupported value type (%T)", v))
	}
	r.value &= r.mask
}

// ROL rotates register 1 bit to the left. Returns new carry status.
func (r *Register) ROL(carry bool) bool {
	retCarry := r.IsNegative()
	r.value <<= 1
	if carry {
		r.value |= 1
	}
	r.value &= r.mask
	return retCarry
}

// ROR rotates register 1 bit to the right. Returns new carry status.
func (r *Register) ROR(carry bool) bool {
	retCarry := r.value&1 == 1
	r.value >>= 1
	if carry {
		r.value |= r.signBit
	}
	r.value &= r.mask
	return retCarry
}
