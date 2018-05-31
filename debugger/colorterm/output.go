package colorterm

import (
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/ui"
)

// UserPrint is the top level output function
func (ct *ColorTerminal) UserPrint(pp ui.PrintProfile, s string, a ...interface{}) {
	if pp != ui.Input {
		ct.Print("\r")
	}

	switch pp {
	case ui.CPUStep:
		ct.Print(ansi.PenColor["yellow"])
	case ui.VideoStep:
		ct.Print(ansi.DimPens["yellow"])
	case ui.MachineInfo:
		ct.Print(ansi.PenColor["cyan"])
	case ui.Error:
		ct.Print(ansi.PenColor["red"])
		ct.Print(ansi.PenColor["bold"])
		ct.Print("* ")
		ct.Print(ansi.NormalPen)
		ct.Print(ansi.PenColor["red"])
	case ui.Feedback:
		ct.Print(ansi.DimPens["white"])
	case ui.Script:
		ct.Print("> ")
	case ui.Prompt:
		ct.Print(ansi.PenStyles["bold"])
	}

	ct.Print(s, a...)
	ct.Print(ansi.NormalPen)

	// add a newline if print profile is anything other than prompt
	if pp != ui.Prompt && pp != ui.Input {
		ct.Print("\n")
	}
}
