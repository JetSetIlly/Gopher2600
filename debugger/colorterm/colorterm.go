package colorterm

import (
	"gopher2600/debugger/colorterm/easyterm"
	"gopher2600/debugger/console"
	"os"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.EasyTerm

	reader         runeReader
	commandHistory []command
	tabCompleter   console.TabCompleter
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
	ct.Print("\r")
	_ = ct.Flush()
	ct.EasyTerm.CleanUp()
}

// RegisterTabCompleter adds an implementation of TabCompleter to the
// ColorTerminal
func (ct *ColorTerminal) RegisterTabCompleter(tc console.TabCompleter) {
	ct.tabCompleter = tc
}

// IsInteractive satisfies the console.UserInput interface
func (ct *ColorTerminal) IsInteractive() bool {
	return true
}
