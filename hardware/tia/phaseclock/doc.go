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

// Package phaseclock defines the two phase clock generator used to drive the
// various polynomial counters in the TIA.
//
// From the TIA_HW_Notes document:
//
//	Beside each counter there is a two-phase clock generator. This takes the
//	incoming 3.58 MHz colour clock (CLK) and divides by 4 using a couple of
//	flip-flops. Two AND gates are then used to generate two independent clock
//	signals thusly:
//	 ___         ___         ___
//	_| |_________| |_________| |_________  PHASE-1 (H@1)
//	       ___         ___         ___
//	_______| |_________| |_________| |___  PHASE-2 (H@2)
//
// Even though the two phases are independent these types of clocks never
// overlap (the skew margin is always positive). This means that the
// implementation can be simplified to a simple count from 0 to 3.
//
// The phaseclock can be "ticked" along by incrementing the integer and making
// sure it doesn't exceed the possible values: The accepted pattern is:
//
//	p++
//	if p >= phaseclock.NumStates {
//		p = 0
//	}
//
// Resetting and aligning the phase clock can be done by simply assigning the
// correct value to the PhaseClock instance. Use ResetValue and AlignValue for
// most purposes.
package phaseclock
