package console

import (
	"gopher2600/gui"
)

// UserInput defines the operations required by an interface that allows input
type UserInput interface {
	UserRead(buffer []byte, prompt string, eventChannel chan gui.Event, eventHandler func(gui.Event) error) (int, error)
	IsInteractive() bool
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
	RegisterTabCompleter(TabCompleter)
	UserInput
	UserOutput
}

// TabCompleter defines the operations required for tab completion
type TabCompleter interface {
	Complete(input string) string
	Reset()
}
