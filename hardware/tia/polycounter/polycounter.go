package polycounter

import (
	"fmt"
	"math"
)

var polycounter6bits []string

func init() {
	polycounter6bits = make([]string, 64)
	var p uint32
	polycounter6bits[0] = "000000"
	for i := 1; i < len(polycounter6bits); i++ {
		p = tick(p, 0x3f)
		polycounter6bits[i] = fmt.Sprintf("%06b", p)
	}
	if polycounter6bits[63] != "000000" {
		panic("error during 6 bit polycounter generation")
	}

	// for the sake of accuracy, force the final value to be the "error value"
	// this is an invalid value and the counter will cycle when we reach it
	polycounter6bits[63] = "111111"
}

// tick returns the next value on from the "val" argument
func tick(val uint32, mask uint32) uint32 {
	val = ((val & (mask - 1)) >> 1) | (((val&1)^((val>>1)&1))^mask)<<5
	return val & mask
}

// Polycounter implements the VCS method of counting.
type Polycounter struct {
	count uint32
	phase int

	resetPoint uint32

	numBits   int
	numPhases int

	mask      uint32
	binformat string
}

func (pk Polycounter) String() string {
	var polycountStr string
	if pk.numBits == 6 {
		polycountStr = polycounter6bits[pk.count]
	} else {
		var val uint32

		if pk.count == 0 {
			val = pk.mask
		} else {
			val = pk.count - 1
		}

		polycountStr = fmt.Sprintf(pk.binformat, tick(val, pk.mask))
	}

	return fmt.Sprintf("%s/%d (%d/%d)", polycountStr, pk.phase, pk.count, pk.phase)
}

// NewPolycounter is the preferred method of initialisation for polycounters
// -- use New6BitPolycounter if possible (always possible in the VCS)
func NewPolycounter(numBits int, numPhases int) *Polycounter {
	pk := new(Polycounter)
	if pk == nil {
		return nil
	}
	pk.numBits = numBits
	pk.mask = uint32(math.Pow(2.0, float64(pk.numBits)) - 1)
	pk.resetPoint = pk.mask
	pk.binformat = fmt.Sprintf("%%0%db", pk.numBits)
	pk.numPhases = numPhases
	return pk
}

// New6BitPolycounter is the preferred method of initialisation for 6bit
// polycounters. the cyclePattern argument allows you to specify the count
// at which the counter resets.
func New6BitPolycounter(cyclePattern string) *Polycounter {
	pk := new(Polycounter)

	pk.numBits = 6
	pk.mask = 0x3f
	pk.binformat = "%06b"
	pk.numPhases = 4

	// find cyclePattern in polycounter6bits
	for i := 1; i < len(polycounter6bits); i++ {
		if polycounter6bits[i] == cyclePattern {
			pk.resetPoint = uint32(i)
			break
		}
	}

	return pk
}

// Reset leaves the polycounter in its "zero" state
func (pk *Polycounter) Reset() {
	pk.count = 0
	pk.phase = 0
}

// Tick advances the count to the next state - returns true if counter has
// reset
func (pk *Polycounter) Tick() bool {
	pk.phase++
	if pk.phase == pk.numPhases {
		pk.phase = 0
		pk.count++
		if pk.count == pk.resetPoint {
			pk.count = 0
			return true
		}
	}
	return false
}
