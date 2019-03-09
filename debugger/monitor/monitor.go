package monitor

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
)

// SystemState represents the current state of the emulation
type SystemState struct {
	Label string
	Group string
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
	var err error
	meta := SystemState{}

	if !mon.MC.RdyFlg {
		meta.Label = "WSYNC"
		err = mon.Rec.SystemStateRecord(meta)
		if err != nil {
			panic(err)
		}
		return nil
	}

	if mon.Mem.LastAddressAccessWrite {
		switch mon.Mem.LastAddressAccessed {
		case 0x03: // RSYNC
			if mon.lastMonitoredMemAddress != 0x03 {
				meta.Label = "RSYNC"
			}
		case 0x2a: // HMOVE
			if mon.lastMonitoredMemAddress != 0x2a {
				meta.Label = "HMOVE"
			}
		case 0x10:
			if mon.lastMonitoredMemAddress != 0x10 {
				meta.Label = "RESP0"
				meta.Group = "sprite reset"
			}
		case 0x11:
			if mon.lastMonitoredMemAddress != 0x11 {
				meta.Label = "RESP1"
				meta.Group = "sprite reset"
			}
		case 0x12:
			if mon.lastMonitoredMemAddress != 0x12 {
				meta.Label = "RESM0"
				meta.Group = "sprite reset"
			}
		case 0x13:
			if mon.lastMonitoredMemAddress != 0x13 {
				meta.Label = "RESM1"
				meta.Group = "sprite reset"
			}
		case 0x14:
			if mon.lastMonitoredMemAddress != 0x14 {
				meta.Label = "RESBL"
				meta.Group = "sprite reset"
			}
		case 0x2b:
			if mon.lastMonitoredMemAddress != 0x2b {
				meta.Label = "HMCLR"
			}
		}

		if meta.Label != "" {
			err = mon.Rec.SystemStateRecord(meta)
			if err != nil {
				panic(err)
			}
		}
	}

	// note address
	mon.lastMonitoredMemAddress = mon.Mem.LastAddressAccessed

	return nil
}
