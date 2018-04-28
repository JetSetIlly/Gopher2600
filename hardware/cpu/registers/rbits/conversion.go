package rbits

import (
	"fmt"
	"log"
	"strings"
)

// bitVals is a lookup table for pow(2,n)
var bitVals = [...]int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}

func max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

// ToBits returns the register as bit pattern (of '0' and '1')
func (r Register) ToBits() string {
	s := make([]string, len(r.value))
	i := 0
	for i < len(r.value) {
		if r.value[i] {
			s[i] = "1"
		} else {
			s[i] = "0"
		}
		i++
	}
	return strings.Join(s, "")
}

// ToHex returns value as hexidecimal string
func (r Register) ToHex() string {
	if r.Size() <= 8 {
		return fmt.Sprintf("0x%02x", r.ToUint())
	}
	return fmt.Sprintf("0x%04x", r.ToUint())
}

// ToUint returns value as type uint, regardless of register size
func (r Register) ToUint() uint {
	var v uint

	i := len(r.value) - 1
	j := 0
	for i >= 0 {
		if r.value[i] {
			v += uint(bitVals[j])
		}
		i--
		j++
	}

	return v
}

// ToUint16 returns value of size uint16, regardless of register size
func (r Register) ToUint16() uint16 {
	if len(r.value) > 16 {
		log.Print("ToUint16: register wider than 16 bits. information may be lost")
	}
	return uint16(r.ToUint())
}

// ToUint8 returns value of size uint8, regardless of register size
func (r Register) ToUint8() uint8 {
	if len(r.value) > 8 {
		log.Print("ToUint8: register wider than 8 bits. information may be lost")
	}
	return uint8(r.ToUint())
}
