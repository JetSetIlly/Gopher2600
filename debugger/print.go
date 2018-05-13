package debugger

import (
	"fmt"
	"os"
	"strings"
)

// this file of the debugger package contains all the code relating to UI
// printing

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

// minimalist print routine -- default assignment to UserPrint
func plainPrint(pp PrintProfile, s string, a ...interface{}) {
	if pp == Error {
		s = fmt.Sprintf("* %s", s)
	}

	if pp != Prompt {
		s = fmt.Sprintf("%s\n", s)
	}

	fmt.Printf(s, a...)
}

func plainRead(input []byte) (int, error) {
	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}

// wrapper function for UserPrint so we can normalise the input string before
// passing to the UserPrint() function
func (dbg Debugger) print(pp PrintProfile, s string, a ...interface{}) {

	// trim *all* trailing newlines - UserPrint() will add newlines if required
	s = strings.TrimRight(s, "\n")

	dbg.UserPrint(pp, s, a...)
}
