package tiaclock

import "strings"

// tiaclock implements the two phase clock generator, as described in
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

// TIAClock is four-phase ticker
type TIAClock int

// valid TIAClock values
const (
	risingH1 TIAClock = iota
	fallingH1
	risingH2
	fallingH2
)

// NumStates is the number of phases the clock can be in
const NumStates = 4

// String creates a two line ASCII representation of the current state of
// the TIAClock
func (clk TIAClock) String() string {
	s := strings.Builder{}
	s.WriteString("   __    __   \n")
	switch clk {
	case risingH1:
		s.WriteString("__*  |__|  |__\n")
	case fallingH1:
		s.WriteString("__|  *__|  |__\n")
	case risingH2:
		s.WriteString("__|  |__*  |__\n")
	case fallingH2:
		s.WriteString("__|  |__|  *__\n")
	}
	return s.String()
}

// MachineInfoTerse returns the TIAClock information in terse format
func (clk TIAClock) MachineInfoTerse() string {
	return clk.String()
}

// MachineInfo returns the TIAClock information in verbose format
func (clk TIAClock) MachineInfo() string {
	return clk.String()
}

// Tick moves TIAClock to next state
func (clk *TIAClock) Tick() {
	switch *clk {
	case risingH1:
		*clk = fallingH1
	case fallingH1:
		*clk = risingH2
	case risingH2:
		*clk = fallingH2
	case fallingH2:
		*clk = risingH1
	}
}

// IsLatched returns true if phase clock is at the
func (clk TIAClock) IsLatched() bool {
	return clk == risingH2
}

// Reset puts the clock into a known initial state
func (clk *TIAClock) Reset() {
	*clk = risingH2
}
