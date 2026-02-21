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
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"golang.org/x/term"
)

// PlainTerminal is the default, most basic terminal interface. It keeps the
// terminal in whatever mode it started, probably cooked mode. As such, it
// offers only rudimentary editing facility and little control over output.
type PlainTerminal struct {
	input      io.Reader
	output     io.Writer
	realInput  bool
	realOutput bool
	silenced   bool
	buffer     []byte
}

// Initialise perfoms any setting up required for the terminal.
func (pt *PlainTerminal) Initialise() error {
	pt.input = os.Stdin
	pt.output = os.Stdout
	pt.realInput = term.IsTerminal(int(os.Stdin.Fd()))
	pt.realOutput = term.IsTerminal(int(os.Stdout.Fd()))
	pt.buffer = make([]byte, 255)
	return nil
}

// CleanUp perfoms any cleaning up required for the terminal.
func (pt *PlainTerminal) CleanUp() {
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

	pt.output.Write([]byte(s))
	pt.output.Write([]byte("\n"))
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

	n, err := pt.input.Read(pt.buffer)
	if err != nil {
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

	return string(pt.buffer[:n]), nil
}

// TermReadCheck implements the terminal.Input interface.
func (pt *PlainTerminal) TermReadCheck() bool {
	return false
}

// IsRealTerminal implements the terminal.Input interface.
func (pt *PlainTerminal) IsRealTerminal() bool {
	return pt.realInput && pt.realOutput
}
