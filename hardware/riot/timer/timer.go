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

package timer

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
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

// List of valid Interval values.
const (
	TIM1T  Interval = 1
	TIM8T  Interval = 8
	TIM64T Interval = 64
	T1024T Interval = 1024
)

// IntervalList is a list of all possible string representations of the Interval type.
var IntervalList = []string{"TIM1T", "TIM8T", "TIM64T", "T1024T"}

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

// Timer implements the timer part of the PIA 6532 (the T in RIOT).
type Timer struct {
	mem bus.ChipBus

	// the interval value most recently requested by the CPU
	Divider Interval

	// INTIMvalue is the current timer value and is a reflection of the INTIM
	// RIOT memory register. set with SetValue() function
	INTIMvalue uint8

	// the state of TIMINT. use timintValue() when writing to register
	expired bool
	pa7     bool

	// TicksRemaining is the number of CPU cycles remaining before the
	// value is decreased. the following rules apply:
	//		* set to 0 when new timer is set
	//		* causes value to decrease whenever it reaches -1
	//		* is reset to divider whenever value is decreased
	//
	// with regards to the last point, note that divider changes to 1
	// once INTIMvalue reaches 0
	TicksRemaining int
}

// NewTimer is the preferred method of initialisation of the Timer type.
func NewTimer(mem bus.ChipBus) *Timer {
	tmr := &Timer{
		mem:            mem,
		Divider:        T1024T,
		TicksRemaining: int(T1024T),
		INTIMvalue:     0,
		pa7:            true,
	}

	tmr.mem.ChipWrite(addresses.INTIM, tmr.INTIMvalue)
	tmr.mem.ChipWrite(addresses.TIMINT, tmr.timintValue())

	return tmr
}

// Snapshot creates a copy of the RIOT Timer in its current state.
func (tmr *Timer) Snapshot() *Timer {
	n := *tmr
	return &n
}

// Plumb a new ChipBus into the Timer.
func (tmr *Timer) Plumb(mem bus.ChipBus) {
	tmr.mem = mem
}

func (tmr Timer) String() string {
	return fmt.Sprintf("INTIM=%#02x remn=%#02x intv=%s TIMINT=%v",
		tmr.INTIMvalue,
		tmr.TicksRemaining,
		tmr.Divider,
		tmr.expired,
	)
}

func (tmr Timer) timintValue() uint8 {
	v := uint8(0)
	if tmr.expired {
		v |= 0x80
	}
	if tmr.pa7 {
		v |= 0x40
	}
	return v
}

// Update checks to see if ChipData applies to the Timer type and updates the
// internal timer state accordingly.
//
// Returns true if ChipData requires more attention.
func (tmr *Timer) Update(data bus.ChipData) bool {
	if tmr.SetInterval(data.Name) {
		return true
	}

	// writing to INTIM register has a similar effect on the expired bit of the
	// TIMINT register as reading. See commentary in the Step() function
	if tmr.TicksRemaining == 0 && tmr.INTIMvalue == 0xff {
		tmr.expired = true
		tmr.mem.ChipWrite(addresses.TIMINT, tmr.timintValue())
	} else {
		tmr.expired = false
		tmr.mem.ChipWrite(addresses.TIMINT, tmr.timintValue())
	}

	tmr.INTIMvalue = data.Value
	tmr.TicksRemaining = 0

	// write value to INTIM straight-away
	tmr.mem.ChipWrite(addresses.INTIM, tmr.INTIMvalue)

	return false
}

// Step timer forward one cycle.
func (tmr *Timer) Step() {
	switch tmr.mem.LastReadRegister() {
	case "INTIM":
		// if INTIM is *read* then the decrement reverts to once per timer
		// interval. this won't have any discernable effect unless the timer
		// interval has been flipped to 1 when INTIM cycles back to 255
		//
		// if the expired flag has *just* been set (ie. in the previous cycle)
		// then do not do the reversion. see discussion:
		//
		// https://atariage.com/forums/topic/303277-to-roll-or-not-to-roll/
		//
		// https://atariage.com/forums/topic/133686-please-explain-riot-timmers/?do=findComment&comment=1617207
		if tmr.TicksRemaining != 0 || tmr.INTIMvalue != 0xff {
			tmr.expired = false
			tmr.mem.ChipWrite(addresses.TIMINT, tmr.timintValue())
		}
	case "TIMINT":
		// from the NMOS 6532:
		//
		// "Clearing of the PA7 Interrupt Flag occurs when the microprocessor
		// reads the Interrupt Flag Register."
		//
		// and from the Rockwell 6532 documentation:
		//
		// "To clear PA7 interrupt flag, simply read the Interrupt Flag
		// Register"
		tmr.pa7 = false
	}

	tmr.TicksRemaining--
	if tmr.TicksRemaining < 0 {
		tmr.INTIMvalue--
		if tmr.INTIMvalue == 0xff {
			tmr.expired = true
			tmr.mem.ChipWrite(addresses.TIMINT, tmr.timintValue())
		}

		// copy value to INTIM memory register
		tmr.mem.ChipWrite(addresses.INTIM, tmr.INTIMvalue)

		if tmr.expired {
			tmr.TicksRemaining = 0
		} else {
			tmr.TicksRemaining = int(tmr.Divider) - 1
		}
	}
}

// SetValue sets the timer value. Prefer this to setting INTIMvalue directly.
func (tmr *Timer) SetValue(value uint8) {
	tmr.INTIMvalue = value
	tmr.mem.ChipWrite(addresses.INTIM, tmr.INTIMvalue)
}

// SetInterval sets the timer interval based on timer register name.
func (tmr *Timer) SetInterval(interval string) bool {
	switch interval {
	case "TIM1T":
		tmr.Divider = TIM1T
	case "TIM8T":
		tmr.Divider = TIM8T
	case "TIM64T":
		tmr.Divider = TIM64T
	case "T1024T":
		tmr.Divider = T1024T
	default:
		return true
	}

	return false
}

// SetTicks sets the number of remaining ticks.
func (tmr *Timer) SetTicks(ticks int) {
	tmr.TicksRemaining = ticks
}
