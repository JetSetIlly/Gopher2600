package register

import (
	"fmt"
	"math"
	"reflect"
)

// !!TODO: handle "invalid" BCD values (ie. nibble values A to F) correctly

// Register is an array of of type bit, used for register representation
type Register struct {
	value      uint32
	size       uint
	label      string
	shortLabel string

	signBit uint32
	vbit    uint32
	mask    uint32

	hexformat string
	binformat string
}

// NewAnonRegister initialises a new register without a name
func NewAnonRegister(value interface{}, size uint) *Register {
	return NewRegister(value, size, "", "")
}

// NewRegister creates a new register of a givel size and name, and initialises
// the value
func NewRegister(value interface{}, size uint, label string, shortLabel string) *Register {
	if size < 2 || size > 32 {
		panic(fmt.Sprintf("cannot create register (%s) - unsupported bit size (%d)", label, size))
	}

	r := new(Register)

	switch value := value.(type) {
	case *Register:
		r.value = value.value
	case int:
		r.value = uint32(value)
	case uint:
		r.value = uint32(value)
	case uint8:
		r.value = uint32(value)
	case uint16:
		r.value = uint32(value)
	default:
		panic(fmt.Sprintf("cannot create register (%s) - unsupported value type (%s)", label, reflect.TypeOf(value)))
	}

	r.label = label
	r.shortLabel = shortLabel
	r.size = size
	r.signBit = 1 << (size - 1)
	r.vbit = 1 << (size - 2)
	r.mask = (1 << size) - 1
	r.binformat = fmt.Sprintf("%%0%dbb", r.size)
	r.hexformat = fmt.Sprintf("%%#0%dx", int(math.Ceil(float64(r.size)/4.0)))

	return r
}

func (r Register) String() string {
	return fmt.Sprintf("%s=%s", r.shortLabel, r.ToHex())
	//return fmt.Sprintf("%s: %d [%s] %s", r.label, r.value, r.ToHex(), r.ToBits())
}

// Size returns the number of bits in register
func (r Register) Size() uint {
	return r.size
}

// IsNegative checks the sign bit of the register
func (r Register) IsNegative() bool {
	return r.value&r.signBit == r.signBit
}

// IsZero checks if register is zero
func (r Register) IsZero() bool {
	return r.value == 0
}

// IsBitV returns the state of the second MSB
func (r Register) IsBitV() bool {
	return r.value&r.vbit == r.vbit
}

// ToBits returns the register as bit pattern (of '0' and '1')
func (r Register) ToBits() string {
	return fmt.Sprintf(r.binformat, r.value)
}

// ToHex returns value as hexidecimal string
func (r Register) ToHex() string {
	return fmt.Sprintf(r.hexformat, r.ToUint())
}

// ToInt returns value of type int, regardless of register size
func (r Register) ToInt() int {
	return int(r.value)
}

// ToUint returns value of type uint, regardless of register size
func (r Register) ToUint() uint {
	return uint(r.value)
}

// ToUint8 returns value of type uint16, regardless of register size
func (r Register) ToUint8() uint8 {
	return uint8(r.value)
}

// ToUint16 returns value of type uint16, regardless of register size
func (r Register) ToUint16() uint16 {
	return uint16(r.value)
}

// Label implements the target interface
func (r Register) Label() string {
	return r.label
}

// ShortLabel implements the target interface
func (r Register) ShortLabel() string {
	return r.shortLabel
}

// Value implements the target interface
func (r Register) Value() interface{} {
	return r.ToInt()
}

// FormatValue implements the target interface
func (r Register) FormatValue(fv interface{}) string {
	return fmt.Sprintf(r.hexformat, fv)
}
