package phaseclock

import "strings"

// phaseclock implements the two phase clock generator, as described in
// TIA_HW_Notes.txt:
//
// Beside each counter there is a two-phase clock generator. This
// takes the incoming 3.58 MHz colour clock (CLK) and divides by
// 4 using a couple of flip-flops. Two AND gates are then used to
// generate two independent clock signals thusly:
//  __          __          __
// _| |_________| |_________| |_________  PHASE-1 (H@1)
//        __          __          __
// _______| |_________| |_________| |___  PHASE-2 (H@2)

// PhaseClock is four-phase ticker. even though Phi1 and Phi2 are independent
// these types of clocks never overlap (the skew margin is always positive).
// this means that we can simply count from one to four to account for all
// possible outputs.
//
// note that the labels H@1 and H@2 are used in the TIA schematics for the
// HSYNC circuit. the phase clocks for the other polycounters are labelled
// differently, eg. P@1 and P@2 for the player sprites. to avoid confusion,
// we're using the labels Phi1 and Phi2, applicable to all polycounter
// phaseclocks.
type PhaseClock int

// valid PhaseClock values/states
const (
	risingPhi1 PhaseClock = iota
	fallingPhi1
	risingPhi2
	fallingPhi2
)

// NumStates is the number of phases the clock can be in
const NumStates = 4

// String creates a single line ASCII representation of the current state of
// the PhaseClock
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

// MachineInfoTerse returns the PhaseClock information in terse format
func (clk PhaseClock) MachineInfoTerse() string {
	return clk.String()
}

// MachineInfo returns the PhaseClock information in verbose format
func (clk PhaseClock) MachineInfo() string {
	s := strings.Builder{}
	switch clk {
	case risingPhi1:
		s.WriteString("_*--._______\n")
		s.WriteString("_______.--._\n")
	case fallingPhi1:
		s.WriteString("_.--*_______\n")
		s.WriteString("_______.--._\n")
	case risingPhi2:
		s.WriteString("_.--._______\n")
		s.WriteString("_______*--._\n")
	case fallingPhi2:
		s.WriteString("_.--._______\n")
		s.WriteString("_______.--*_\n")
	}
	return s.String()
}

// Align the phaseclock with the master clock by resetting to the rise of Phi1
func (clk *PhaseClock) Align() {
	*clk = risingPhi1
}

// Reset the phaseclock to the rise of Phi2
func (clk *PhaseClock) Reset() {
	*clk = risingPhi2
}

// Tick moves PhaseClock to next state
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

// Count returns the current clock state as an integer
func (clk PhaseClock) Count() int {
	return int(clk)
}

// Phi1 returns true if the Phi1 clock is on its rising edge
func (clk PhaseClock) Phi1() bool {
	return clk == risingPhi1
}

// Phi2 returns true if the Phi2 clock is on its rising edge
func (clk PhaseClock) Phi2() bool {
	return clk == risingPhi2
}

// LatePhi1 returns true if the Phi1 clock is on its falling edge
func (clk PhaseClock) LatePhi1() bool {
	return clk == fallingPhi1
}

// LatePhi2 returns true if the Phi2 clock is on its falling edge
func (clk PhaseClock) LatePhi2() bool {
	return clk == fallingPhi2
}
