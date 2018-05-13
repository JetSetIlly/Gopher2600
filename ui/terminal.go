package ui

import (
	"fmt"
	"gopher2600/debugger"
)

// SetupTerminal figures out the capabilities of the terminal and changes the
// UI callbacks in the debugger accordingly
func SetupTerminal(dbg *debugger.Debugger) {
	// TODO: test for color capabilities
	dbg.UserPrint = colorPrint

	// TODO: change UserRead callback if possible
}

// colorPrint routine -- alternative to default plainPrint routiner in debugger
// package
func colorPrint(pp debugger.PrintProfile, s string, a ...interface{}) {
	switch pp {
	case debugger.StepResult:
		fmt.Print(ansiYellow)
	case debugger.MachineInfo:
		fmt.Print(ansiCyan)
	case debugger.Error:
		fmt.Print(ansiRed)
		fmt.Sprintf("* %s", s)
	case debugger.Feedback:
		fmt.Print(ansiGray)
	case debugger.Prompt:
	}

	fmt.Printf(s, a...)
	fmt.Print(ansiOff)
}

const ansiOff = "\033[0m"

const ansiRed = "\033[31m"
const ansiGreen = "\033[32m"
const ansiYellow = "\033[33m"
const ansiBlue = "\033[34m"
const ansiMagenta = "\033[35m"
const ansiCyan = "\033[36m"
const ansiGray = "\033[37m"
