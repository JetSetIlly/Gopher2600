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

	// whether or not hmove is active or not is distinct from the hmove count.
	// the latch is reset whever the colorClock cycles but we still need to
	// complete the movement of the sprites with the tick() function, which is
	// governed by the count
	latch bool
}

func newHmove(colorClock *polycounter.Polycounter) *hmove {
	hm := new(hmove)
	hm.colorClock = colorClock
	return hm
}

// MachineInfoTerse returns the HMOVE information in verbose format
func (hm hmove) MachineInfoTerse() string {
	if hm.latch {
		return fmt.Sprintf("HM=%d", hm.count)
	}
	return "HM=-"
}

// MachineInfo returns the HMOVE information in verbose format
func (hm hmove) MachineInfo() string {
	if hm.latch {
		return fmt.Sprintf("hmove: %d more tick(s)", hm.count)
	}
	return "hmove: no movement"
}

// MachineInfoInternal returns low state information about the type
func (hm hmove) MachineInfoInternal() string {
	if hm.latch {
		return fmt.Sprintf("%04b\n", hm.count)
	}
	return fmt.Sprintf("0000\n")
}

// map String to MachineInfo
func (hm hmove) String() string {
	return hm.MachineInfo()
}

func (hm *hmove) set() {
	hm.latch = true
	hm.count = 15
	hm.phase = hm.colorClock.Phase
}

func (hm *hmove) unset() {
	hm.latch = false
}

func (hm *hmove) isset() bool {
	return hm.latch
}

func (hm *hmove) tick() (int, bool) {
	if hm.count == -1 {
		return 0, true
	}

	if hm.count <= 0 {
		return hm.count, false
	}

	if hm.phase == hm.colorClock.Phase {
		ct := hm.count
		hm.count--
		return ct, true
	}

	return hm.count, false
}
