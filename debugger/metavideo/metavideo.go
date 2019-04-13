package metavideo

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
)

// MetaSignalAttributes contains information about the last television signal. it is up to
// the Renderer to match this up with the last television signal
type MetaSignalAttributes struct {
	Label string

	// Renderer implementations are free to use the color information
	// as they wish (adding alpha information seems a probable scenario).
	Red, Green, Blue byte
}

// Renderer implementations will add signal information to a presentation layer
// somehow.
type Renderer interface {
	MetaSignal(MetaSignalAttributes) error
}

// Monitor watches for writes to specific video related memory locations. when
// these locations are written to, a MetaSignal is sent to the Renderer
// implementation.
type Monitor struct {
	Mem  *memory.VCSMemory
	MC   *cpu.CPU
	Rend Renderer

	// the emulation doesn't access memory every video cycle. we do check if it
	// every cycle however, so we need a way of filtering out false-positives
	// indicators that a memory address has been triggered.
	lastAddress uint16
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mon *Monitor) Check() error {
	var err error
	var sig MetaSignalAttributes

	// special handling of WSYNC signal - we want every pixel to be coloured
	// while the RdyFlag is false, not just when WSYNC is first triggered.
	if !mon.MC.RdyFlg {
		sig = MetaSignalAttributes{Label: "WSYNC", Red: 0, Green: 0, Blue: 255}
		err = mon.Rend.MetaSignal(sig)
		if err != nil {
			return err
		}
		return nil
	}

	if mon.Mem.LastAddressAccessWrite && mon.Mem.LastAddressAccessed != mon.lastAddress {
		sendSignal := true

		switch mon.Mem.LastAddressAccessed {
		case 0x03: // RSYNC
			sig = MetaSignalAttributes{Label: "RSYNC", Red: 255, Green: 0, Blue: 0}
		case 0x2a: // HMOVE
			sig = MetaSignalAttributes{Label: "HMOVE", Red: 0, Green: 255, Blue: 0}
		case 0x10:
			sig = MetaSignalAttributes{Label: "RESP0", Red: 0, Green: 255, Blue: 255}
		case 0x11:
			sig = MetaSignalAttributes{Label: "RESP1", Red: 0, Green: 255, Blue: 255}
		case 0x12:
			sig = MetaSignalAttributes{Label: "RESM0", Red: 0, Green: 255, Blue: 255}
		case 0x13:
			sig = MetaSignalAttributes{Label: "RESM1", Red: 0, Green: 255, Blue: 255}
		case 0x14:
			sig = MetaSignalAttributes{Label: "RESBL", Red: 0, Green: 255, Blue: 255}
		case 0x2b:
			sig = MetaSignalAttributes{Label: "HMCLR", Red: 255, Green: 0, Blue: 255}
		default:
			sendSignal = false
		}

		if sendSignal {
			err = mon.Rend.MetaSignal(sig)
			if err != nil {
				return err
			}
		}

		// note address
		mon.lastAddress = mon.Mem.LastAddressAccessed
	}

	return nil
}
