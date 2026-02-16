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

//go:build !windows

// Package colorterm implements the Terminal interface for the gopher2600
// debugger. It supports color output, history and tab completion.
package colorterm

import (
	"fmt"
	"os"

	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm/easyterm"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"golang.org/x/term"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.EasyTerm

	buffer         []byte
	reader         runeReader
	commandHistory []command
	tabCompletion  *commandline.TabCompletion

	silenced bool
}

type command struct {
	input []byte
}

// Initialise perfoms any setting up required for the terminal
func (ct *ColorTerminal) Initialise() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("colorterm: stdin is not a real terminal (use plain terminal)")
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("colorterm: stdout is not a real terminal (use plain terminal)")
	}

	err := ct.EasyTerm.Initialise(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	ct.buffer = make([]byte, 255)
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

func (ct *ColorTerminal) RegisterTabCompletion(tc *commandline.TabCompletion) {
	ct.tabCompletion = tc
}

// IsInteractive satisfies the terminal.Input interface
func (ct *ColorTerminal) IsInteractive() bool {
	return true
}

// IsRealTerminal implements the terminal.Input interface
func (ct *ColorTerminal) IsRealTerminal() bool {
	return true
}

// Silence implements terminal.Terminal interface
func (ct *ColorTerminal) Silence(silenced bool) {
	ct.silenced = silenced
}
