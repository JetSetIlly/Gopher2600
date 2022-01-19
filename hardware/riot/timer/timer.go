// Tjis file is part of Gopher2600.
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

	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
)

// Divider indicates how often (in CPU cycles) the timer value decreases.
// the following rules apply:
//		* set to 1, 8, 64 or 1024 depending on which address has been
//			written to by the CPU
//		* is used to reset the cyclesRemaining
//		* is changed to 1 once value reaches 0
//		* is reset to its initial value of 1, 8, 64, or 1024 whenever INTIM
//			is read by the CPU
type Divider int

// List of valid Divider values.
const (
	TIM1T  Divider = 1
	TIM8T  Divider = 8
	TIM64T Divider = 64
	T1024T Divider = 1024
)

func (in Divider) String() string {
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

	// return empty string for any unknown divider value
	return ""
}

// Timer implements the timer part of the PIA 6532 (the T in RIOT).
type Timer struct {
	mem chipbus.Memory

	instance *instance.Instance

	// the divider value most recently requested by the CPU
	divider Divider

	// the state of TIMINT. use timintValue() when writing to register
	expired bool
	pa7     bool

	// ticksRemaining is the number of CPU cycles remaining before the
	// value is decreased. the following rules apply:
	//		* set to 0 when new timer is set
	//		* causes value to decrease whenever it reaches -1
	//		* is reset to divider whenever value is decreased
	//
	// with regards to the last point, note that divider changes to 1
	// once INTIMvalue reaches 0
	ticksRemaining int
}

// NewTimer is the preferred method of initialisation of the Timer type.
func NewTimer(instance *instance.Instance, mem chipbus.Memory) *Timer {
	tmr := &Timer{
		instance: instance,
		mem:      mem,
		divider:  T1024T,
	}

	tmr.Reset()

	return tmr
}

// Reset timer to an initial state.
func (tmr *Timer) Reset() {
	tmr.pa7 = true

	if tmr.instance.Prefs.RandomState.Get().(bool) {
		tmr.divider = T1024T
		tmr.ticksRemaining = tmr.instance.Random.NoRewind(0xffff)
		tmr.mem.ChipWrite(chipbus.INTIM, uint8(tmr.instance.Random.NoRewind(0xff)))
	} else {
		tmr.divider = T1024T
		tmr.ticksRemaining = int(T1024T)
		tmr.mem.ChipWrite(chipbus.INTIM, 0)
	}

	tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
}

// Snapshot creates a copy of the RIOT Timer in its current state.
func (tmr *Timer) Snapshot() *Timer {
	n := *tmr
	return &n
}

// Plumb a new ChipBus into the Timer.
func (tmr *Timer) Plumb(mem chipbus.Memory) {
	tmr.mem = mem
}

func (tmr *Timer) String() string {
	return fmt.Sprintf("INTIM=%#02x remn=%#02x intv=%s TIMINT=%v",
		tmr.mem.ChipRefer(chipbus.INTIM),
		tmr.ticksRemaining,
		tmr.divider,
		tmr.expired,
	)
}

// MaskTIMINT defines the bits of TIMINT that are actually used.
const MaskTIMINT = 0b11000000

// the individual TIMINT bits and what they do
const (
	timintExpired = 0b10000000
	timintPA7     = 0b01000000
)

func (tmr *Timer) timintValue() uint8 {
	v := uint8(0)
	if tmr.expired {
		v |= timintExpired
	}
	if tmr.pa7 {
		v |= timintPA7
	}
	return v
}

