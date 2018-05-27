package television

import "fmt"

// TVstate is similar to the cpu/register type in that it implements the
// BreakTarget interface in debugger pacakge. in other words the debugger can
// use a TVState instance to control the flow of the debugger.  unlike the
// cpu/register type though, TVState exists solely for this reason. it would
// be clearer for "tv states" to be straight-forward int types

// TVState is used to store information about the high-level tv state (eg.
// framg number, current scanline, etc.)
type TVState struct {
	label       string
	shortLabel  string
	value       int
	valueFormat string
}

// Label returns the verbose label of the TVState
func (ts TVState) Label() string {
	return ts.label
}

// ShortLabel returns the terse label of the TVState
func (ts TVState) ShortLabel() string {
	return ts.shortLabel
}

// MachineInfoTerse returns the TVState in terse format
func (ts TVState) MachineInfoTerse() string {
	s := fmt.Sprintf(ts.valueFormat, ts.value)
	return fmt.Sprintf("%s=%s", ts.shortLabel, s)
}

// MachineInfo returns the TVState in verbose format
func (ts TVState) MachineInfo() string {
	s := fmt.Sprintf(ts.valueFormat, ts.value)
	return fmt.Sprintf("%s=%s", ts.label, s)
}

// map String to MachineInfoTerse
func (ts TVState) String() string {
	return ts.MachineInfoTerse()
}

// ToInt returns the value as an integer
// (implements debugger.target)
func (ts TVState) ToInt() int {
	return ts.value
}
