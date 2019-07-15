package colorterm

import (
	"fmt"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/console"
)

// UserPrint is the top level output function
func (ct *ColorTerminal) UserPrint(sty console.Style, s string, a ...interface{}) {
	if sty != console.StyleInput {
		ct.Print("\r")
	}

	switch sty {
	case console.StyleCPUStep:
		ct.Print(ansi.PenColor["yellow"])
	case console.StyleVideoStep:
		ct.Print(ansi.DimPens["yellow"])
	case console.StyleMachineInfo:
		ct.Print(ansi.PenColor["cyan"])
	case console.StyleEmulatorInfo:
		ct.Print(ansi.PenColor["blue"])
	case console.StyleError:
		ct.Print(ansi.PenColor["red"])
		ct.Print("* ")
	case console.StyleHelp:
		ct.Print(ansi.DimPens["white"])
		ct.Print("  ")
	case console.StyleFeedback:
		ct.Print(ansi.DimPens["white"])
	case console.StylePrompt:
		ct.Print(ansi.PenStyles["bold"])
	case console.StylePromptAlt:
		// nothing special
	}

	if len(a) > 0 {
		ct.Print(fmt.Sprintf(s, a...))
	} else {
		ct.Print(s)
	}
	ct.Print(ansi.NormalPen)

	// add a newline if print style is anything other than prompt
	if sty != console.StylePrompt && sty != console.StylePromptAlt && sty != console.StyleInput {
		ct.Print("\n")
	}
}
