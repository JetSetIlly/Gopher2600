// Package reflection monitors the emulated hardware for conditions that would
// otherwise not be visible. In particular it signals the MetaPixelRenderer
// when certain memory addresses have been written to. For example, the HMOVE
// register.
//
// In addition it monitors the state of WSYNC and signals the
// MetaPixelRenderer when the CPU is idle. This makes for quite a nice visual
// indication of "lost cycles" or potential extra cycles that could be regained
// with a bit of reorgnisation.
//
// There are lots of other things we could potentially do with the reflection
// idea but as it currently is, it is a little underdeveloped. In particular,
// it's rather slow but I'm not too worried about that because this is for
// debugging not actually playing games and such.
//
// I think the next thing this needs is a way of making the various monitors
// switchable at runtime. As it is, what's compiled is what we get. If we
// monitored every possible thing, the MetaPixelRenderer would get cluttered
// very quickly. It would be nice to be able to define groups (say, a player
// sprites group, a HMOVE group, etc.) and to turn them on and off according to
// our needs.
package reflection

import (
	"gopher2600/gui"
	"gopher2600/hardware"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/future"
)

// Monitor watches for writes to specific video related memory locations. when
// these locations are written to, a signal is sent to the metapixels.Renderer
// implementation. moreover, if the monitor detects that the effect of the
// memory write is delayed or sustained, then the signal is repeated as
// appropriate.
type Monitor struct {
	vcs      *hardware.VCS
	renderer gui.MetaPixelRenderer

	groupTIA      addressMonitor
	groupPlayer0  addressMonitor
	groupPlayer1  addressMonitor
	groupMissile0 addressMonitor
	groupMissile1 addressMonitor
	groupBall     addressMonitor
}

// NewMonitor is the preferred method of initialisation for the Monitor type
func NewMonitor(vcs *hardware.VCS, renderer gui.MetaPixelRenderer) *Monitor {
	mon := &Monitor{vcs: vcs, renderer: renderer}

	mon.groupTIA.addresses = overlaySignals{
		0x03: gui.MetaPixel{Label: "RSYNC", Red: 255, Green: 10, Blue: 0, Alpha: 255, Scheduled: true},
		0x2a: gui.MetaPixel{Label: "HMOVE", Red: 255, Green: 20, Blue: 0, Alpha: 255, Scheduled: true},
		0x2b: gui.MetaPixel{Label: "HMCLR", Red: 255, Green: 30, Blue: 0, Alpha: 255, Scheduled: false},
	}

	mon.groupPlayer0.addresses = overlaySignals{
		0x04: gui.MetaPixel{Label: "NUSIZx", Red: 0, Green: 10, Blue: 255, Alpha: 255, Scheduled: true},
		0x10: gui.MetaPixel{Label: "RESPx", Red: 0, Green: 30, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupPlayer1.addresses = overlaySignals{
		0x05: gui.MetaPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: true},
		0x11: gui.MetaPixel{Label: "RESPx", Red: 0, Green: 70, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile0.addresses = overlaySignals{
		0x04: gui.MetaPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: false},
		0x11: gui.MetaPixel{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile1.addresses = overlaySignals{
		0x05: gui.MetaPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 0, Alpha: 255, Scheduled: false},
		0x12: gui.MetaPixel{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupBall.addresses = overlaySignals{
		0x14: gui.MetaPixel{Label: "RESBL", Red: 0, Green: 255, Blue: 10, Alpha: 255, Scheduled: true},
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
	sig := gui.MetaPixel{Label: "WSYNC", Red: 0, Green: 0, Blue: 0, Alpha: 200}
	return mon.renderer.SetMetaPixel(sig)
}

type overlaySignals map[uint16]gui.MetaPixel

type addressMonitor struct {
	// the map of memory addresses to monitor
	addresses overlaySignals

	// when memory has been written to we note the address and timestamp. then,
	// a few cycles later, we check to see if lastAddress is one the group is
	// interested in seeing
	lastAddress         uint16
	lastAddressAccessID int
	lastAddressFound    int

	// if the memory write resulted in an effect that won't occur until
	// sometime in the future then the Delay attribute for the part of the
	// system monitored by the group will yield a pointer to the future Event
	lastEvent *future.Event

	// a copy of the last signal sent to the overlay renderer. we use
	// this to repeat a signal when lastEvent is not nil and has not yet
	// completed
	signal gui.MetaPixel
}

func (adm *addressMonitor) check(rend gui.MetaPixelRenderer, mem *memory.VCSMemory, delay future.Observer) error {
	// if a new memory location (any memory location) has been written, then
	// note the new address and begin the delayed signalling process
	//
	// we filter on LastAccessTimeStamp rather than LastAccessAddress.
	// filtering by address will probably work in most instances but it won't
	// capture repeated writes to the same memory location.
	if mem.LastAccessWrite && mem.LastAccessID != adm.lastAddressAccessID {
		adm.lastAddress = mem.LastAccessAddress
		adm.lastAddressAccessID = mem.LastAccessID

		// 4 cycles seems plenty of time for an address to be serviced
		adm.lastAddressFound = 4
	}

	var signalStart bool
	var sig gui.MetaPixel

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
		err := rend.SetMetaPixel(adm.signal)
		if err != nil {
			return err
		}
	}

	return nil
}
