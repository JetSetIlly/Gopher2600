package console

import (
	"gopher2600/gui"
)

// Prompt represents the text that is to pose as a prompt to the user when user
// input is required
type Prompt struct {
	Content string
	Style   Style
}

// UserInput defines the operations required by an interface that allows input
type UserInput interface {
	UserRead(buffer []byte, prompt Prompt, eventChannel chan gui.Event, eventHandler func(gui.Event) error) (int, error)
	IsInteractive() bool
}

// UserOutput defines the operations required by an interface that allows
// output
type UserOutput interface {
	UserPrint(Style, string, ...interface{})
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
