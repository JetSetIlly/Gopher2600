package tia

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

// rsync is tricky but I've inpterpreted the various literature (and
// observation of the Stella emulator) in the following way:
//  - color clock phase is reset to 0 when RSYNC is triggered (see tia.go)
//	- we note that rsync is now active (active flag below)
//  - when the color clock phase reaches the end of it's phase cycle start
//		a new scanline in the normal way (colorClock.tick() in tia.go)

type rsync struct {
	active     bool
	colorClock *colorClock
}

func newRsync(cc *colorClock) *rsync {
	rs := new(rsync)
	if rs == nil {
		return nil
	}
	rs.reset()
	rs.colorClock = cc
	return rs
}

// StringTerse returns the RSYNC information in verbose format
func (rs rsync) StringTerse() string {
	if rs.isActive() {
		return fmt.Sprintf("RS=%d", rs.remainingCycles())
	}
	return "RS=-"
}

// String returns the RSYNC information in verbose format
func (rs rsync) String() string {
	if rs.isActive() {
		return fmt.Sprintf("RSYNC -> reset in %d cycle(s)\n", rs.remainingCycles())
	}
	return "RSYNC -> not set\n"
}

func (rs rsync) remainingCycles() int {
	return polycounter.MaxPhase - rs.colorClock.Phase + 1
}

func (rs rsync) isActive() bool {
	return rs.active
}

func (rs *rsync) reset() {
	rs.active = false
}

func (rs *rsync) set() {
	rs.active = true
}

func (rs rsync) check() bool {
	return rs.active && rs.colorClock.Phase == polycounter.MaxPhase
}
