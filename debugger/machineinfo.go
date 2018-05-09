package debugger

// types that satisfy machineInfo return information about the state of the
// emulated machine. String() should return verbose info, while StringTerse()
// the more terse equivalent.

type machineInfo interface {
	String() string
	StringTerse() string
}

func (dbg Debugger) printMachineInfo(mi machineInfo) {
	if dbg.verbose {
		dbg.print("%v", mi)
	} else {
		dbg.print("%s\n", mi.StringTerse())
	}
}
