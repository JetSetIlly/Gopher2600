package ui

import (
	"fmt"
	"gopher2600/debugger"
	"os"
)

// ColorTerminal implements debugger UI interface
type ColorTerminal struct {
	prevProfile debugger.PrintProfile
}

// no NewColorTerminal() function currently

// UserPrint implementation for debugger.UI interface
func (ct *ColorTerminal) UserPrint(pp debugger.PrintProfile, s string, a ...interface{}) {
	switch pp {
	case debugger.StepResult:
		fmt.Print(ansiYellow)
	case debugger.MachineInfo:
		fmt.Print(ansiCyan)
	case debugger.Error:
		fmt.Print(ansiRed)
		s = fmt.Sprintf("* %s", s)
	case debugger.Feedback:
		fmt.Print(ansiGray)
	case debugger.Prompt:
	}

	if pp != debugger.Prompt {
		s = fmt.Sprintf("%s\n", s)
	}

	fmt.Printf(s, a...)
	fmt.Print(ansiOff)

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

const ansiOff = "\033[0m"

const ansiRed = "\033[31m"
const ansiGreen = "\033[32m"
const ansiYellow = "\033[33m"
const ansiBlue = "\033[34m"
const ansiMagenta = "\033[35m"
const ansiCyan = "\033[36m"
const ansiGray = "\033[37m"
