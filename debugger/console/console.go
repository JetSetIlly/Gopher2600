package console

import "gopher2600/debugger/input"

// UserInput defines the operations required by an interface that allows input
type UserInput interface {
	UserRead(buffer []byte, prompt string, interruptChannel chan func()) (int, error)
}

// UserOutput defines the operations required by an interface that allows
// output
type UserOutput interface {
	UserPrint(PrintProfile, string, ...interface{})
}

// UserInterface defines the user interface operations required by the debugger
type UserInterface interface {
	Initialise() error
	CleanUp()
	RegisterTabCompleter(*input.TabCompletion)
	UserInput
	UserOutput
}
