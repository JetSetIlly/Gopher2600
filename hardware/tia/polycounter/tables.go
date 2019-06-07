package polycounter

import "fmt"

type table []string

// Table is the polycounter sequence over the space of 6 bits
var Table table

// initialise the 6 bit table representing the polycounter sequence. we use to
// match the current count with the correct polycounter pattern. this is
// currently used only in the String()/ToString() functions for presentation
// purposes and when specifying the reset pattern in the call to Reset()
func init() {
	Table = make([]string, 64)
	var p int
	Table[0] = "000000"
	for i := 1; i < len(Table); i++ {
		p = ((p & (0x3f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x3f)<<5
		p = p & 0x3f
		Table[i] = fmt.Sprintf("%06b", p)
	}

	// sanity check that the table has looped correctly
	if Table[63] != "000000" {
		panic(fmt.Sprintf("error during 6 bit polycounter generation"))
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	Table[63] = "111111"
}

// LookupPattern returns the index of the specified pattern
func (tab table) LookupPattern(pattern string) int {
	for i := 0; i < len(Table); i++ {
		if Table[i] == pattern {
			return i
		}
	}
	panic(fmt.Sprintf("could not find pattern (%s) in 6 bit lookup table", pattern))
}