// Update checks to see if ChipData applies to the Timer type and updates the
// internal timer state accordingly.
//
// Returns true if ChipData has *not* been serviced.
func (tmr *Timer) Update(data chipbus.ChangedRegister) bool {
	// change divider and return immediately if it hasn't been changed
	switch data.Register {
	case cpubus.TIM1T:
		tmr.divider = TIM1T
	case cpubus.TIM8T:
		tmr.divider = TIM8T
	case cpubus.TIM64T:
		tmr.divider = TIM64T
	case cpubus.T1024T:
		tmr.divider = T1024T
	default:
		return true
	}

	// writing to INTIM register has a similar effect on the expired bit of the
	// TIMINT register as reading. See commentary in the Step() function
	if tmr.ticksRemaining == 0 && tmr.mem.ChipRefer(chipbus.INTIM) == 0xff {
		tmr.expired = true
		tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
	} else {
		tmr.expired = false
		tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
	}

	// the ticks remaining value should be zero or one for accurate timing (as
	// tested with these test ROMs https://github.com/stella-emu/stella/issues/108).
	//
	// I'm not sure which value is correct so setting at zero until there's a
	// good reason to do otherwise
	//
	// note however, the internal values in the emulated machine (and as reported by
	// the debugger) will not match the debugging values in stella. to match
	// the debugging values in stella a value of 2 is required.
	tmr.ticksRemaining = 0

	// write value to INTIM straight-away
	tmr.mem.ChipWrite(chipbus.INTIM, data.Value)

	return false

}

// Step timer forward one cycle.
func (tmr *Timer) Step() {
	intim := tmr.mem.ChipRefer(chipbus.INTIM)

	if ok, a := tmr.mem.LastReadAddress(); ok {
		switch cpubus.Read[a] {
		case cpubus.INTIM:
			// if INTIM is *read* then the decrement reverts to once per timer
			// divider. this won't have any discernable effect unless the timer
			// divider has been flipped to 1 when INTIM cycles back to 255
			//
			// if the expired flag has *just* been set (ie. in the previous cycle)
			// then do not do the reversion. see discussion:
			//
			// https://atariage.com/forums/topic/303277-to-roll-or-not-to-roll/
			//
			// https://atariage.com/forums/topic/133686-please-explain-riot-timmers/?do=findComment&comment=1617207
			if tmr.ticksRemaining != 0 || intim != 0xff {
				tmr.expired = false
				tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
			}
		case cpubus.TIMINT:
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
	}

	tmr.ticksRemaining--
	if tmr.ticksRemaining <= 0 {
		intim--
		if intim == 0xff {
			tmr.expired = true
			tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
		}

		// copy value to INTIM memory register
		tmr.mem.ChipWrite(chipbus.INTIM, intim)

		if tmr.expired {
			tmr.ticksRemaining = 0
		} else {
			tmr.ticksRemaining = int(tmr.divider)
		}
	}
}

// PeekINTIM pokes a new value into the INTIM register. Same as peeking the
// INTIM register on the cpubus - provided here for convenience.
//
// Supported fields:
//  intim (uint8)
//  timint (uint8)
//  ticksRemainging (int)
//  divider (timer.Divider)
func (tmr *Timer) PeekField(fld string) interface{} {
	switch fld {
	case "intim":
		return tmr.mem.ChipRefer(chipbus.INTIM)
	case "timint":
		return tmr.mem.ChipRefer(chipbus.TIMINT)
	case "ticksRemaining":
		return tmr.ticksRemaining
	case "divider":
		return tmr.divider
	}
	panic(fmt.Sprintf("Timer.PeekField: unknown field: %s", fld))
}

// PokeINTIM pokes a new value into the INTIM register. Same as poking the
// INTIM register on the cpubus - provided here for convenience.
//
// Fields as described for PeekField().
func (tmr *Timer) PokeField(fld string, v interface{}) {
	switch fld {
	case "intim":
		tmr.mem.ChipWrite(chipbus.INTIM, v.(uint8))
	case "timint":
		tmr.expired = v.(uint8)&timintExpired == timintExpired
		tmr.pa7 = v.(uint8)&timintPA7 == timintPA7
		tmr.mem.ChipWrite(chipbus.TIMINT, tmr.timintValue())
	case "ticksRemaining":
		tmr.ticksRemaining = v.(int)
	case "divider":
		tmr.divider = v.(Divider)
	default:
		panic(fmt.Sprintf("Timer.PokeField: unknown field: %s", fld))
	}
}
