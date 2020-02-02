// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package reflection

import (
	"gopher2600/hardware"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/future"
)

// Monitor watches for writes to specific video related memory locations. when
// these locations are written to, a signal is sent to the Renderer
// implementation. moreover, if the monitor detects that the effect of the
// memory write is delayed or sustained, then the signal is repeated as
// appropriate.
type Monitor struct {
	vcs      *hardware.VCS
	renderer Renderer

	groupTIA      addressMonitor
	groupPlayer0  addressMonitor
	groupPlayer1  addressMonitor
	groupMissile0 addressMonitor
	groupMissile1 addressMonitor
	groupBall     addressMonitor
}

// NewMonitor is the preferred method of initialisation for the Monitor type
func NewMonitor(vcs *hardware.VCS, renderer Renderer) *Monitor {
	mon := &Monitor{vcs: vcs, renderer: renderer}

	mon.groupTIA.addresses = overlaySignals{
		0x03: ReflectPixel{Label: "RSYNC", Red: 255, Green: 10, Blue: 0, Alpha: 255, Scheduled: true},
		0x2a: ReflectPixel{Label: "HMOVE", Red: 255, Green: 20, Blue: 0, Alpha: 255, Scheduled: true},
		0x2b: ReflectPixel{Label: "HMCLR", Red: 255, Green: 30, Blue: 0, Alpha: 255, Scheduled: false},
	}

	mon.groupPlayer0.addresses = overlaySignals{
		0x04: ReflectPixel{Label: "NUSIZx", Red: 0, Green: 10, Blue: 255, Alpha: 255, Scheduled: true},
		0x10: ReflectPixel{Label: "RESPx", Red: 0, Green: 30, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupPlayer1.addresses = overlaySignals{
		0x05: ReflectPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: true},
		0x11: ReflectPixel{Label: "RESPx", Red: 0, Green: 70, Blue: 255, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile0.addresses = overlaySignals{
		0x04: ReflectPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255, Scheduled: false},
		0x11: ReflectPixel{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupMissile1.addresses = overlaySignals{
		0x05: ReflectPixel{Label: "NUSIZx", Red: 0, Green: 50, Blue: 0, Alpha: 255, Scheduled: false},
		0x12: ReflectPixel{Label: "RESMx", Red: 0, Green: 70, Blue: 0, Alpha: 255, Scheduled: true},
	}

	mon.groupBall.addresses = overlaySignals{
		0x14: ReflectPixel{Label: "RESBL", Red: 0, Green: 255, Blue: 10, Alpha: 255, Scheduled: true},
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
	sig := ReflectPixel{Label: "WSYNC", Red: 0, Green: 0, Blue: 0, Alpha: 200}
	return mon.renderer.SetReflectPixel(sig)
}

type overlaySignals map[uint16]ReflectPixel

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
	signal ReflectPixel
}

func (adm *addressMonitor) check(rend Renderer, mem *memory.VCSMemory, delay future.Observer) error {
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
	var sig ReflectPixel

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
		err := rend.SetReflectPixel(adm.signal)
		if err != nil {
			return err
		}
	}

	return nil
}
