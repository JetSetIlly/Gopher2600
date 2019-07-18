package polycounter

// polycounter implements the counting method used in the VCS TIA chip and as
// described in "TIA_HW_Notes.txt"

import (
	"fmt"
)

// Polycounter counts from 0 to Limit. can be used to index a polycounter
// table
type Polycounter struct {
	Count int
}

func (pcnt Polycounter) String() string {
	// assumes maximum limit of 2 digits
	return fmt.Sprintf("%s (%02d)", Table[pcnt.Count], pcnt.Count)
}

// Reset is a convenience function to reset count value to 0
func (pcnt *Polycounter) Reset() {
	pcnt.Count = 0
}

// Tick advances the Polycounter and resets when it reaches the limit.
// returns true if counter has reset
func (pcnt *Polycounter) Tick() bool {
	pcnt.Count++
	if pcnt.Count == 63 {
		pcnt.Count = 0
		return true
	}
	return false
}
