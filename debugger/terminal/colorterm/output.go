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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package colorterm

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/colorterm/ansi"
)

// TermPrintLine implements the terminal.Terminal interface
func (ct *ColorTerminal) TermPrintLine(style terminal.Style, s string, a ...interface{}) {
	if ct.silenced && style != terminal.StyleError {
		return
	}

	if style != terminal.StyleInput {
		ct.EasyTerm.TermPrint("\r")
	}

	switch style {
	case terminal.StyleCPUStep:
		ct.EasyTerm.TermPrint(ansi.PenColor["yellow"])
	case terminal.StyleVideoStep:
		ct.EasyTerm.TermPrint(ansi.DimPens["yellow"])
	case terminal.StyleInstrument:
		ct.EasyTerm.TermPrint(ansi.PenColor["cyan"])
	case terminal.StyleEmulatorInfo:
		ct.EasyTerm.TermPrint(ansi.PenColor["blue"])
	case terminal.StyleError:
		ct.EasyTerm.TermPrint(ansi.PenColor["red"])
		ct.EasyTerm.TermPrint("* ")
	case terminal.StyleHelp:
		ct.EasyTerm.TermPrint(ansi.DimPens["white"])
	case terminal.StyleFeedback:
		ct.EasyTerm.TermPrint(ansi.DimPens["white"])
	case terminal.StylePromptCPUStep:
		ct.EasyTerm.TermPrint(ansi.PenStyles["bold"])
	case terminal.StylePromptVideoStep:
		// nothing special
	case terminal.StylePromptConfirm:
		ct.EasyTerm.TermPrint(ansi.PenColor["blue"])
	}

	if len(a) > 0 {
		ct.EasyTerm.TermPrint(fmt.Sprintf(s, a...))
	} else {
		ct.EasyTerm.TermPrint(s)
	}
	ct.EasyTerm.TermPrint(ansi.NormalPen)

	// add a newline if print style is anything other than prompt or input line
	if !style.IsPrompt() && style != terminal.StyleInput {
		ct.EasyTerm.TermPrint("\n")
	}
}
