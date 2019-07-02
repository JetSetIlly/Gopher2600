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

// PhaseClock is four-phase ticker
type PhaseClock int

// valid PhaseClock values/states. we are ordering the states differently to
// that suggested by the diagram above and the String() function below. this is
// because the clock starts at the beginning of Phase-2 and as such, it is more
// convenient to think of risingPhi2 as the first state, rather than
// risingPhi1.
const (
	risingPhi1 PhaseClock = iota
	fallingPhi1
	risingPhi2
	fallingPhi2
)

// NumStates is the number of phases the clock can be in
const NumStates = 4

// String creates a two line ASCII representation of the current state of
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
	return clk.String()
}

// Reset puts the clock into a known initial state
func (clk *PhaseClock) Reset(outOfPhase bool) {
	if outOfPhase {
		*clk = risingPhi1
	} else {
		*clk = risingPhi2
	}
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

// InPhase returns true if the clock is at the tick point that polycounters
// should be advanced
func (clk PhaseClock) InPhase() bool {
	return clk == risingPhi2
}

// OutOfPhase returns true if the clock suggests that events goverened by MOTCK
// should take place. from TIA_HW_Notes.txt:
//
// "The [MOTCK] (motion clock?) line supplies the CLK signals
// for all movable graphics objects during the visible part of
// the scanline. It is an inverted (out of phase) CLK signal."
func (clk PhaseClock) OutOfPhase() bool {
	return clk == risingPhi1
}
