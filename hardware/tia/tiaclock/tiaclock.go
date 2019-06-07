package tiaclock

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
