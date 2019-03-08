package monitor

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
)

// SystemState represents the current state of the emulation
type SystemState struct {
	Hmove bool
	Rsync bool
	Wsync bool
}

// SystemStateRecorder implementations will take note of the system state as
// preesented to them
type SystemStateRecorder interface {
	SystemStateRecord(SystemState) error
}

// SystemMonitor is a low level shim into the emulation
type SystemMonitor struct {
	Mem *memory.VCSMemory
	MC  *cpu.CPU
	Rec SystemStateRecorder

	lastMonitoredMemAddress uint16
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mon *SystemMonitor) Check() error {
	meta := SystemState{}
	meta.Wsync = !mon.MC.RdyFlg

	if mon.Mem.LastAddressAccessWrite {
		switch mon.Mem.LastAddressAccessed {
		case 0x03: // RSYNC
			if mon.lastMonitoredMemAddress != 0x03 {
				meta.Rsync = true
			}
		case 0x2a: // HMOVE
			if mon.lastMonitoredMemAddress != 0x2a {
				meta.Hmove = true
			}
		}
	}
	mon.lastMonitoredMemAddress = mon.Mem.LastAddressAccessed

	// send metasignal information
	err := mon.Rec.SystemStateRecord(meta)
	if err != nil {
		panic(err)
	}

	return nil
}
