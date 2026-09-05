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

// Package plainterm implements the Terminal interface for the gopher2600
// debugger. It's a simple as simple can be and offers no special features.
package plainterm

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"golang.org/x/term"
)

// PlainTerminal is the default, most basic terminal interface. It keeps the
// terminal in whatever mode it started, probably cooked mode. As such, it
// offers only rudimentary editing facility and little control over output.
type PlainTerminal struct {
	input      *bufio.Reader
	output     *bufio.Writer
	realInput  bool
	realOutput bool
	silenced   bool
}

// Initialise perfoms any setting up required for the terminal.
func (pt *PlainTerminal) Initialise() error {
	pt.input = bufio.NewReader(os.Stdin)
	pt.output = bufio.NewWriter(os.Stdout)
	pt.realInput = term.IsTerminal(int(os.Stdin.Fd()))
	pt.realOutput = term.IsTerminal(int(os.Stdout.Fd()))
	return nil
}

// CleanUp perfoms any cleaning up required for the terminal.
func (pt *PlainTerminal) CleanUp() {
	pt.output.Flush()
}

// RegisterTabCompletion adds an implementation of TabCompletion to the terminal.
func (pt *PlainTerminal) RegisterTabCompletion(*commandline.TabCompletion) {
}

// Silence implements the terminal.Terminal interface.
func (pt *PlainTerminal) Silence(silenced bool) {
	pt.silenced = silenced
}

// TermPrintLine implements the terminal.Output interface.
func (pt PlainTerminal) TermPrintLine(style terminal.Style, s string) {
	if pt.silenced && style != terminal.StyleError {
		return
	}

	// we don't need to echo user input for this type of terminal
	if style == terminal.StyleEcho {
		return
	}

	switch style {
	case terminal.StyleError:
		s = fmt.Sprintf("* %s", s)
	}

	pt.output.WriteString(s)
	pt.output.WriteString("\n")
	pt.output.Flush()
}

// TermRead implements the terminal.Input interface.
func (pt PlainTerminal) TermRead(prompt terminal.Prompt, events *terminal.ReadEvents) (string, error) {
	if pt.silenced {
		return "", nil
	}

	// insert prompt into output stream
	if pt.realInput {
		pt.output.Write([]byte(prompt.String()))
	}

	s, err := pt.input.ReadString('\n')
	if err != nil {
		// from the ReadString docs: "ReadString returns err != nil if and only if the returned data
		// does not end in delim". we never want to process partially read commands so we always
		// return on error regardless of the error type or the amount of data read
		return "", err
	}

	// while we were waiting for the call to Read() to return we may have
	// received an interrupt event. if we have then return a UserInterrupt
	// error to the debugging loop
	//
	// other events do not need to be checked - they will be serviced by the
	// debugger inputer loop elsewhere
	select {
	case sig := <-events.Signal:
		return "", events.SignalHandler(sig)
	default:
	}

	return s, nil
}

// TermReadCheck implements the terminal.Input interface.
func (pt *PlainTerminal) TermReadCheck() bool {
	return false
}

// IsInteractive implements the terminal.Input interface.
func (pt *PlainTerminal) IsInteractive() bool {
	return pt.realInput
}

// IsRealTerminal implements the terminal.Input interface.
func (pt *PlainTerminal) IsRealTerminal() bool {
	return pt.realInput && pt.realOutput
}
