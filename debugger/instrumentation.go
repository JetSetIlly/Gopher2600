package debugger

import (
	"gopher2600/debugger/terminal"
)

// types that satisfy instrumentation return information about the state of the
// emulated machine
type instrumentation interface {
	String() string
}

func (dbg *Debugger) printInstrument(mi instrumentation) {
	dbg.printLine(terminal.StyleInstrument, "%s", mi.String())
}
