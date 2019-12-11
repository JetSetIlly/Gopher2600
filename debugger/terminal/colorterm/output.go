package colorterm

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/colorterm/ansi"
)

// TermPrint implements the terminal.Terminal interface
func (ct *ColorTerminal) TermPrint(style terminal.Style, s string, a ...interface{}) {
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
		ct.EasyTerm.TermPrint("  ")
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
