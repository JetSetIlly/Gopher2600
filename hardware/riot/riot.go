package riot

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"math/rand"
)

// RIOT contains all the sub-components of the VCS RIOT sub-system
type RIOT struct {
	mem memory.ChipBus

	// timerRegister is the name of the currently selected RIOT timer. used as a
	// label in MachineInfo()
	timerRegister string

	// timerInterval indicates how often (in CPU cycles) the timer value
	// descreases. the following rules apply
	//		* set to 1, 8, 64 or 1024 depending on which address has been
	//			written to by the CPU
	//		* is used to reset the timerTickCyclesRemaining
	//		* is changed to 1 once timerValue reaches 0
	//		* is reset to its initial value of 1, 8, 64, or 1024 whenever INTIM
	//			is read by the CPU
	timerInterval int

	// timerValue is the current timer value and is a reflection of the INTIM
	// RIO memory register
	timerValue uint8

	// timerTickCyclesRemaining is the number of CPU cycles remaining before
	// the timerValue is decreased. the following rules apply:
	//		* set to 1 when new timer is set
	//		* causes timerValue to decrease whenever it reaches 0
	//		* is reset to timerInterval whenever timerValue is decreased
	//
	// with regards to the last point, note that timerInterval changes to 1
	// once timerValue reaches 0 (see timerInterval commentary above)
	//
	// the initial reset value is 1 because the first decrease of INTIM occurs
	// immediately after ReadRIOTMemory(); we want the timer cycle to hit 0 at
	// that time
	timerTickCyclesRemaining int
}

// NewRIOT creates a RIOT, to be used in a VCS emulation
func NewRIOT(mem memory.ChipBus) *RIOT {
	riot := new(RIOT)
	riot.mem = mem

	riot.timerRegister = "TIM1024"
	riot.timerInterval = 1024
	riot.timerTickCyclesRemaining = 1024
	riot.timerValue = uint8(rand.Intn(255))

	riot.mem.ChipWrite(addresses.INTIM, uint8(riot.timerValue))
	riot.mem.ChipWrite(addresses.TIMINT, 0)

	return riot
}

// MachineInfoTerse returns the RIOT information in terse format
func (riot RIOT) MachineInfoTerse() string {
	return fmt.Sprintf("INTIM=%#02x clks=%#04x (%s)", riot.timerValue, riot.timerTickCyclesRemaining, riot.timerRegister)
}

// MachineInfo returns the RIOT information in verbose format
func (riot RIOT) MachineInfo() string {
	return fmt.Sprintf("%s\nINTIM: %d (%#02x)\nINTIM clocks = %d (%#02x)", riot.timerRegister, riot.timerValue, riot.timerValue, riot.timerTickCyclesRemaining, riot.timerTickCyclesRemaining)
}

// map String to MachineInfo
func (riot RIOT) String() string {
	return riot.MachineInfo()
}

// ReadRIOTMemory checks for side effects to the RIOT sub-system
func (riot *RIOT) ReadRIOTMemory() {
	service, register, value := riot.mem.ChipRead()
	if service {
		switch register {
		case "TIM1T":
			riot.timerRegister = register
			riot.timerInterval = 1
			riot.timerTickCyclesRemaining = 1
			riot.timerValue = value
		case "TIM8T":
			riot.timerRegister = register
			riot.timerInterval = 8
			riot.timerTickCyclesRemaining = 1
			riot.timerValue = value
		case "TIM64T":
			riot.timerRegister = register
			riot.timerInterval = 64
			riot.timerTickCyclesRemaining = 1
			riot.timerValue = value
		case "TIM1024":
			riot.timerRegister = register
			riot.timerInterval = 1024
			riot.timerTickCyclesRemaining = 1
			riot.timerValue = value

			// TODO: handle other RIOT registers
		}

		// write value to INTIM straight-away
		riot.mem.ChipWrite(addresses.INTIM, uint8(riot.timerValue))

		// clear bit 7 of TIMINT register
		riot.mem.ChipWrite(addresses.TIMINT, 0x0)
	}
}

// Step moves the state of the riot forward one video cycle
func (riot *RIOT) Step() {
	// some documentation (Atari 2600 Specifications.htm) claims that if INTIM is
	// *read* then the decrement reverts to once per timer interval. this won't
	// have any discernable effect unless the timer interval has been flipped to
	// 1 when INTIM cycles back to 255
	if riot.mem.LastReadRegister() == "INTIM" {
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

		// reading the INTIM register always clears TIMINT
		riot.mem.ChipWrite(addresses.TIMINT, 0x0)
	}

	riot.timerTickCyclesRemaining--
	if riot.timerTickCyclesRemaining <= 0 {
		if riot.timerValue == 0 {
			// set bit 7 of TIMINT register
			riot.mem.ChipWrite(addresses.TIMINT, 0x80)

			// reset timer value
			riot.timerValue = 255

			// because timer value has cycled we flip timer interval to 1
			riot.timerInterval = 1
		} else {
			riot.timerValue--
		}

		// copy timerValue to INTIM memory register
		riot.mem.ChipWrite(addresses.INTIM, riot.timerValue)
		riot.timerTickCyclesRemaining = riot.timerInterval
	}
}
