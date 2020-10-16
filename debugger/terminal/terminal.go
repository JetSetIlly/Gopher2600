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

	"github.com/jetsetilly/gopher2600/gui"
)

// Input defines the operations required by an interface that allows input.
type Input interface {
	// the TermRead loop should listenfor events on eventChannel and call
	// eventHandler with the received event as the argument.
	//
	// for example, where someChannel is private to the Input implementation
	//
	//	select {
	//	case <- someChannel
	//	case ev := <-eventChannel:
	//		return 0, eventHandler(ev)
	//	}
	TermRead(buffer []byte, prompt Prompt, events *ReadEvents) (int, error)

	// TermReadCheck() returns true if there is input to be read. not all
	// terminals will be able to implement this meaningfully. returning false
	// is fine.
	TermReadCheck() bool

	// IsInteractive() should return true for implementations that require user
	// interaction. implementations that don't require a user to interact with
	// the debugger should return false.
	IsInteractive() bool
}

// ReadEvents encapsulates the event channels that need to be monitored during
// a TermRead.
type ReadEvents struct {
	GuiEvents       chan gui.Event
	GuiEventHandler func(gui.Event) error
	IntEvents       chan os.Signal
	RawEvents       chan func()
}

// Output defines the operations required by an interface that allows output.
type Output interface {
	TermPrintLine(Style, string)
}

// Terminal defines the operations required by the debugger's command line interface.
type Terminal interface {
	// Userinterfaces, by definition, embed the Input and Output interfaces
	Input
	Output

	// initialise the terminal. not all terminal implementations will need to
	// do anything.
	Initialise() error

	// restore the terminal to it's original state, if possible. for example,
	// we could use this to make sure the terminal is returned to canonical
	// mode. not all terminal implementations will need to do anything.
	CleanUp()

	// register the tab completion engine to use with the UserInput
	// implementation
	RegisterTabCompletion(TabCompletion)

	// Silence all input and output (except error messages)
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

// Sentinal errors.
const (
	UserInterrupt = "user interrupt"
	UserAbort     = "user abort"
)
