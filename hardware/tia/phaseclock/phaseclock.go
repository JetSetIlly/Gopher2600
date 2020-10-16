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
package phaseclock

import "strings"

// The four-phase clock can be represent as an integer.
type PhaseClock int

// Valid PhaseClock values/states
//
// Note that the labels H@1 and H@2 are used in the TIA schematics for the
// HSYNC circuit. the phase clocks for the other polycounters are labelled
// differently, eg. P@1 and P@2 for the player sprites. to avoid confusion,
// we're using the labels Phi1 and Phi2, applicable to all polycounter
// phaseclocks.
const (
	risingPhi1 PhaseClock = iota
	fallingPhi1
	risingPhi2
	fallingPhi2
)

// NumStates is the number of phases the clock can be in.
const NumStates = 4

// String creates a single line ASCII representation of the current state of
// the PhaseClock.
func (clk PhaseClock) String() string {
	s := strings.Builder{}
	switch clk {
	case risingPhi1:
		s.WriteString("_*--.__.--._")
	case fallingPhi1:
		s.WriteString("_.--*__.--._")
	case risingPhi2:
		s.WriteString("_.--.__*--._")
	case fallingPhi2:
		s.WriteString("_.--.__.--*_")
	}
	return s.String()
}

// Sync two clocks to the same phase.
func (clk *PhaseClock) Sync(oclk PhaseClock) {
	switch oclk {
	case risingPhi1:
		*clk = fallingPhi2
	case fallingPhi1:
		*clk = risingPhi1
	case risingPhi2:
		*clk = fallingPhi1
	case fallingPhi2:
		*clk = risingPhi2
	}
}

// Align the phaseclock with the master clock by resetting to the rise of Phi1.
func (clk *PhaseClock) Align() {
	*clk = risingPhi1
}

// Reset the phaseclock to the rise of Phi2.
func (clk *PhaseClock) Reset() {
	*clk = risingPhi2
}

// Tick moves PhaseClock to next state.
func (clk *PhaseClock) Tick() {
	switch *clk {
	case risingPhi1:
		*clk = fallingPhi1
	case fallingPhi1:
		*clk = risingPhi2
	case risingPhi2:
		*clk = fallingPhi2
	case fallingPhi2:
		*clk = risingPhi1
	}
}

// Count returns the current clock state as an integer.
func (clk PhaseClock) Count() int {
	return int(clk)
}

// Phi1 returns true if the Phi1 clock is on its rising edge.
func (clk PhaseClock) Phi1() bool {
	return clk == risingPhi1
}

// Phi2 returns true if the Phi2 clock is on its rising edge.
func (clk PhaseClock) Phi2() bool {
	return clk == risingPhi2
}

// LatePhi1 returns true if the Phi1 clock is on its falling edge.
func (clk PhaseClock) LatePhi1() bool {
	return clk == fallingPhi1
}

// LatePhi2 returns true if the Phi2 clock is on its falling edge.
func (clk PhaseClock) LatePhi2() bool {
	return clk == fallingPhi2
}
