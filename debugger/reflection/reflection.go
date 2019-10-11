package reflection

// the whole reflection system is slow. probably to do with indexing maps every
// video cycle. but I'm not too worried about it at the moment because it only
// ever runs in the debugger and the debugger is slow anyway (when compared to
// the playmode loop)
//
// it's also a bit of a hack. I didn't want to invade the emulation code too
// much but if we want to get fancier with this idea then we may have to. but
// in that case emulation performance should remain the priority.

import (
	"gopher2600/gui/overlay"
	"gopher2600/hardware"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/future"
	"time"
)

// Monitor watches for writes to specific video related memory locations. when
// these locations are written to, a signal is sent to the overlay.Renderer
// implementation. moreover, if the monitor detects that the effect of the
// memory write is delayed or sustained, then the signal is repeated as
// appropriate.
type Monitor struct {
	vcs      *hardware.VCS
	renderer overlay.Renderer

	groupTIA      addressMonitor
	groupPlayer0  addressMonitor
	groupPlayer1  addressMonitor
	groupMissile0 addressMonitor
	groupMissile1 addressMonitor
	groupBall     addressMonitor
}

// NewMonitor is the preferred method of initialisation for the Monitor type
func NewMonitor(vcs *hardware.VCS, renderer overlay.Renderer) *Monitor {
	mon := &Monitor{vcs: vcs, renderer: renderer}

	mon.groupTIA.addresses = overlaySignals{
		0x03: overlay.Signal{Label: "RSYNC", Red: 255, Green: 10, Blue: 0, Alpha: 255, Scheduled: true},
		0x2a: overlay.Signal{Label: "HMOVE", Red: 255, Green: 20, Blue: 0, Alpha: 255, Scheduled: true},
		0x2b: overlay.Signal{Label: "HMCLR", Red: 255, Green: 30, Blue: 0, Alpha: 255, Scheduled: false},
	}

	mon.groupPlayer0.addresses = overlaySignals{
		0x04: overlay.Signal{Label: "NUSIZx", Red: 0, Green: 10, Blue: 255, Alpha: 255, Scheduled: true},
		0x10: overlay.Signal{Label: "RESPx", Red: 0, Green: 30, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupPlayer1.addresses = overlaySignals{
		0x05: overlay.Signal{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: true},
		0x11: overlay.Signal{Label: "RESPx", Red: 0, Green: 70, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile0.addresses = overlaySignals{
		0x04: overlay.Signal{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: false},
		0x11: overlay.Signal{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile1.addresses = overlaySignals{
		0x05: overlay.Signal{Label: "NUSIZx", Red: 0, Green: 50, Blue: 0, Alpha: 255, Scheduled: false},
		0x12: overlay.Signal{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupBall.addresses = overlaySignals{
		0x14: overlay.Signal{Label: "RESBL", Red: 0, Green: 255, Blue: 10, Alpha: 255, Scheduled: true},
	}

	return mon
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mon *Monitor) Check() error {
	if err := mon.checkWSYNC(); err != nil {
		return err
	}

	if err := mon.groupTIA.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Delay); err != nil {
		return err
	}

	if err := mon.groupPlayer0.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Video.Player0.Delay); err != nil {
		return err
	}

	if err := mon.groupPlayer1.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Video.Player1.Delay); err != nil {
		return err
	}

	if err := mon.groupMissile0.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Video.Missile0.Delay); err != nil {
		return err
	}

	if err := mon.groupMissile1.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Video.Missile1.Delay); err != nil {
		return err
	}

	if err := mon.groupBall.check(mon.renderer, mon.vcs.Mem, mon.vcs.TIA.Video.Ball.Delay); err != nil {
		return err
	}

	return nil
}

func (mon *Monitor) checkWSYNC() error {
	if mon.vcs.CPU.RdyFlg {
		return nil
	}

	// special handling of WSYNC signal - we want every pixel to be coloured
	// while the RdyFlag is false, not just when WSYNC is first triggered.
	sig := overlay.Signal{Label: "WSYNC", Red: 0, Green: 0, Blue: 0, Alpha: 200}
	return mon.renderer.OverlaySignal(sig)
}

type overlaySignals map[uint16]overlay.Signal

type addressMonitor struct {
	// the map of memory addresses to monitor
	addresses overlaySignals

	// -----------------

	// when memory has been written to we note the address and timestamp. then,
	// a few cycles later, we check to see if lastAddress is one the group is
	// interested in seeing
	lastAddress          uint16
	lastAddressTimestamp time.Time
	lastAddressFound     int

	// if the memory write resulted in an effect that won't occur until
	// sometime in the future then the Delay attribute for the part of the
	// system monitored by the group will yield a pointer to the future Event
	lastEvent *future.Event

	// a copy of the last signal sent to the overlay renderer. we use
	// this to repeat a signal when lastEvent is not nil and has not yet
	// completed
	signal overlay.Signal
}

func (adm *addressMonitor) check(rend overlay.Renderer, mem *memory.VCSMemory, delay future.Observer) error {
	// if a new memory location (any memory location) has been written, then
	// note the new address and begin the delayed signalling process
	//
	// we filter on LastAccessTimeStamp rather than LastAccessAddress.
	// filtering by address will probably work in most instances but it won't
	// capture repeated writes to the same memory location.
	if mem.LastAccessWrite && mem.LastAccessTimeStamp != adm.lastAddressTimestamp {
		adm.lastAddress = mem.LastAccessAddress
		adm.lastAddressTimestamp = mem.LastAccessTimeStamp

		// 4 cycles seems plenty of time for an address to be serviced
		adm.lastAddressFound = 4
	}

	var signalStart bool
	var sig overlay.Signal

	if adm.lastAddressFound > 0 {
		if sig, signalStart = adm.addresses[adm.lastAddress]; signalStart {
			if sig.Scheduled {
				// associate memory write with delay observation
				if ev, ok := delay.Observe(sig.Label); ok {
					adm.lastEvent = ev
					adm.signal = sig
					adm.lastAddressFound = 1 // reduced to 0 almost immediately
				}
			} else {
				adm.lastEvent = nil
				adm.signal = sig
				adm.lastAddressFound = 1 // reduced to 0 almost immediately
			}
		}
		adm.lastAddressFound--
	}

	// send signal if an event is still running or if this is the end of a
	// writeDelay period. the second condition catches memory writes that do
	// not have an associated future.Event
	if adm.lastEvent != nil || signalStart {
		adm.lastEvent = nil
		err := rend.OverlaySignal(adm.signal)
		if err != nil {
			return err
		}
	}

	return nil
}
