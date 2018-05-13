package debugger

import (
	"strings"
)

// PrintProfile specifies the printing mode
type PrintProfile int

// enumeration of print profile types
const (
	StepResult PrintProfile = iota
	MachineInfo
	Error
	Feedback
	Prompt
)

// wrapper function for UserPrint() so we can normalise the input string before
// passing to UI interface
func (dbg Debugger) print(pp PrintProfile, s string, a ...interface{}) {

	// trim *all* trailing newlines - UserPrint() will add newlines if required
	s = strings.TrimRight(s, "\n")

	dbg.ui.UserPrint(pp, s, a...)
}
