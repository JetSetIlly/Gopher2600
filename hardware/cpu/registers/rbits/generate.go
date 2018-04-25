package rbits

import (
	"fmt"
	"reflect"
)

// Generate is used to create a register of bit length bitlen, using a value
// (v) to initialise it. v can be another register or an integer type  (int,
// uint8 or uint16). if v is nil then an unitialised register of length bitlen
// is created
func Generate(v interface{}, bitlen int) (Register, error) {
	var r Register
	var val int

	if v == nil {
		r := make(Register, bitlen)
		return r, nil
	}

	switch v := v.(type) {
	default:
		return nil, fmt.Errorf(fmt.Sprintf("value is of an unsupported type [%s]", reflect.TypeOf(v)))

	case Register:
		if len(v) > bitlen {
			return nil, fmt.Errorf("[1] value is too big (%d) for bit length of register (%d)", v.ToUint16(), bitlen)
		}

		r = make(Register, bitlen)

		// we may be copying a smaller register into a larger register so we need
		// to account for the difference
		copy(r[bitlen-len(v):], v)

		return r, nil

	case uint16:
		val = int(v)
	case uint8:
		val = int(v)
	case int:
		val = v
	}

	if bitlen == 8 && val >= 0 && val < len(bitPatterns8b) {
		r = make(Register, 8)
		copy(r, bitPatterns8b[val])
		return r, nil
	}

	if bitlen == 16 && val >= 0 && val < len(bitPatterns16b) {
		r = make(Register, 16)
		copy(r, bitPatterns16b[val])
		return r, nil
	}

	if val >= bitVals[bitlen] {
		return nil, fmt.Errorf("[2] value is too big (%d) for bit length of register (%d)", val, bitlen)
	}

	// optimally, we'll never get to this point

	return createBitPattern(val, bitlen), nil
}
