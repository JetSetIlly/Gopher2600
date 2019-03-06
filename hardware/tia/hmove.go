package tia

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

// according to TIA_HW_Notes.txt the hardware equivalent of _HMOVE is a ripple
// counter that counts from 15 to zero every 4 clocks. we've interpreted this
// latter point to mean 4 clocks counted from the phase that the HMOVE was
// triggered (represented by 'phase' field)

type hmove struct {
	count      int
	phase      int
	colorClock *polycounter.Polycounter

	// latch is set whenever HMOVE is triggered and resets whenever
	// the colorClock cycles back to zero. it is used to control whether the
	// hblank period should be extended.
	latch bool

	// justset is true for a single videocycle
	justset bool
}

func newHmove(colorClock *polycounter.Polycounter) *hmove {
	hm := new(hmove)
	hm.colorClock = colorClock
	return hm
}

// MachineInfoTerse returns the HMOVE information in verbose format
func (hm hmove) MachineInfoTerse() string {
	if hm.count >= 0 {
		return fmt.Sprintf("HM=%d", hm.count)
	}
	return "HM=-"
}

// MachineInfo returns the HMOVE information in verbose format
func (hm hmove) MachineInfo() string {
	if hm.count >= 0 {
		return fmt.Sprintf("hmove: %d more tick(s)", hm.count)
	}
	return "hmove: no movement"
}

// MachineInfoInternal returns low state information about the type
func (hm hmove) MachineInfoInternal() string {
	if hm.count >= 0 {
		return fmt.Sprintf("%04b\n", hm.count)
	}
	return fmt.Sprintf("0000\n")
}

// map String to MachineInfo
func (hm hmove) String() string {
	return hm.MachineInfo()
}

// set begins the horizontal movement sequence
func (hm *hmove) set() {
	hm.latch = true
	hm.count = 15
	hm.phase = hm.colorClock.Phase
	hm.justset = true
}

// unset send the horizontal movement sequence
func (hm *hmove) unset() {
	hm.latch = false
}

// isset check to see if the horiztonal movement sequence is currently running
func (hm *hmove) isset() bool {
	return hm.latch
}

// isjustset checks to see if the horiztonal movement sequence has just started
func (hm *hmove) isjustset() bool {
	return hm.count == 15 &&
		((hm.phase < polycounter.MaxPhase && hm.phase+1 == hm.colorClock.Phase) ||
			(hm.phase == polycounter.MaxPhase && 0 == hm.colorClock.Phase))
}

// tick returns the current hmove ripple counter and whether a tick has occurred
func (hm *hmove) tick() (int, bool) {
	// if we've reached a count of -1 then no tick will ever occur
	if hm.count == -1 {
		return -1, false
	}

	// count has not yet concluded so whenever the color clock reaches the same
	// phase as when we started, return the current count and the fact that a
	// tick has occurred. reduce the current count by 1
	if hm.phase == hm.colorClock.Phase {
		ct := hm.count
		hm.count--
		return ct, true
	}

	// count has not yet concluded but nothing else has happened this video
	// cycle. return the current count and the fact that no tick has occurred.
	return hm.count, false
}
