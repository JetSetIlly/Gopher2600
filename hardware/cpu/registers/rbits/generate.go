package rbits

import (
	"fmt"
	"reflect"
)

// Generate is used to create a register of bit length bitlen, using a value
// (v) to initialise it. v can be another rbits.Register or an integer type
// (int, uint8 or uint16)
func Generate(v interface{}, bitlen int, label string) (Register, error) {
	var val int

	switch v := v.(type) {
	default:
		return Register{}, fmt.Errorf(fmt.Sprintf("value is of an unsupported type [%s]", reflect.TypeOf(v)))

	case Register:
		if len(v.value) > bitlen {
			return Register{}, fmt.Errorf("[1] value is too big (%d) for bit length of register (%d)", v.ToUint16(), bitlen)
		}

		r := new(Register)
		r.value = make([]bit, bitlen)
		r.label = label

		// we may be copying a smaller register into a larger register so we need
		// to account for the difference
		copy(r.value[bitlen-len(v.value):], v.value)

		return *r, nil

	case uint16:
		val = int(v)
	case uint8:
		val = int(v)
	case uint:
		val = int(v)
	case int:
		val = v
	}

	if bitlen == 8 && val >= 0 && val < len(bitPatterns8b) {
		r := new(Register)
		r.value = make([]bit, 8)
		r.label = label
		copy(r.value, bitPatterns8b[val])
		return *r, nil
	}

	if bitlen == 16 && val >= 0 && val < len(bitPatterns16b) {
		r := new(Register)
		r.value = make([]bit, 16)
		r.label = label
		copy(r.value, bitPatterns16b[val])
		return *r, nil
	}

	if val >= bitVals[bitlen] {
		return Register{}, fmt.Errorf("[2] value is too big (%d) for bit length of register (%d)", val, bitlen)
	}

	// optimally, we'll never get to this point

	r := new(Register)
	r.value = createBitPattern(val, bitlen)
	r.label = label
	return *r, nil
}
