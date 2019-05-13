package colorterm

import (
	"fmt"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/console"
)

// UserPrint is the top level output function
func (ct *ColorTerminal) UserPrint(sty console.Style, s string, a ...interface{}) {
	if sty != console.Input {
		ct.Print("\r")
	}

	switch sty {
	case console.CPUStep:
		ct.Print(ansi.PenColor["yellow"])
	case console.VideoStep:
		ct.Print(ansi.DimPens["yellow"])
	case console.MachineInfo:
		ct.Print(ansi.PenColor["cyan"])
	case console.EmulatorInfo:
		ct.Print(ansi.PenColor["blue"])
	case console.Error:
		ct.Print(ansi.PenColor["red"])
		ct.Print("* ")
	case console.Help:
		ct.Print(ansi.DimPens["white"])
		ct.Print("  ")
	case console.Feedback:
		ct.Print(ansi.DimPens["white"])
	case console.Prompt:
		ct.Print(ansi.PenStyles["bold"])
	}

	if len(a) > 0 {
		ct.Print(fmt.Sprintf(s, a...))
	} else {
		ct.Print(s)
	}
	ct.Print(ansi.NormalPen)

	// add a newline if print style is anything other than prompt
	if sty != console.Prompt && sty != console.Input {
		ct.Print("\n")
	}
}
