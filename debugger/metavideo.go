package debugger

import (
	"gopher2600/gui/metavideo"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
)

// metavideoMonitor watches for writes to specific video related memory locations. when
// these locations are written to, a MetaSignal is sent to the Renderer
// implementation.
type metavideoMonitor struct {
	Mem      *memory.VCSMemory
	MC       *cpu.CPU
	Renderer metavideo.Renderer

	// the emulation doesn't access memory every video cycle. we do check if it
	// every cycle however, so we need a way of filtering out false-positives
	// indicators that a memory address has been triggered.
	lastAddress uint16
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mv *metavideoMonitor) Check() error {
	var err error
	var sig metavideo.MetaSignalAttributes

	// special handling of WSYNC signal - we want every pixel to be coloured
	// while the RdyFlag is false, not just when WSYNC is first triggered.
	if !mv.MC.RdyFlg {
		sig = metavideo.MetaSignalAttributes{Label: "WSYNC", Red: 0, Green: 0, Blue: 255}
		err = mv.Renderer.MetaSignal(sig)
		if err != nil {
			return err
		}
		return nil
	}

	if mv.Mem.LastAccessWrite && mv.Mem.LastAccessAddress != mv.lastAddress {
		sendSignal := true

		switch mv.Mem.LastAccessAddress {
		case 0x03: // RSYNC
			sig = metavideo.MetaSignalAttributes{Label: "RSYNC", Red: 255, Green: 0, Blue: 0}
		case 0x2a: // HMOVE
			sig = metavideo.MetaSignalAttributes{Label: "HMOVE", Red: 0, Green: 255, Blue: 0}
		case 0x10:
			sig = metavideo.MetaSignalAttributes{Label: "RESP0", Red: 0, Green: 255, Blue: 255}
		case 0x11:
			sig = metavideo.MetaSignalAttributes{Label: "RESP1", Red: 0, Green: 255, Blue: 255}
		case 0x12:
			sig = metavideo.MetaSignalAttributes{Label: "RESM0", Red: 0, Green: 255, Blue: 255}
		case 0x13:
			sig = metavideo.MetaSignalAttributes{Label: "RESM1", Red: 0, Green: 255, Blue: 255}
		case 0x14:
			sig = metavideo.MetaSignalAttributes{Label: "RESBL", Red: 0, Green: 255, Blue: 255}
		case 0x2b:
			sig = metavideo.MetaSignalAttributes{Label: "HMCLR", Red: 255, Green: 0, Blue: 255}
		default:
			sendSignal = false
		}

		if sendSignal {
			err = mv.Renderer.MetaSignal(sig)
			if err != nil {
				return err
			}
		}

		// note address
		mv.lastAddress = mv.Mem.LastAccessAddress
	}

	return nil
}
