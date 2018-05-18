package debugger

// types that satisfy machineInfo return information about the state of the
// emulated machine. String() should return verbose info, while StringTerse()
// the more terse equivalent.

type machineInfo interface {
	MachineInfo() string
	MachineInfoTerse() string
}

func (dbg Debugger) printMachineInfo(mi machineInfo) {
	if dbg.machineInfoVerbose {
		dbg.print(MachineInfo, "%s", mi.MachineInfo())
	} else {
		dbg.print(MachineInfo, "%s", mi.MachineInfoTerse())
	}
}
