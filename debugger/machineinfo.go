package debugger

import "fmt"

// types that satisfy machineInfo return information about the state of the
// emulated machine. String() should return verbose info, while StringTerse()
// the more terse equivalent.

type machineInfo interface {
	MachineInfo() string
	MachineInfoTerse() string
}

func (dbg Debugger) printMachineInfo(mi machineInfo) {
	dbg.print(MachineInfo, "%s", dbg.sprintMachineInfo(mi))
}

func (dbg Debugger) sprintMachineInfo(mi machineInfo) string {
	if dbg.machineInfoVerbose {
		return fmt.Sprintf("%s", mi.MachineInfo())
	}
	return fmt.Sprintf("%s", mi.MachineInfoTerse())
}
