// Package colorterm implements the Terminal interface for the gopher2600
// debugger. It supports color output, history and tab completion.
package colorterm

import (
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/colorterm/easyterm"
	"os"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.EasyTerm

	reader         runeReader
	commandHistory []command
	tabCompletion  terminal.TabCompletion

	silenced bool
}

type command struct {
	input []byte
}

// Initialise perfoms any setting up required for the terminal
func (ct *ColorTerminal) Initialise() error {
	err := ct.EasyTerm.Initialise(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	ct.commandHistory = make([]command, 0)
	ct.reader = initRuneReader(os.Stdin)

	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (ct *ColorTerminal) CleanUp() {
	ct.EasyTerm.TermPrint("\r")
	_ = ct.Flush()
	ct.EasyTerm.CleanUp()
}

// RegisterTabCompletion adds an implementation of TabCompletion to the
// ColorTerminal
func (ct *ColorTerminal) RegisterTabCompletion(tc terminal.TabCompletion) {
	ct.tabCompletion = tc
}

// IsInteractive satisfies the terminal.UserInput interface
func (ct *ColorTerminal) IsInteractive() bool {
	return true
}

// Silence implements terminal.UserOutput interface
func (ct *ColorTerminal) Silence(silenced bool) {
	ct.silenced = silenced
}
