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

/* plain UI is the default, most basic terminal interface */

type plainUI struct {
	UI
}

// no newPlainUI() function currently required

// minimalist print routine -- default assignment to UserPrint
func (ui plainUI) UserPrint(pp PrintProfile, s string, a ...interface{}) {
	if pp == Error {
		s = fmt.Sprintf("* %s", s)
	}

	if pp != Prompt {
		s = fmt.Sprintf("%s\n", s)
	}

	fmt.Printf(s, a...)
}

func (ui plainUI) UserRead(input []byte) (int, error) {
	n, err := os.Stdin.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}
