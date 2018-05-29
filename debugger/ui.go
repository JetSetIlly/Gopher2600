package debugger

import (
	"fmt"
	"os"
)

// UserInterface defines the user interface operations required by the debugger
type UserInterface interface {
	Initialise() error
	CleanUp()
	UserPrint(PrintProfile, string, ...interface{})
	UserRead([]byte, string) (int, error)
}

// UserInterrupt can be returned by UserRead() when user has cause an
// interrupt (ie. CTRL-C)
type UserInterrupt struct {
	Message string
}

// implement Error interface for UserInterrupt
func (intr UserInterrupt) Error() string {
	return intr.Message
}

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

// UserPrint is the plain terminal print routine
func (pt PlainTerminal) UserPrint(pp PrintProfile, s string, a ...interface{}) {
	switch pp {
	case Error:
		s = fmt.Sprintf("* %s", s)
	case Script:
		s = fmt.Sprintf("> %s", s)
	}

	fmt.Printf(s, a...)

	if pp != Prompt {
		fmt.Println("")
	}
}

// UserRead is the plain terminal read routine
func (pt PlainTerminal) UserRead(input []byte, prompt string) (int, error) {
	pt.UserPrint(Prompt, prompt)

	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}
