package debugger

import (
	"gopher2600/debugger/console"
)

// types that satisfy instrumentation return information about the state of the
// emulated machine
type instrumentation interface {
	String() string
}

func (dbg *Debugger) printInstrument(mi instrumentation) {
	dbg.print(console.StyleInstrument, "%s", mi.String())
}
