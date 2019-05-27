package riot

import (
	"gopher2600/hardware/memory"
)

// RIOT contains all the sub-components of the VCS RIOT sub-system
type RIOT struct {
	mem memory.ChipBus

	Timer *timer
}

// NewRIOT creates a RIOT, to be used in a VCS emulation
func NewRIOT(mem memory.ChipBus) *RIOT {
	riot := &RIOT{mem: mem}
	riot.Timer = newTimer(mem)

	return riot
}

// MachineInfoTerse returns the RIOT information in terse format
func (riot RIOT) MachineInfoTerse() string {
	return riot.Timer.MachineInfoTerse()
}

// MachineInfo returns the RIOT information in verbose format
func (riot RIOT) MachineInfo() string {
	return riot.Timer.MachineInfo()
}

// map String to MachineInfo
func (riot RIOT) String() string {
	return riot.MachineInfo()
}

// ReadRIOTMemory checks for side effects to the RIOT sub-system
func (riot *RIOT) ReadRIOTMemory() {
	service, register, value := riot.mem.ChipRead()
	if !service {
		return
	}

	if riot.Timer.readMemory(register, value) {
		return
	}
}

// Step moves the state of the riot forward one video cycle
func (riot *RIOT) Step() {
	riot.Timer.step()
}
