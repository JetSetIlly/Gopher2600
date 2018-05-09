package tia

import "fmt"

// according to TIA_HW_Notes.txt the hardware equivalent of _HMOVE is a ripple
// counter that counts from 15 to zero every 4 clocks. we've interpreted this
// latter point to mean 4 clocks counted from the phase that the HMOVE was
// triggered (represented by 'phase' property)

type hmove struct {
	count      int
	phase      int
	colorClock *colorClock
}

func newHmove(cc *colorClock) *hmove {
	hm := new(hmove)
	if hm == nil {
		return nil
	}
	hm.reset()
	hm.colorClock = cc
	return hm
}

// String returns the HMOVE information in verbose format
func (hm hmove) StringTerse() string {
	if hm.isActive() {
		return fmt.Sprintf("HM=%d", hm.count)
	}
	return "HM=-"
}

// String returns the HMOVE information in verbose format
func (hm hmove) String() string {
	if hm.isActive() {
		return fmt.Sprintf("HMOVE -> %d more tick(s)\n", hm.count)
	}
	return "HMOVE -> no movement\n"
}

func (hm hmove) isActive() bool {
	return hm.count > -1
}

func (hm *hmove) reset() {
	hm.count = -1
	hm.phase = -1
}

func (hm *hmove) set() {
	hm.count = 15
	hm.phase = hm.colorClock.Phase
}

func (hm *hmove) tick() int {
	// if hmove is active, when color clock phase cycles to where it was when
	// hmove.set() was called reduce the hmove count
	ct := 0
	if hm.count > 0 && hm.phase == hm.colorClock.Phase {
		ct = hm.count
		hm.count--
	}
	return ct
}
