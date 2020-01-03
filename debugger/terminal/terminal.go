package terminal

import (
	"gopher2600/gui"
)

// Prompt specifies the prompt text and the prompt style.
type Prompt struct {
	Content string
	Style   Style
}

// Input defines the operations required by an interface that allows input.
type Input interface {
	// the TermRead loop should listen (if possible) for events on eventChannel
	// and call eventHandler with the received event as the argument.
	TermRead(buffer []byte, prompt Prompt, eventChannel chan gui.Event, eventHandler func(gui.Event) error) (int, error)

	// IsInteractive() should return true for implementations that require user
	// interaction. implementations that don't require a user to interact with
	// the debugger should return false.
	IsInteractive() bool
}

// Output defines the operations required by an interface that allows output.
type Output interface {
	TermPrintLine(Style, string, ...interface{})
}

// Terminal defines the operations required by the debugger's command line interface.
type Terminal interface {
	Initialise() error
	CleanUp()

	// register the tab completion engine to use with the UserInput
	// implementation
	RegisterTabCompletion(TabCompletion)

	// Silence all input and output (except error messages)
	Silence(silenced bool)

	// Userinterfaces, by definition, embed the Input and Output interfaces
	Input
	Output
}

// TabCompletion defines the operations required for tab completion. A good
// implementation can be found in the commandline sub-package.
type TabCompletion interface {
	Complete(input string) string
	Reset()
}
