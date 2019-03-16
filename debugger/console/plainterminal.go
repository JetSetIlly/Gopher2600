package console

import (
	"fmt"
	"gopher2600/debugger/input"
	"os"
)

// PlainTerminal is the default, most basic terminal interface
type PlainTerminal struct {
}

// Initialise perfoms any setting up required for the terminal
func (pt *PlainTerminal) Initialise() error {
	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (pt *PlainTerminal) CleanUp() {
}

// RegisterTabCompleter adds an implementation of TabCompleter to the terminal
func (pt *PlainTerminal) RegisterTabCompleter(tc *input.TabCompletion) {
}

// UserPrint is the plain terminal print routine
func (pt PlainTerminal) UserPrint(pp PrintProfile, s string, a ...interface{}) {
	switch pp {
	case Error:
		s = fmt.Sprintf("* %s", s)
	case Script:
		s = fmt.Sprintf("> %s", s)
	case Help:
		s = fmt.Sprintf("  %s", s)
	}

	fmt.Printf(s, a...)

	if pp != Prompt {
		fmt.Println("")
	}
}

// UserRead is the plain terminal read routine
func (pt PlainTerminal) UserRead(input []byte, prompt string, interruptChannel chan func()) (int, error) {
	pt.UserPrint(Prompt, prompt)

	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}

// IsInteractive satisfies the console.UserInput interface
func (pt *PlainTerminal) IsInteractive() bool {
	return true
}
