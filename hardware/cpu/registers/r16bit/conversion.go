package r16bit

import "fmt"

// ToBits returns the register as bit pattern (of '0' and '1')
func (r Register) ToBits() string {
	return fmt.Sprintf("%016b", r.value)
}

// ToHex returns value as hexidecimal string
func (r Register) ToHex() string {
	if r.Size() <= 8 {
		return fmt.Sprintf("0x%02x", r.ToUint())
	}
	return fmt.Sprintf("0x%04x", r.ToUint())
}

// ToUint returns value of type uint, regardless of register size
func (r Register) ToUint() uint {
	return uint(r.value)
}

// ToUint16 returns value of type uint16, regardless of register size
func (r Register) ToUint16() uint16 {
	return uint16(r.value)
}
