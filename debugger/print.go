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
		fmt.Sprintf("* %s", s)
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

// wrapper function for UserPrint so we can normalise the input string and not
// have to worry about strings too much in the main body of the emulator.  for
// example, when passing strings back from String() methods and whatnot,
// there's a danger we'll accumulate too many trailing newlines. with the
// wrapper function we can normalise trailing newlines before passing on to
// UserPrint
func (dbg Debugger) print(pp PrintProfile, s string, a ...interface{}) {
	if s[len(s)-1] == '\n' {
		s = fmt.Sprintf("%s\n", strings.TrimRight(s, "\n"))
	}
	dbg.UserPrint(pp, s, a...)
}
