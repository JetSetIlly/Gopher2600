package riot

import (
	"gopher2600/hardware/memory"
)

// RIOT contains all the sub-components of the VCS RIOT sub-system
type RIOT struct {
	mem memory.ChipBus

	timerRegister    string
	timerInterval    int
	timerINTIM       uint8
	timerINTIMClocks int
}

// New is the preferred method of initialisation for the PIA structure
func New(mem memory.ChipBus) *RIOT {
	riot := new(RIOT)
	if riot == nil {
		return nil
	}

	riot.mem = mem

	return riot
}

// ReadRIOTMemory checks for side effects to the RIOT sub-system
func (riot *RIOT) ReadRIOTMemory() {
	service, register, value := riot.mem.ChipRead()
	if service {
		switch register {
		case "TIM1T":
			riot.timerRegister = register
			riot.timerInterval = 1
			riot.timerINTIM = value
			riot.timerINTIMClocks = 1
		case "TIM8T":
			riot.timerRegister = register
			riot.timerInterval = 8
			riot.timerINTIM = value
			riot.timerINTIMClocks = 1
		case "TIM64T":
			riot.timerRegister = register
			riot.timerInterval = 64
			riot.timerINTIM = value
			riot.timerINTIMClocks = 1
		case "TIM1024":
			riot.timerRegister = register
			riot.timerInterval = 1024
			riot.timerINTIM = value
			riot.timerINTIMClocks = 1
		}
		// TODO: implement ports
	}
}

// StepVideoCycle moves the state of the riot forward one video cycle
func (riot *RIOT) StepVideoCycle() {
	// some documentation (Atari 2600 Specifications.htm) claims that if INTIM is
	// *read* then the decrement reverts to once per timer interval. this won't
	// have any effect unless the timer interval has been flipped to 1 when INTIM
	// cycles back to 255
	if riot.mem.ChipLastRegisterReadByCPU() == "INTIM" {
		switch riot.timerRegister {
		case "TIM1T":
			riot.timerInterval = 1
		case "TIM8T":
			riot.timerInterval = 8
		case "TIM64T":
			riot.timerInterval = 64
		case "TIM1024":
			riot.timerInterval = 1024
		}
	}

	if riot.timerRegister != "" {
		riot.timerINTIMClocks--
		if riot.timerINTIMClocks == 0 {
			if riot.timerINTIM == 0 {
				// reset INTIM value
				riot.timerINTIM = 255

				// because INTIM value has cycled we flip timer interval to 1
				riot.timerInterval = 1
			} else {
				riot.timerINTIM--
			}
			riot.mem.ChipWrite("INTIM", riot.timerINTIM)
			riot.timerINTIMClocks = riot.timerInterval
		}
	}
}
