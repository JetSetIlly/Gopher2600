package timer

import (
	"fmt"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
)

// Timer implements the timer part of the PIA 6532 (the T in RIOT)
type Timer struct {
	mem bus.ChipBus

	// register is the name of the currently selected RIOT timer
	register string

	// interval indicates how often (in CPU cycles) the timer value decreases.
	// the following rules apply
	//		* set to 1, 8, 64 or 1024 depending on which address has been
	//			written to by the CPU
	//		* is used to reset the cyclesRemaining
	//		* is changed to 1 once value reaches 0
	//		* is reset to its initial value of 1, 8, 64, or 1024 whenever INTIM
	//			is read by the CPU
	interval int

	// value is the current timer value and is a reflection of the INTIM
	// RIO memory register
	value uint8

	// cyclesRemaining is the number of CPU cycles remaining before the
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
	cyclesRemaining int

	// the number of CPU cyles taken place since last write to a TIMxxx
	// register
	cyclesElapsed int
}

// NewTimer is the preferred method of initialisation of the Timer type
func NewTimer(mem bus.ChipBus) *Timer {
	tmr := &Timer{
		mem:             mem,
		register:        "TIM1024",
		interval:        1024,
		cyclesRemaining: 1024,
		cyclesElapsed:   0,
		value:           0,
	}

	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.value))
	tmr.mem.ChipWrite(addresses.TIMINT, 0)

	return tmr
}

func (tmr Timer) String() string {
	return fmt.Sprintf("INTIM=%#02x elpsd=%02d remn=%#04x intv=%d (%s)",
		tmr.value,
		tmr.cyclesElapsed,
		tmr.cyclesRemaining,
		tmr.interval,
		tmr.register,
	)
}

// ServiceMemory checks to see if ChipData applies to the Timer type and
// updates the internal timer state accordingly. Returns true if the ChipData
// was *not* serviced.
func (tmr *Timer) ServiceMemory(data bus.ChipData) bool {
	switch data.Name {
	case "TIM1T":
		tmr.register = data.Name
		tmr.interval = 1
		tmr.cyclesRemaining = 1
		tmr.value = data.Value
	case "TIM8T":
		tmr.register = data.Name
		tmr.interval = 8
		tmr.cyclesRemaining = 1
		tmr.value = data.Value
	case "TIM64T":
		tmr.register = data.Name
		tmr.interval = 64
		tmr.cyclesRemaining = 1
		tmr.value = data.Value
	case "TIM1024":
		tmr.register = data.Name
		tmr.interval = 1024
		tmr.cyclesRemaining = 1
		tmr.value = data.Value

	default:
		return true
	}

	tmr.cyclesElapsed = 0

	// write value to INTIM straight-away
	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.value))

	// clear bit 7 of TIMINT register
	tmr.mem.ChipWrite(addresses.TIMINT, 0x0)

	return false
}

// Step timer forward one cycle
func (tmr *Timer) Step() {
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

	tmr.cyclesRemaining--
	if tmr.cyclesRemaining <= 0 {
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
		tmr.cyclesRemaining = tmr.interval
	}

	tmr.cyclesElapsed++
}
