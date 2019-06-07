package polycounter

// polycounter implements the counting method used in the VCS TIA chip and as
// described in TIA_HW_Notes.txt
//
// there's nothing particularly noteworthy about the implementation except that
// the Count value can be used to index the predefined polycounter table, which
// maybe useful for debugging.
//
// intended to be used in conjunction with TIAClock

import "fmt"

// Polycounter counts from 0 to Limit. can be used to index a polycounter
// table
type Polycounter struct {
	Count int
	limit int
}

func (pcnt Polycounter) String() string {
	// assumes maximum limit of 2 digits
	return fmt.Sprintf("%02d", pcnt.Count)
}

// SetLimit sets the point after which the counter will return to 0
// will panic if limit is greater than 64
func (pcnt *Polycounter) SetLimit(limit int) {
	if limit < 0 {
		panic("polycounter SetLimit minimum is 0")
	}
	if limit > 64 {
		panic("polycounter SetLimit maximum is 64")
	}
	pcnt.limit = limit
}

// Reset is a convenience function to reset count value to 0
func (pcnt *Polycounter) Reset() {
	pcnt.Count = 0
}

// Tick advances the Polycounter and resets when it reaches the limit.
// returns true if counter has reset
func (pcnt *Polycounter) Tick() bool {
	if pcnt.Count == pcnt.limit {
		pcnt.Count = 0
		return true
	}
	pcnt.Count++
	return false
}
