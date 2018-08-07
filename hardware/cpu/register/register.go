package register

import (
	"fmt"
	"reflect"
)

// Register is an array of of type bit, used for register representation
type Register struct {
	value      uint32
	size       int
	label      string
	shortLabel string

	signBit uint32
	vbit    uint32
	mask    uint32

	hexformat string
	binformat string
}

// NewAnonRegister initialises a new register without a name
func NewAnonRegister(value interface{}, size int) *Register {
	return NewRegister(value, size, "", "")
}

// NewRegister creates a new register of a givel size and name, and initialises
// the value
func NewRegister(value interface{}, size int, label string, shortLabel string) *Register {
	if size != 8 && size != 16 {
		panic(fmt.Errorf("cannot create register (%s) - unsupported bit size (%d)", label, size))
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
		panic(fmt.Errorf("cannot create register (%s) - unsupported value type (%s)", label, reflect.TypeOf(value)))
	}

	r.size = size
	r.label = label
	r.shortLabel = shortLabel

	if size == 8 {
		r.signBit = 0x00000080
		r.vbit = 0x00000040
		r.mask = 0x000000FF
		r.hexformat = "%#02x"
		r.binformat = "%08b"
	} else if size == 16 {
		r.signBit = 0x00008000
		r.vbit = 0x00004000
		r.mask = 0x0000FFFF
		r.hexformat = "%#04x"
		r.binformat = "%016b"
	}

	return r
}

// Size returns the number of bits in register
func (r Register) Size() int {
	return 8
}

// IsNegative checks the sign bit of the register
func (r Register) IsNegative() bool {
	return r.value&r.signBit == r.signBit
}

// IsZero checks if register is zero
func (r Register) IsZero() bool {
	return r.value == 0
}

// IsBitV is used by the BIT instruction and returns the state of Bit6 (the bit
// next to the sign bit. it's a bit odd because it is only ever used by the BIT
// instruction and the BIT instruction only ever uses 8 bit registers.
// none-the-less, we've generalised it so it can be used with 16 bit registers
// too (for completion)
func (r Register) IsBitV() bool {
	return r.value&r.vbit == r.vbit
}

// FromInt returns the string representation of an arbitrary integer
func (r Register) FromInt(v interface{}) string {
	switch v.(type) {
	case int:
		tr := NewRegister(v, r.size, r.label, r.shortLabel)
		return fmt.Sprintf("%s=%s", tr.shortLabel, tr.ToHex())
	default:
		return r.shortLabel
	}
}

// Label returns the verbose label of the register
func (r Register) Label() string {
	return r.label
}

// ShortLabel returns the terse labelname of the register
func (r Register) ShortLabel() string {
	return r.shortLabel
}

// MachineInfoTerse returns the register information in terse format
func (r Register) MachineInfoTerse() string {
	return fmt.Sprintf("%s=%s", r.shortLabel, r.ToHex())
}

// MachineInfo returns the register information in verbose format
func (r Register) MachineInfo() string {
	return fmt.Sprintf("%s: %d [%s] %s", r.label, r.value, r.ToHex(), r.ToBits())
}

// map String to MachineInfo
func (r Register) String() string {
	return r.MachineInfo()
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

// Value returns the canonical value for the register -- implements target
// interface
func (r Register) Value() interface{} {
	return r.ToInt()
}
