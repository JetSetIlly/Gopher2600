package tia

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

// rsync is tricky but I've inpterpreted the various literature (and
// observation of the Stella emulator) in the following way:
//  - color clock phase is reset to 0 when RSYNC is triggered
//	- set remainingCycles
//	- rsync.tick() is called every video cycle
//  - when the remainingCycles reaches zero during a rsync.tick(), return true.
//  - a new scanline is then begun in the normal way

type rsync struct {
	remainingCycles int
	colorClock      *polycounter.Polycounter
}

func newRsync(colorClock *polycounter.Polycounter) *rsync {
	rs := new(rsync)
	rs.colorClock = colorClock
	rs.reset()
	return rs
}

// MachineInfoTerse returns the RSYNC information in verbose format
func (rs rsync) MachineInfoTerse() string {
	if rs.isactive() {
		return fmt.Sprintf("RS=%d", rs.remainingCycles)
	}
	return "RS=-"
}

// MachineInfo returns the RSYNC information in verbose format
func (rs rsync) MachineInfo() string {
	if rs.isactive() {
		return fmt.Sprintf("rsync: reset in %d cycle(s)", rs.remainingCycles)
	}
	return "rsync: not set"
}

// map String to MachineInfo
func (rs rsync) String() string {
	return rs.MachineInfo()
}

func (rs rsync) isactive() bool {
	return rs.remainingCycles > -1
}

func (rs *rsync) reset() {
	rs.remainingCycles = -1
}

func (rs *rsync) set() {
	// after a lot of faffing and experimentation, I'm fairly sure that rsync
	// is activated after 5 cycles (well, 4 but we need to account for the
	// immediate tick in TIA)
	rs.remainingCycles = 5
	rs.colorClock.ResetPhase()
}

func (rs *rsync) isjustset() bool {
	return rs.remainingCycles == 5
}

func (rs *rsync) tick() bool {
	if rs.remainingCycles == -1 {
		return false
	}
	if rs.remainingCycles == 0 {
		rs.remainingCycles = -1
		return true
	}
	rs.remainingCycles--
	return false
}
