package registers

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

// ToString returns the register as bit pattern (of '0' and '1')
func (r Bits) ToString() string {
	s := make([]string, len(r))
	i := 0
	for i < len(r) {
		if r[i] {
			s[i] = "1"
		} else {
			s[i] = "0"
		}
		i++
	}
	return strings.Join(s, "")
}

// ToUint returns value as type uint, regardless of register size
func (r Bits) ToUint() uint {
	var v uint

	i := len(r) - 1
	j := 0
	for i >= 0 {
		if r[i] {
			v += uint(bitVals[j])
		}
		i--
		j++
	}

	return v
}

// ToUint16 returns value of size uint16, regardless of register size
func (r Bits) ToUint16() uint16 {
	if len(r) > 16 {
		log.Print("ToUint16: register wider than 16 bits. information may be lost")
	}
	return uint16(r.ToUint())
}

// ToUint8 returns value of size uint8, regardless of register size
func (r Bits) ToUint8() uint8 {
	if len(r) > 8 {
		log.Print("ToUint8: register wider than 8 bits. information may be lost")
	}
	return uint8(r.ToUint())
}

// Generate is used to create a register of bit length bitlen, using a value
// (v) to initialise it. v can be another register or an integer type  (int,
// uint8 or uint16). if v is nil then a unitialised register of length
// bitlen is created; although, if this is the effect you want, then it is
// suggested that a plain "make(Register, bitlen)" is used instead
func Generate(v interface{}, bitlen int) (Bits, error) {
	var r Bits
	var val int

	if v == nil {
		r := make(Bits, bitlen)
		return r, nil
	}

	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("value is of an unsupported type")

	case Bits:
		if len(v) > bitlen {
			return nil, fmt.Errorf("[1] value is too big (%d) for bit length of register (%d)", v.ToUint16(), bitlen)
		}

		r = make(Bits, bitlen)

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

	// I have no idea (none) why the following doesn't work - why can I not use
	// val to index the bitPattern arrays? why does it always return entry 0?
	if bitlen == 8 && val >= 0 && val < len(bitPatterns8b) {
		r = make(Bits, 8)
		copy(r, bitPatterns8b[val])
		return r, nil
	}

	if bitlen == 16 && val >= 0 && val < len(bitPatterns16b) {
		r = make(Bits, 16)
		copy(r, bitPatterns16b[val])
		return r, nil
	}

	if val >= bitVals[bitlen] {
		return nil, fmt.Errorf("[2] value is too big (%d) for bit length of register (%d)", val, bitlen)
	}

	// optimally, we'll never get to this point

	return createBitPattern(val, bitlen), nil
}
