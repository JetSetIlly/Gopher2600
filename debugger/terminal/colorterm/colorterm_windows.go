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

//go:build windows
// +build windows

// Package colorterm is not available under windows.
package colorterm

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal.
type ColorTerminal struct {
}

// Initialise perfoms any setting up required for the terminal.
func (ct *ColorTerminal) Initialise() error {
	return fmt.Errorf("color terminal not available on windows")
}

// CleanUp perfoms any cleaning up required for the terminal.
func (ct *ColorTerminal) CleanUp() {
}

// RegisterTabCompletion adds an implementation of TabCompletion to the
// ColorTerminal.
func (ct *ColorTerminal) RegisterTabCompletion(tc *commandline.TabCompletion) {
}

// IsInteractive satisfies the terminal.Input interface.
func (ct *ColorTerminal) IsInteractive() bool {
	return false
}

// IsRealTerminal implements the terminal.Input interface.
func (ct *ColorTerminal) IsRealTerminal() bool {
	return true
}

// Silence implements terminal.Terminal interface.
func (ct *ColorTerminal) Silence(silenced bool) {
}

// note that the followinf Term*() functions are not implemented for the
// unix version of ColorTerminal. this is because they are implemented by the
// embedded EasyTerm type.

// TermRead implements the terminal.Input interface.
func (ct *ColorTerminal) TermRead(input []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	return 0, nil
}

// TermReadCheck implements the terminal.Input interface.
func (ct *ColorTerminal) TermReadCheck() bool {
	return false
}

// TermPrintLine implements the terminal.Output interface.
func (ct *ColorTerminal) TermPrintLine(style terminal.Style, s string) {
}
