package r16bit

import (
	"fmt"
	"reflect"
)

// Generate is used to create a register of bit length bitlen, using a value
// (v) to initialise it. v can be another r16bit.Register or an integer type
// (int, uint8 or uint16)
func Generate(v interface{}, bitlen int, label string) (Register, error) {
	if bitlen != 16 {
		return Register{}, fmt.Errorf("bitlen of %d not allowed; only 16 bits allowed with this implementation", bitlen)
	}

	switch v := v.(type) {
	case Register:
		return v, nil
	case uint16:
		return Register{value: v, label: label}, nil
	case uint8:
		return Register{value: uint16(v), label: label}, nil
	case uint:
		return Register{value: uint16(v), label: label}, nil
	case int:
		return Register{value: uint16(v), label: label}, nil
	}

	return Register{}, fmt.Errorf(fmt.Sprintf("value is of an unsupported type [%s]", reflect.TypeOf(v)))
}
