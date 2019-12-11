// Package plainterm implements the Terminal interface for the gopher2600
// debugger. It's a simple as simple can be and offers no special features.
package plainterm

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/gui"
	"io"
	"os"
)

// PlainTerminal is the default, most basic terminal interface. It keeps the
// terminal in whatever mode it started, probably cooked mode. As such, it
// offers only rudimentary editing facility and little control over output.
type PlainTerminal struct {
	input    io.Reader
	output   io.Writer
	silenced bool
}

// Initialise perfoms any setting up required for the terminal
func (pt *PlainTerminal) Initialise() error {
	pt.input = os.Stdin
	pt.output = os.Stdout
	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (pt *PlainTerminal) CleanUp() {
}

// RegisterTabCompletion adds an implementation of TabCompletion to the terminal
func (pt *PlainTerminal) RegisterTabCompletion(terminal.TabCompletion) {
}

// TermPrint implements the terminal.Terminal interface
func (pt PlainTerminal) TermPrint(style terminal.Style, s string, a ...interface{}) {
	if pt.silenced && style != terminal.StyleError {
		return
	}

	switch style {
	case terminal.StyleError:
		s = fmt.Sprintf("* %s", s)
	case terminal.StyleHelp:
		s = fmt.Sprintf("  %s", s)
	}

	s = fmt.Sprintf(s, a...)
	pt.output.Write([]byte(s))

	if !style.IsPrompt() {
		pt.output.Write([]byte("\n"))
	}
}

// TermRead implements the terminal.Terminal interface
func (pt PlainTerminal) TermRead(input []byte, prompt terminal.Prompt, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	if pt.silenced {
		return 0, nil
	}

	pt.TermPrint(prompt.Style, prompt.Content)

	n, err := pt.input.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}

// IsInteractive implements the terminal.Input interface
func (pt *PlainTerminal) IsInteractive() bool {
	return true
}

// Silence implemented the terminal.Output interface
func (pt *PlainTerminal) Silence(silenced bool) {
	pt.silenced = silenced
}
