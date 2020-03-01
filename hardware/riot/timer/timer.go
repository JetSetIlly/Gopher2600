// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package timer

import (
	"fmt"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
)

// Interval indicates how often (in CPU cycles) the timer value decreases.
// the following rules apply
//		* set to 1, 8, 64 or 1024 depending on which address has been
//			written to by the CPU
//		* is used to reset the cyclesRemaining
//		* is changed to 1 once value reaches 0
//		* is reset to its initial value of 1, 8, 64, or 1024 whenever INTIM
//			is read by the CPU
type Interval int

// List of valid Interval values
const (
	TIM1T  Interval = 1
	TIM8T  Interval = 8
	TIM64T Interval = 64
	T1024T Interval = 1024
)

func (in Interval) String() string {
	switch in {
	case TIM1T:
		return "TIM1T"
	case TIM8T:
		return "TIM8T"
	case TIM64T:
		return "TIM64T"
	case T1024T:
		return "T1024T"
	}
	panic("unknown timer interval")
}

// Timer implements the timer part of the PIA 6532 (the T in RIOT)
type Timer struct {
	mem bus.ChipBus

	// the interval value most recently requested by the CPU
	Requested Interval

	// the current interval value (requested value can be superceded when timer
	// value reaches zero)
	Current Interval

	// INTIMvalue is the current timer value and is a reflection of the INTIM
	// RIOT memory register
	INTIMvalue uint8

	// CyclesRemaining is the number of CPU cycles remaining before the
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
	CyclesRemaining int
}

// NewTimer is the preferred method of initialisation of the Timer type
func NewTimer(mem bus.ChipBus) *Timer {
	tmr := &Timer{
		mem:             mem,
		Current:         T1024T,
		Requested:       T1024T,
		CyclesRemaining: int(T1024T),
		INTIMvalue:      0,
	}

	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.INTIMvalue))
	tmr.mem.ChipWrite(addresses.TIMINT, 0)

	return tmr
}

func (tmr Timer) String() string {
	return fmt.Sprintf("INTIM=%#02x remn=%#02x intv=%d (%s)",
		tmr.INTIMvalue,
		tmr.CyclesRemaining,
		tmr.Current,
		tmr.Requested,
	)
}

// ReadMemory checks to see if ChipData applies to the Timer type and
// updates the internal timer state accordingly. Returns true if the ChipData
// was *not* serviced.
func (tmr *Timer) ReadMemory(data bus.ChipData) bool {
	switch data.Name {
	case "TIM1T":
		tmr.Requested = TIM1T
	case "TIM8T":
		tmr.Requested = TIM8T
	case "TIM64T":
		tmr.Requested = TIM64T
	case "T1024T":
		tmr.Requested = T1024T
	default:
		return true
	}

	tmr.Current = tmr.Requested
	tmr.INTIMvalue = data.Value
	tmr.CyclesRemaining = 1

	// write value to INTIM straight-away
	tmr.mem.ChipWrite(addresses.INTIM, uint8(tmr.INTIMvalue))

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
		tmr.Current = tmr.Requested

		// reading the INTIM register always clears TIMINT
		tmr.mem.ChipWrite(addresses.TIMINT, 0x0)
	}

	tmr.CyclesRemaining--
	if tmr.CyclesRemaining <= 0 {
		if tmr.INTIMvalue == 0 {
			// set bit 7 of TIMINT register
			tmr.mem.ChipWrite(addresses.TIMINT, 0x80)

			// reset timer value
			tmr.INTIMvalue = 255

			// because timer value has cycled we flip timer interval to 1
			tmr.Current = 1
		} else {
			tmr.INTIMvalue--
		}

		// copy value to INTIM memory register
		tmr.mem.ChipWrite(addresses.INTIM, tmr.INTIMvalue)
		tmr.CyclesRemaining = int(tmr.Current)
	}
}
