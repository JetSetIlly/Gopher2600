package colorterm

import (
	"gopher2600/debugger/colorterm/easyterm"
	"gopher2600/debugger/input"
	"os"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.Terminal

	reader         runeReader
	commandHistory []command
	tabCompleter   *input.TabCompletion
}

type command struct {
	input []byte
}

// Initialise perfoms any setting up required for the terminal
func (ct *ColorTerminal) Initialise() error {
	err := ct.Terminal.Initialise(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	ct.commandHistory = make([]command, 0)
	ct.reader = initRuneReader()

	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (ct *ColorTerminal) CleanUp() {
	ct.Print("\r")
	_ = ct.Flush()
	ct.Terminal.CleanUp()
}

// RegisterTabCompleter adds an implementation of TabCompleter to the
// ColorTerminal
func (ct *ColorTerminal) RegisterTabCompleter(tc *input.TabCompletion) {
	ct.tabCompleter = tc
}

// IsInteractive satisfies the console.UserInput interface
func (ct *ColorTerminal) IsInteractive() bool {
	return true
}
