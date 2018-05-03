package riot

import (
	"gopher2600/hardware/memory"
)

// RIOT contains all the sub-components of the VCS RIOT sub-system
type RIOT struct {
	mem memory.ChipBus
}

// NewRIOT is the preferred method of initialisation for the PIA structure
func NewRIOT(mem memory.ChipBus) *RIOT {
	riot := new(RIOT)
	if riot == nil {
		return nil
	}

	riot.mem = mem

	return riot
}

// ReadRIOTMemory checks for side effects to the RIOT sub-system
func (riot *RIOT) ReadRIOTMemory() {
	service, register, _ := riot.mem.ChipRead()
	if service {
		switch register {
		// TODO: implement timer
		// TODO: implement ports
		}
	}
}

// StepVideoCycle moves the state of the riot forward one video cycle
func (riot *RIOT) StepVideoCycle() {
}
