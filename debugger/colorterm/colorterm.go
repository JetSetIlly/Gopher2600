package colorterm

import (
	"fmt"
	"gopher2600/debugger"
	"os"
)

// ColorTerminal implements debugger UI interface
type ColorTerminal struct {
	prevProfile debugger.PrintProfile
}

// no NewColorTerminal() function currently required

// UserPrint implementation for debugger.UI interface
func (ct *ColorTerminal) UserPrint(pp debugger.PrintProfile, s string, a ...interface{}) {
	switch pp {
	case debugger.StepResult:
		fmt.Print(pens["yellow"])
	case debugger.MachineInfo:
		fmt.Print(pens["cyan"])
	case debugger.Error:
		fmt.Print(pens["red"])
		fmt.Print(pens["bold"])
		fmt.Print("* ")
		fmt.Print(ansiOff)
		fmt.Print(pens["red"])
	case debugger.Feedback:
		fmt.Print(dimPens["white"])
	case debugger.Prompt:
	}

	fmt.Printf(s, a...)
	fmt.Print(ansiOff)

	// add a newline if print profile is anything other than prompt
	if pp != debugger.Prompt {
		fmt.Println("")
	}

	ct.prevProfile = pp
}

// UserRead implementation for debugger.UI interface
func (ct *ColorTerminal) UserRead(input []byte) (int, error) {
	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}
