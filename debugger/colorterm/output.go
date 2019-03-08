package colorterm

import (
	"fmt"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/ui"
)

// UserPrint is the top level output function
func (ct *ColorTerminal) UserPrint(profile ui.PrintProfile, s string, a ...interface{}) {
	if profile != ui.Input {
		ct.Print("\r")
	}

	switch profile {
	case ui.CPUStep:
		ct.Print(ansi.PenColor["yellow"])
	case ui.VideoStep:
		ct.Print(ansi.DimPens["yellow"])
	case ui.MachineInfo:
		ct.Print(ansi.PenColor["cyan"])
	case ui.MachineInfoInternal:
		ct.Print(ansi.PenColor["blue"])
	case ui.Error:
		ct.Print(ansi.PenColor["red"])
		ct.Print("* ")
	case ui.Help:
		ct.Print(ansi.DimPens["white"])
		ct.Print("  ")
	case ui.Feedback:
		ct.Print(ansi.DimPens["white"])
	case ui.Script:
		ct.Print("> ")
	case ui.Prompt:
		ct.Print(ansi.PenStyles["bold"])
	}

	if len(a) > 0 {
		ct.Print(fmt.Sprintf(s, a...))
	} else {
		ct.Print(s)
	}
	ct.Print(ansi.NormalPen)

	// add a newline if print profile is anything other than prompt
	if profile != ui.Prompt && profile != ui.Input {
		ct.Print("\n")
	}
}
