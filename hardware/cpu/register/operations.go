package register

import (
	"fmt"
)

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
	default:
		panic(fmt.Errorf("unsupported value type (%T)", v))
	}
}

// Add value to register. Returns carry and overflow states
func (r *Register) Add(v interface{}, carry bool) (bool, bool) {
	var preNeg, postNeg bool

	preNeg = r.IsNegative()

	switch v := v.(type) {
	case *Register:
		r.value += v.value
		if carry {
			r.value++
		}

		postNeg = v.IsNegative()
	case int:
		r.value += uint32(v)
		if carry {
			r.value++
		}
		postNeg = uint32(v)&r.signBit == r.signBit
	case uint8:
		r.value += uint32(v)
		if carry {
			r.value++
		}
		postNeg = uint32(v)&r.signBit == r.signBit
	case uint16:
		r.value += uint32(v)
		if carry {
			r.value++
		}
		postNeg = uint32(v)&r.signBit == r.signBit
	default:
		panic(fmt.Errorf("unsupported value type (%T)", v))
	}

	carry = ^r.mask&r.value != 0
	overflow := !r.IsNegative() && preNeg && postNeg

	if carry {
		r.value &= r.mask
	}

	return carry, overflow
}

// Subtract value from register. Returns carry and overflow states
func (r *Register) Subtract(v interface{}, carry bool) (bool, bool) {
	var val uint16

	switch v := v.(type) {
	case *Register:
		val = uint16(v.value)
	case int:
		val = uint16(v)
	case uint8:
		val = uint16(v)
	default:
		panic(fmt.Errorf("unsupported value type (%T)", v))
	}

	// no need to do anything if operand is zero
	if val == 0 {
		return carry, false
	}

	// two's complement
	val = ^val
	if carry {
		val++
	}
	val &= uint16(r.mask)

	return r.Add(val, false)
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
		panic(fmt.Errorf("unsupported value type (%T)", v))
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
		panic(fmt.Errorf("unsupported value type (%T)", v))
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
		panic(fmt.Errorf("unsupported value type (%T)", v))
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
