package debugger

import (
	"gopher2600/debugger/console"
)

// types that satisfy machineInformer return information about the state of the
// emulated machine. String() should return verbose info, while StringTerse()
// the more terse equivalent.
type machineInformer interface {
	MachineInfo() string
	MachineInfoTerse() string
}

func (dbg *Debugger) printMachineInfo(mi machineInformer) {
	dbg.print(console.StyleMachineInfo, "%s", dbg.getMachineInfo(mi))
}

func (dbg *Debugger) getMachineInfo(mi machineInformer) string {
	if dbg.machineInfoVerbose {
		return mi.MachineInfo()
	}
	return mi.MachineInfoTerse()
}
