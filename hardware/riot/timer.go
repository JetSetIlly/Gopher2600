package riot

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
)

type timer struct {
	mem memory.ChipBus

	// register is the name of the currently selected RIOT timer. used as a
	// label in MachineInfo()
	register string

	// interval indicates how often (in CPU cycles) the timer value decreases.
	// the following rules apply
	//		* set to 1, 8, 64 or 1024 depending on which address has been
	//			written to by the CPU
	//		* is used to reset the tickCyclesRemaining
	//		* is changed to 1 once value reaches 0
	//		* is reset to its initial value of 1, 8, 64, or 1024 whenever INTIM
	//			is read by the CPU
	interval int

	// value is the current timer value and is a reflection of the INTIM
	// RIO memory register
	value uint8

	// tickCyclesRemaining is the number of CPU cycles remaining before the
	// value is decreased. the following rules apply:
	//		* set to 1 when new timer is set
	//		* causes value to decrease whenever it reaches 0
	//		* is reset to interval whenever value is decreased
	//
	// with regards to the last point, note that interval changes to 1
	// once value reaches 0 (see interval commentary above)
	//
	// the initial reset value is 1 because the first decrease of INTIM occurs
	// immediately after ReadRIOTMemory(); we want the timer cycle to hit 0 at
	// that time
	tickCyclesRemaining int
}

func newTimer(mem memory.ChipBus) *timer {
	tmr := new(timer)
	tmr.mem = mem

	tmr.register = "TIM1024"
	tmr.interval = 1024
	tmr.tickCyclesRemaining = 1024
	tmr.value = 0

	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.value))
	tmr.mem.ChipWrite(addresses.TIMINT, 0)

	return tmr
}

// MachineInfoTerse returns the RIOT information in terse format
func (tmr timer) MachineInfoTerse() string {
	return fmt.Sprintf("INTIM=%#02x clks=%#04x (%s)", tmr.value, tmr.tickCyclesRemaining, tmr.register)
}

// MachineInfo returns the RIOT information in verbose format
func (tmr timer) MachineInfo() string {
	return fmt.Sprintf("%s\nINTIM: %d (%#02x)\nClocks Rem: %d (%#03x)", tmr.register, tmr.value, tmr.value, tmr.tickCyclesRemaining, tmr.tickCyclesRemaining)
}

func (tmr *timer) readMemory(register string, value uint8) bool {
	switch register {
	case "TIM1T":
		tmr.register = register
		tmr.interval = 1
		tmr.tickCyclesRemaining = 1
		tmr.value = value
	case "TIM8T":
		tmr.register = register
		tmr.interval = 8
		tmr.tickCyclesRemaining = 1
		tmr.value = value
	case "TIM64T":
		tmr.register = register
		tmr.interval = 64
		tmr.tickCyclesRemaining = 1
		tmr.value = value
	case "TIM1024":
		tmr.register = register
		tmr.interval = 1024
		tmr.tickCyclesRemaining = 1
		tmr.value = value

	default:
		return false
	}

	// write value to INTIM straight-away
	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.value))

	// clear bit 7 of TIMINT register
	tmr.mem.ChipWrite(addresses.TIMINT, 0x0)

	return true
}

func (tmr *timer) step() {
	// some documentation (Atari 2600 Specifications.htm) claims that if INTIM is
	// *read* then the decrement reverts to once per timer interval. this won't
	// have any discernable effect unless the timer interval has been flipped to
	// 1 when INTIM cycles back to 255
	if tmr.mem.LastReadRegister() == "INTIM" {
		switch tmr.register {
		case "TIM1T":
			tmr.interval = 1
		case "TIM8T":
			tmr.interval = 8
		case "TIM64T":
			tmr.interval = 64
		case "TIM1024":
			tmr.interval = 1024
		}

		// reading the INTIM register always clears TIMINT
		tmr.mem.ChipWrite(addresses.TIMINT, 0x0)
	}

	tmr.tickCyclesRemaining--
	if tmr.tickCyclesRemaining <= 0 {
		if tmr.value == 0 {
			// set bit 7 of TIMINT register
			tmr.mem.ChipWrite(addresses.TIMINT, 0x80)

			// reset timer value
			tmr.value = 255

			// because timer value has cycled we flip timer interval to 1
			tmr.interval = 1
		} else {
			tmr.value--
		}

		// copy value to INTIM memory register
		tmr.mem.ChipWrite(addresses.INTIM, tmr.value)
		tmr.tickCyclesRemaining = tmr.interval
	}
}
