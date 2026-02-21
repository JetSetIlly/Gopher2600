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
	"errors"
	"fmt"
	"os"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
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
	//
	// Implementations that can't check ReadEvents will surely limit the
	// functionality of the debugger.
	TermRead(prompt Prompt, events *ReadEvents) (string, error)

	// TermReadCheck() returns true if there is input to be read. Not all
	// implementations will be able return anything meaningful in which case a
	// return value of false is fine.
	//
	// Note that TermReadCheck() does not check for events like TermRead().
	TermReadCheck() bool

	// IsRealTerminal returns true if the terminal implementation is using a
	// real terminal for Input.
	IsRealTerminal() bool
}

// sentinal errors controlling program exit
var (
	UserSignal    = errors.New("user signal")
	UserQuit      = fmt.Errorf("%w: quit", UserSignal)
	UserInterrupt = fmt.Errorf("%w: interrupt", UserSignal)
	UserReload    = fmt.Errorf("%w: reload", UserSignal)
)

// ReadEvents *must* be monitored during a TermRead().
type ReadEvents struct {
	// user input events. these are the inputs into the emulation
	// (ie. joystick, paddle, etc.)
	UserInput        chan userinput.Event
	UserInputHandler func(userinput.Event) error

	// signals from the operating system
	Signal        chan os.Signal
	SignalHandler func(os.Signal) error

	// PushedFunction allows functions to be pushed into the debugger goroutine.
	// errors are not returned by PushedFunction so errors should be logged
	PushedFunction chan func()

	// PushedFunctionImmediate is the same as PushedFunctions but handlers
	// must return control to the inputloop after the function has run
	PushedFunctionImmediate chan func()
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
	RegisterTabCompletion(*commandline.TabCompletion)

	// Silence all input and output except error messages. In other words,
	// TermPrintLine() should display error messages even if silenced is true.
	Silence(silenced bool)
}

// Broker implementations can identify a terminal.
type Broker interface {
	GetTerminal() Terminal
}
