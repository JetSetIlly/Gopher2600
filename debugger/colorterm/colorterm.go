package colorterm

import (
	"bufio"
	"gopher2600/debugger/colorterm/easyterm"
	"gopher2600/debugger/commands"
	"os"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.Terminal

	reader         *bufio.Reader
	commandHistory []command

	tabCompleter *commands.TabCompletion
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

	ct.reader = bufio.NewReader(os.Stdin)
	ct.commandHistory = make([]command, 0)

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
func (ct *ColorTerminal) RegisterTabCompleter(tc *commands.TabCompletion) {
	ct.tabCompleter = tc
}
