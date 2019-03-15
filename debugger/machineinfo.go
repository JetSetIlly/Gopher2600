package debugger

import "gopher2600/debugger/console"

// types that satisfy machineInfo return information about the state of the
// emulated machine. String() should return verbose info, while StringTerse()
// the more terse equivalent.

type machineInfo interface {
	MachineInfo() string
	MachineInfoTerse() string
}

func (dbg *Debugger) printMachineInfo(mi machineInfo) {
	dbg.print(console.MachineInfo, "%s", dbg.getMachineInfo(mi))
}

func (dbg *Debugger) getMachineInfo(mi machineInfo) string {
	if dbg.machineInfoVerbose {
		return mi.MachineInfo()
	}
	return mi.MachineInfoTerse()
}
