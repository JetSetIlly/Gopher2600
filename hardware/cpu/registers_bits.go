package cpu

import (
	"fmt"
	"log"
	"strings"
)

type bit bool

// bitVals is a lookup table for pow(2,n)
var bitVals = [...]int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}

func max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

// ToUint returns value as type uint, regardless of register size
func (r Register) ToUint() uint {
	var v uint

	i := len(r) - 1
	j := 0
	for i >= 0 {
		if r[i] != false {
			v += uint(bitVals[j])
		}
		i--
		j++
	}

	return v
}

// ToBits returns the register as bit pattern (of '0' and '1')
func (r Register) ToBits() string {
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

// ToUint16 returns value of size uint16, regardless of register size
func (r Register) ToUint16() uint16 {
	if len(r) > 16 {
		log.Print("ToUint16: register wider than 16 bits. information may be lost")
	}
	return uint16(r.ToUint())
}

// ToUint8 returns value of size uint8, regardless of register size
func (r Register) ToUint8() uint8 {
	if len(r) > 8 {
		log.Print("ToUint8: register wider than 8 bits. information may be lost")
	}
	return uint8(r.ToUint())
}

// generateRegister is used to create a register of bit length bitlen, using a
// value (v) to initialise it v can be another register or an integer type
// (int, uint8 or uint16). if v is nil then a unitialised register of length
// bitlen is created; although, if this is the effect you want, then it is
// suggested that a plain "make(Register, bitlen)" is used instead
//
// when a register is supplied, the register will be reused unless the bit
// length is wrong, in which case a copy is made. a pointer to a register
// indicates that you definitely want a new copy of the register regardless of
// bit length
func generateRegister(v interface{}, bitlen int) (Register, error) {
	var r Register
	var val uint16

	if v == nil {
		r := make(Register, bitlen)
		return r, nil
	}

	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("value is of an unsupported type")
	case *Register:
		r = make(Register, bitlen)
		val = v.ToUint16()
	case Register:
		// reuse register if possible
		if len(v) == bitlen {
			return v, nil
		}

		if len(v) > bitlen {
			return nil, fmt.Errorf("[0] value is too big (%d) for bit length of register (%d)", v.ToUint16(), bitlen)
		}

		val = v.ToUint16()
		r = make(Register, max(16, bitlen))
	case uint16:
		if int(v) >= bitVals[bitlen] {
			return nil, fmt.Errorf("[1] value is too big (%d) for bit length of register (%d)", v, bitlen)
		}
		val = uint16(v)
		r = make(Register, max(16, bitlen))
	case uint8:
		if int(v) >= bitVals[bitlen] {
			return nil, fmt.Errorf("[2] value is too big (%d) for bit length of register (%d)", v, bitlen)
		}
		val = uint16(v)
		r = make(Register, max(8, bitlen))
	case int:
		if v >= bitVals[bitlen] {
			return nil, fmt.Errorf("[3] value is too big (%d) for bit length of register (%d)", v, bitlen)
		}
		val = uint16(v)
		r = make(Register, max(8, bitlen))
	}

	// create bit pattern
	i := 0
	j := len(r) - 1
	for j >= 0 {
		bv := uint16(bitVals[j])
		if val/bv != 0 {
			r[i] = true
			val = val - bv
		}
		i++
		j--
	}

	// belt & braces test
	if val != 0 {
		return nil, fmt.Errorf("(2) value is too big (%d) for bit length of register (%d)", v, bitlen)
	}

	return r, nil
}
