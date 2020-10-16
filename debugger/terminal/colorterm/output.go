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

// +build !windows

package colorterm

import (
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm/easyterm/ansi"
)

// TermPrintLine implements the terminal.Output interface.
func (ct *ColorTerminal) TermPrintLine(style terminal.Style, s string) {
	if ct.silenced && style != terminal.StyleError {
		return
	}

	// we don't need to echo user input for this type of terminal
	if style == terminal.StyleEcho {
		return
	}

	ct.EasyTerm.TermPrint("\r")

	switch style {
	case terminal.StyleHelp:
		ct.EasyTerm.TermPrint(ansi.DimPens["white"])

	case terminal.StyleFeedback:
		ct.EasyTerm.TermPrint(ansi.DimPens["white"])

	case terminal.StyleCPUStep:
		ct.EasyTerm.TermPrint(ansi.Pens["yellow"])

	case terminal.StyleVideoStep:
		ct.EasyTerm.TermPrint(ansi.DimPens["yellow"])

	case terminal.StyleInstrument:
		ct.EasyTerm.TermPrint(ansi.Pens["cyan"])

	case terminal.StyleError:
		ct.EasyTerm.TermPrint(ansi.Pens["red"])

	case terminal.StyleLog:
		ct.EasyTerm.TermPrint(ansi.Pens["magenta"])
	}

	ct.EasyTerm.TermPrint(s)
	ct.EasyTerm.TermPrint(ansi.NormalPen)

	ct.EasyTerm.TermPrint("\n")
}
