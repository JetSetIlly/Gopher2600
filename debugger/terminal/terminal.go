// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package terminal

import (
	"os"

	"github.com/jetsetilly/gopher2600/userinput"
)

// Input defines the operations required by an interface that allows input.
type Input interface {
	// TermRead will return the number of characters inserted into the buffer,
	// or an error, when completed.
	//
	// If possible the TermRead() implementation should regularly check the
	// ReadEvents channels for activity. Not all implementations will be able
	// to do so because of the context in which they operate.
	TermRead(buffer []byte, prompt Prompt, events *ReadEvents) (int, error)

	// TermReadCheck() returns true if there is input to be read. Not all
	// implementations will be able return anything meaningful in which case a
	// return value of false is fine.
	TermReadCheck() bool

	// IsInteractive() should return true for implementations that require user
	// interaction. Instances that don't expect user intervention should return
	// false.
	IsInteractive() bool
}

// Sentinal errors. Returned by TermRead() if caught whilst waiting for input.
// Not all terminal implementations will return these errors because of the
// context in which they operate and in those instances signals should be
// cuaght by the IntEvents channel found in the ReadEvents type.
const (
	UserInterrupt = "user interrupt"
	UserAbort     = "user abort"
)

// ReadEvents *must* be monitored during a TermRead().
type ReadEvents struct {
	// gui events. must be handled immediately by accompanying GuiEventHandler
	UserInput        chan userinput.Event
	UserInputHandler func(userinput.Event) error

	// interrupt signals from the operating system
	IntEvents chan os.Signal

	// RawEvents allows functions to be pushed into the debugger goroutine
	RawEvents chan func()

	// RawEventsImm is a variation of RawEvents that returns control to the
	// input loop as soon as the function is run.
	//
	// Useful when pushed function have side-effects that must be serviced by
	// the input loop immediately.
	RawEventsImm chan func()

	// errors are not returned by RawEvents or RawEventsGUI so errors
	// accumulated inside a raw event function should be logged rather than
	// dropped.
}

// Output defines the operations required by an interface that allows output.
type Output interface {
	TermPrintLine(Style, string)
}

// Terminal defines the operations required by the debugger's command line interface.
type Terminal interface {
	// Terminal implementation also implement the Input and Output interfaces.
	Input
	Output

	// Initialise the terminal. not all terminal implementations will need to
	// do anything.
	Initialise() error

	// Restore the terminal to it's original state, if possible. for example,
	// we could use this to make sure the terminal is returned to canonical
	// mode. not all terminal implementations will need to do anything.
	CleanUp()

	// Register a tab completion implementation to use with the terminal. Not
	// all implementations need to respond meaningfully to this.
	RegisterTabCompletion(TabCompletion)

	// Silence all input and output except error messages. In other words,
	// TermPrintLine() should display error messages even if silenced is true.
	Silence(silenced bool)
}

// TabCompletion defines the operations required for tab completion. A good
// implementation can be found in the commandline sub-package.
type TabCompletion interface {
	Complete(input string) string
	Reset()
}

// Broker implementations can identify a terminal.
type Broker interface {
	GetTerminal() Terminal
}
