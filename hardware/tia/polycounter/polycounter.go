package polycounter

import (
	"fmt"
)

// MaxPhase is the maximum value attainable by Polycounter.Phase
const MaxPhase = 3

// initialise the 6 bit table representing the polycounter sequence. we use to
// match the current count with the correct polycounter pattern. this is
// currently used only in the String()/ToString() functions for presentation
// purposes and when specifying the reset pattern in the call to Reset()
var table6bits []string

func init() {
	table6bits = make([]string, 64)
	var p int
	table6bits[0] = "000000"
	for i := 1; i < len(table6bits); i++ {
		p = ((p & (0x3f - 1)) >> 1) | (((p&1)^((p>>1)&1))^0x3f)<<5
		p = p & 0x3f
		table6bits[i] = fmt.Sprintf("%06b", p)
	}
	if table6bits[63] != "000000" {
		panic(fmt.Errorf("error during 6 bit polycounter generation"))
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	table6bits[63] = "111111"
}

// LookupPattern returns the index of the specified pattern
func LookupPattern(pattern string) int {
	for i := 0; i < len(table6bits); i++ {
		if table6bits[i] == pattern {
			return i
		}
	}
	panic(fmt.Errorf("could not find pattern (%s) in 6 bit lookup table", pattern))
}

// Polycounter implements the VCS method of counting. It is doesn't require
// special initialisation so is a good candidate for struct embedding
type Polycounter struct {
	// this implementation of the VCS polycounter uses a regular go-integer
	Count int

	// the phase ranges from 0 and MaxPhase
	Phase int

	// reset point is the value of count at which the polycounter should reset
	// to 0
	ResetPoint int
}

// SetResetPattern specifies the pattern at which the polycounter automatically
// resets during a Tick(). this should be called at least once or the reset
// pattern will be "000000" which is probably not what you want
func (pk *Polycounter) SetResetPattern(resetPattern string) {
	i := LookupPattern(resetPattern)
	pk.ResetPoint = i
}

// MachineInfoTerse returns the polycounter information in terse format
func (pk Polycounter) MachineInfoTerse() string {
	return fmt.Sprintf("%d@%d", pk.Count, pk.Phase)
}

// MachineInfo returns the polycounter information in verbose format
func (pk Polycounter) MachineInfo() string {
	return fmt.Sprintf("(%d) %s@%d", pk.Count, table6bits[pk.Count], pk.Phase)
}

// map String to MachineInfo
func (pk Polycounter) String() string {
	return pk.MachineInfo()
}

// ResetPhase resets the phase *only*
func (pk *Polycounter) ResetPhase() {
	pk.Phase = 0
}

// Reset leaves the polycounter in its "zero" state. resetPattern
func (pk *Polycounter) Reset() {
	pk.Count = 0
	pk.Phase = 0
}

// Tick advances the count to the next state - returns true if counter has
// looped. the force argument allows the function to be called and for the loop
// to definitely take place. we use this in the VCS during for the RSYNC check
func (pk *Polycounter) Tick(force bool) bool {
	if force || pk.Count == pk.ResetPoint && pk.Phase == MaxPhase {
		pk.Reset()
		return true
	}

	pk.Phase++
	if pk.Phase > MaxPhase {
		pk.ResetPhase()
		pk.Count++
	}

	return false
}

// Match check whether polycounter is at the given count, any phase
func (pk Polycounter) Match(count int) bool {
	return pk.Count == count
}

// MatchEnd checks whether polycounter is at the *end* (ie. last phase) of the given count
func (pk Polycounter) MatchEnd(count int) bool {
	return pk.Count == count && pk.Phase == MaxPhase
}

// MatchBeginning checks whether polycounter is at the *beginning* (ie. first phase) of the given count
func (pk Polycounter) MatchBeginning(count int) bool {
	return pk.Count == count && pk.Phase == 0
}
