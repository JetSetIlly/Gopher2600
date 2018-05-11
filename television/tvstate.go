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

// StringTerse returns the TVState in terse format
func (ts TVState) StringTerse() string {
	return ts.AsString(ts.value)
}

// String returns the TVState in verbose format
func (ts TVState) String() string {
	v := fmt.Sprintf(ts.valueFormat, ts.value)
	return fmt.Sprintf("%s=%s\n", ts.label, v)
}

// AsString returns the (terse) string representation of an aribtrary value
func (ts TVState) AsString(v interface{}) string {
	val := fmt.Sprintf(ts.valueFormat, v.(int))
	return fmt.Sprintf("%s=%s", ts.shortLabel, val)
}

// ToInt returns the value as an unsigned integer
func (ts TVState) ToInt() int {
	return ts.value
}
