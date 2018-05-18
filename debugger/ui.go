package debugger

import (
	"fmt"
	"os"
)

// UI defines the user interface operations required by the debugger
type UI interface {
	UserPrint(PrintProfile, string, ...interface{})
	UserRead([]byte) (int, error)
}

// PlainTerminal is the default, most basic terminal interface
type PlainTerminal struct {
	UI
}

// no newPlainUI() function currently required

// UserPrint is the plain terminal print routine
func (ui PlainTerminal) UserPrint(pp PrintProfile, s string, a ...interface{}) {
	switch pp {
	case Error:
		s = fmt.Sprintf("* %s", s)
	case Prompt:
		s = fmt.Sprintf("%s", s)
	case Script:
		s = fmt.Sprintf("> %s", s)
	}

	fmt.Printf(s, a...)

	if pp != Prompt {
		fmt.Println("")
	}
}

// UserRead is the plain terminal read routine
func (ui PlainTerminal) UserRead(input []byte) (int, error) {
	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}
