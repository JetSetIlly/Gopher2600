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
}

func newHmove(colorClock *polycounter.Polycounter) *hmove {
	hm := new(hmove)
	hm.reset()
	hm.colorClock = colorClock
	return hm
}

// MachineInfoTerse returns the HMOVE information in verbose format
func (hm hmove) MachineInfoTerse() string {
	if hm.isActive() {
		return fmt.Sprintf("HM=%d", hm.count)
	}
	return "HM=-"
}

// MachineInfo returns the HMOVE information in verbose format
func (hm hmove) MachineInfo() string {
	if hm.isActive() {
		return fmt.Sprintf("HMOVE -> %d more tick(s)", hm.count)
	}
	return "HMOVE -> no movement"
}

// map String to MachineInfo
func (hm hmove) String() string {
	return hm.MachineInfo()
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

func (hm *hmove) tick() (ct int, tick bool) {
	// if hmove is active, when color clock phase cycles to where it was when
	// hmove.set() was called reduce the hmove count
	if hm.count > 0 && hm.phase == hm.colorClock.Phase {
		ct = hm.count
		hm.count--
		tick = true
	}
	return ct, tick
}
