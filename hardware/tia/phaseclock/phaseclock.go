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
	RisingPhi1 PhaseClock = iota
	FallingPhi1
	RisingPhi2
	FallingPhi2
)

// ResetValue is used to reset the phaseclock.
const ResetValue = RisingPhi2

// ResetValue is used to align the phaseclock.
const AlignValue = RisingPhi1

// NumStates is the number of phases the clock can be in.
const NumStates = 4

// String creates a single line ASCII representation of the current state of
// the PhaseClock.
func (clk PhaseClock) String() string {
	s := strings.Builder{}
	switch clk {
	case RisingPhi1:
		s.WriteString("_*--.__.--._")
	case FallingPhi1:
		s.WriteString("_.--*__.--._")
	case RisingPhi2:
		s.WriteString("_.--.__*--._")
	case FallingPhi2:
		s.WriteString("_.--.__.--*_")
	}
	return s.String()
}
