package television

import "fmt"

// TVstate is similar to the cpu/register type in that it implements the Target
// interface from the debugger pacakge. in other words the debugger can use a
// TVState instance to control the flow of the debugger.  unlike the
// cpu/register type though, TVState exists solely for this reason. it would be
// clearer for "tv states" to be straight-forward int types

// TVState is used to store information about the high-level tv state (eg.
// frame number, current scanline, etc.)
type TVState struct {
	label       string
	shortLabel  string
	value       int
	valueFormat string

	// invalid indicates that value is currently not valid
	// -- we've inverted the logic so that the default value of false is useful
	// for most cases, meaning that we don't have to think about this field at
	// all
	invalid bool
}

// Label returns the verbose label of the TVState
// (implements debugger.target)
func (ts TVState) Label() string {
	return ts.label
}

// ShortLabel returns the terse label of the TVState
// (implements debugger.target)
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

// map String to MachineInfo
func (ts TVState) String() string {
	return ts.MachineInfo()
}

// ToInt returns the value as an integer
// (implements debugger.target)
func (ts TVState) ToInt() int {
	return ts.value
}
