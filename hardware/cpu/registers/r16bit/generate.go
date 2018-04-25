package r16bit

import (
	"fmt"
	"reflect"
)

// Generate is used to create a register of bit length bitlen, using a value
// (v) to initialise it. v can be another (native) register or an integer type
// (int, uint8 or uint16).
func Generate(v interface{}, bitlen int) (Register, error) {
	if bitlen != 16 {
		return 0, fmt.Errorf("bitlen of %d not allowed; only 16 bits allowed with this implementation", bitlen)
	}

	switch v := v.(type) {
	case Register:
		return v, nil
	case uint16:
		return Register(v), nil
	case uint8:
		return Register(v), nil
	case int:
		return Register(v), nil
	}

	return 0, fmt.Errorf(fmt.Sprintf("value is of an unsupported type [%s]", reflect.TypeOf(v)))
}
