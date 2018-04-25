package r16bit

import (
	"fmt"
	"log"
)

// Register is an array of of type bit, used for register representation
type Register uint16

// Size returns the number of bits in register
func (r Register) Size() int {
	return 16
}

func (r Register) String() string {
	return fmt.Sprintf("%s (%d) [0x%04x]", r.ToString(), r.ToUint(), r.ToUint())
}

// ToUint returns value of type uint, regardless of register size
func (r Register) ToUint() uint {
	return uint(r)
}

// ToUint16 returns value of type uint16, regardless of register size
func (r Register) ToUint16() uint16 {
	return uint16(r)
}

// ToString returns the register as bit pattern (of '0' and '1')
func (r Register) ToString() string {
	return fmt.Sprintf("%016b", r)
}

// Load value into register
func (r *Register) Load(v interface{}) {
	b, err := Generate(v, 16)
	if err != nil {
		log.Fatalln(err)
	}
	*r = b
}

// Add value to register. Returns carry and overflow states -- for this native
// implementation, carry flag is ignored and return values are undefined
func (r *Register) Add(v interface{}, carry bool) (bool, bool) {
	b, err := Generate(v, 16)
	if err != nil {
		log.Fatalln(err)
	}
	*r += b
	return false, false
}
