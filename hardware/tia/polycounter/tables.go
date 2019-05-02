package polycounter

import "fmt"

// MaxPhase is the maximum value attainable by Polycounter.Phase
const MaxPhase = 3

type polycounterTable []string

// initialise the 6 bit table representing the polycounter sequence. we use to
// match the current count with the correct polycounter pattern. this is
// currently used only in the String()/ToString() functions for presentation
// purposes and when specifying the reset pattern in the call to Reset()
var table6bits polycounterTable

func init() {
	table6bits = make([]string, 64)
	var p int
	table6bits[0] = "000000"
	for i := 1; i < len(table6bits); i++ {
		p = ((p & (0x3f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x3f)<<5
		p = p & 0x3f
		table6bits[i] = fmt.Sprintf("%06b", p)
	}

	// sanity check that the table has looped correctly
	if table6bits[63] != "000000" {
		panic(fmt.Sprintf("error during 6 bit polycounter generation"))
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	table6bits[63] = "111111"
}

// LookupPattern returns the index of the specified pattern
func (tab polycounterTable) LookupPattern(pattern string) int {
	for i := 0; i < len(table6bits); i++ {
		if table6bits[i] == pattern {
			return i
		}
	}
	panic(fmt.Sprintf("could not find pattern (%s) in 6 bit lookup table", pattern))
}

// New6Bit initialises a new instance of a 6 bit polycounter
func New6Bit() *Polycounter {
	pk := new(Polycounter)
	pk.table = table6bits
	return pk
}
