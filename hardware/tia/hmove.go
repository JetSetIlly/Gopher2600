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
	colorClock *polycounter.Polycounter

	// latch is set whenever HMOVE is triggered and resets whenever
	// the colorClock cycles back to zero. it is used to control whether the
	// hblank period should be extended.
	latch bool
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

// EmulatorInfo returns low state information about the type
func (hm hmove) EmulatorInfo() string {
	if hm.count >= 0 {
		return fmt.Sprintf("%04b\n", hm.count)
	}
	return fmt.Sprintf("0000\n")
}

// map String to MachineInfo
func (hm hmove) String() string {
	return hm.MachineInfo()
}

// setLatch begins the horizontal movement sequence
func (hm *hmove) setLatch() {
	hm.latch = true
	hm.count = 15
}

// unsetLatch send the horizontal movement sequence
func (hm *hmove) unsetLatch() {
	hm.latch = false
}

// isLatched check to see if the horiztonal movement sequence is currently running
func (hm *hmove) isLatched() bool {
	return hm.latch
}

// tick returns the current hmove ripple counter and whether a tick has occurred
func (hm *hmove) tick() (int, bool) {
	if hm.colorClock.Phase != 0 {
		return hm.count, false
	}

	switch hm.count {
	case -1:
		return -1, false
	case 0:
		hm.count--
		return 0, true
	default:
		ct := hm.count
		hm.count--
		return ct, true
	}
}
