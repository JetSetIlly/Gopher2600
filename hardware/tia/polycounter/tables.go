package polycounter

import "fmt"

type table []string

// Poly6Bit is the polycounter sequence over the space of 6 bits
var Poly6Bit table

// Poly5Bit is the polycounter sequence over the space of 5 bits
// (used by audio generator)
var Poly5Bit table

// Poly4Bit is the polycounter sequence over the space of 4 bits
// (used by audio generator)
var Poly4Bit table

// initialise the 6 bit table representing the polycounter sequence. we use to
// match the current count with the correct polycounter pattern. this is
// currently used only in the String()/ToString() functions for presentation
// purposes and when specifying the reset pattern in the call to Reset()
func init() {
	var p int

	// poly 6 bit generation

	Poly6Bit = make([]string, 64)
	Poly6Bit[0] = "000000"
	for i := 1; i < len(Poly6Bit); i++ {
		p = ((p & (0x3f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x3f)<<5
		p = p & 0x3f
		Poly6Bit[i] = fmt.Sprintf("%06b", p)
	}

	// sanity check that the table has looped correctly
	if Poly6Bit[63] != "000000" {
		panic("error during 6 bit polycounter generation")
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	Poly6Bit[63] = "111111"

	// poly 5 bit generation

	Poly5Bit = make([]string, 22)
	Poly5Bit[0] = "00000"
	for i := 1; i < len(Poly5Bit); i++ {
		p = ((p & (0x1f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x1f)<<4
		p = p & 0x1f
		Poly5Bit[i] = fmt.Sprintf("%05b", p)
	}

	// sanity check that the table has looped correctly
	if Poly5Bit[21] != "00000" {
		panic("error during 5 bit polycounter generation")
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	Poly5Bit[21] = "11111"

	// poly 4 bit generation

	Poly4Bit = make([]string, 16)
	Poly4Bit[0] = "0000"
	for i := 1; i < len(Poly4Bit); i++ {
		p = ((p & (0x0f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x0f)<<3
		p = p & 0x0f
		Poly4Bit[i] = fmt.Sprintf("%04b", p)
	}

	// sanity check that the table has looped correctly
	if Poly4Bit[15] != "0000" {
		panic("error during 5 bit polycounter generation")
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	Poly4Bit[15] = "1111"
}

// LookupPattern returns the index of the specified pattern
func (tab table) LookupPattern(pattern string) int {
	for i := 0; i < len(Poly6Bit); i++ {
		if Poly6Bit[i] == pattern {
			return i
		}
	}
	panic(fmt.Sprintf("could not find pattern (%s) in 6 bit lookup table", pattern))
}
