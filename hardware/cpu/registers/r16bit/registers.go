package r16bit

import (
	"fmt"
	"log"
)

// Register is an array of of type bit, used for register representation
type Register struct {
	value uint16
	label string
}

// Size returns the number of bits in register
func (r Register) Size() int {
	return 16
}

// Label returns the label assigned to the register
func (r Register) Label() string {
	return r.label
}

// Load value into register
func (r *Register) Load(v interface{}) {
	b, err := Generate(v, 16, "")
	if err != nil {
		log.Fatalln(err)
	}
	r.value = b.value
}

// Add value to register. Returns carry and overflow states -- for this native
// implementation, carry flag is ignored and return values are undefined
func (r *Register) Add(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, 16, "")
	if err != nil {
		log.Fatalln(err)
	}
	r.value += b.value
	return false, false
}

// IsNegative checks the sign bit of the register
func (r Register) IsNegative() bool {
	return r.value&0x8000 == 0x8000
}

// IsZero checks if register is zero
func (r Register) IsZero() bool {
	return r.value == 0
}

func (r Register) String() string {
	return fmt.Sprintf("%s: %d [%s] %s", r.label, r.ToUint(), r.ToHex(), r.ToBits())
}

// ToString returns the string representation of an aribtrary value
func (r Register) ToString(v interface{}) string {
	vr, err := Generate(v, r.Size(), r.Label())
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%v", vr)
}
