package debugger

// the whole metavideo system is slow. probably to do with indexing maps every
// video cycle. but I'm not too worried about it at the moment because it only
// ever runs in the debugger and the debugger is slow anyway (when compared to
// the playmode loop)
//
// it's also a bit of a hack. I didn't want to invade the emulation code too
// much but if we want to get fancier with this metavideo idea then we may have
// to. but in that case emulation performance should remain the priority.

import (
	"gopher2600/gui/metavideo"
	"gopher2600/hardware"
	"gopher2600/hardware/tia/delay/future"
	"time"
)

// metavideoMonitor watches for writes to specific video related memory locations. when
// these locations are written to, a MetaSignal is sent to the Renderer
// implementation. moreover, if the monitor detects that the effect of the
// memory write is delayed or sustained, then the signal is repeated as
// appropriate.
type metavideoMonitor struct {
	VCS      *hardware.VCS
	Renderer metavideo.Renderer

	groupTIA     metavideoGroup
	groupPlayer0 metavideoGroup
	groupPlayer1 metavideoGroup
}

func newMetavideoMonitor(vcs *hardware.VCS, renderer metavideo.Renderer) *metavideoMonitor {
	mon := &metavideoMonitor{VCS: vcs, Renderer: renderer}

	mon.groupTIA.addresses = metaSignals{
		0x03: metavideo.MetaSignalAttributes{Label: "RSYNC", Red: 255, Green: 10, Blue: 0, Alpha: 255},
		0x2a: metavideo.MetaSignalAttributes{Label: "HMOVE", Red: 255, Green: 20, Blue: 0, Alpha: 255},
		0x2b: metavideo.MetaSignalAttributes{Label: "HMCLR", Red: 255, Green: 30, Blue: 0, Alpha: 255},
	}

	mon.groupPlayer0.addresses = metaSignals{
		0x04: metavideo.MetaSignalAttributes{Label: "NUSIZx", Red: 0, Green: 10, Blue: 255, Alpha: 255},
		0x10: metavideo.MetaSignalAttributes{Label: "RESPx", Red: 0, Green: 30, Blue: 255, Alpha: 255},
	}

	mon.groupPlayer1.addresses = metaSignals{
		0x05: metavideo.MetaSignalAttributes{Label: "NUSIZx", Red: 0, Green: 50, Blue: 255, Alpha: 255},
		0x11: metavideo.MetaSignalAttributes{Label: "RESPx", Red: 0, Green: 70, Blue: 255, Alpha: 255},
	}

	return mon
}

type metaSignals map[uint16]metavideo.MetaSignalAttributes

type metavideoGroup struct {
	// the map of memory addresses to monitor
	addresses metaSignals

	// -----------------

	// when memory has been written to we note the address and timestamp. then,
	// a few cycles later, we check to see if lastAddress is one the group is
	// interested in seeing
	lastAddress          uint16
	lastAddressTimestamp time.Time

	// when the CPU has written to a memory location other parts of the VCS
	// will not necessarily see the new value until sometime later.
	writeDelay int

	// if the memory write resulted in an effect that won't occur until
	// sometime in the future then the Delay attribute for the part of the
	// system monitored by the group will yield a pointer to the future Event
	lastEvent *future.Event

	// a copy of the last metasignal sent to the metavideo renderer. we use
	// this to repeat a signal when lastEvent is not nil and has not yet
	// completed
	signal metavideo.MetaSignalAttributes
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mon *metavideoMonitor) Check() error {
	if err := mon.groupWSYNC(); err != nil {
		return err
	}

	if err := mon.checkGroup(&mon.groupTIA, mon.VCS.TIA.Delay); err != nil {
		return err
	}

	if err := mon.checkGroup(&mon.groupPlayer0, mon.VCS.TIA.Video.Player0.Delay); err != nil {
		return err
	}

	if err := mon.checkGroup(&mon.groupPlayer1, mon.VCS.TIA.Video.Player1.Delay); err != nil {
		return err
	}

	return nil
}

func (mon *metavideoMonitor) groupWSYNC() error {
	if mon.VCS.CPU.RdyFlg {
		return nil
	}

	// special handling of WSYNC signal - we want every pixel to be coloured
	// while the RdyFlag is false, not just when WSYNC is first triggered.
	sig := metavideo.MetaSignalAttributes{Label: "WSYNC", Red: 0, Green: 0, Blue: 0, Alpha: 200}
	return mon.Renderer.MetaSignal(sig)
}

func (mon *metavideoMonitor) checkGroup(group *metavideoGroup, delay future.Observer) error {
	// if a new memory location (any memory location) has been written, then
	// note the new address and begin the delayed metasignal process
	//
	// we filter on LastAccessTimeStamp rather than LastAccessAddress.
	// filtering by address will probably work in most instances but it won't
	// capture repeated writes to the same memory location. timestamp in that
	// sense, is unique
	if mon.VCS.Mem.LastAccessWrite && mon.VCS.Mem.LastAccessTimeStamp != group.lastAddressTimestamp {
		group.lastAddress = mon.VCS.Mem.LastAccessAddress
		group.lastAddressTimestamp = mon.VCS.Mem.LastAccessTimeStamp

		// when the CPU has written to a memory location the TIA will not see
		// the new value until sometime later. the delay makes sure the
		// metavideo subsystem sees it at the same time
		//
		// * this is affected by when the call to TIA.ReadMemory() is made
		group.writeDelay = 3
	}

	// when delay reaches 0 check to see if last address written is an address
	// being monitored by the group
	//
	// we wait until now to check if address is of interest because we don't
	// want to interfere (until as late as possible) any existing events that
	// may be in the middle of a MetaSignal sequence
	if group.writeDelay == 1 {
		if sig, ok := group.addresses[group.lastAddress]; ok {
			group.signal = sig

			// associate memory write with delay observation
			var ok bool
			if group.lastEvent, ok = delay.Observe(sig.Label); !ok {
				group.lastEvent = nil
			}
		} else {
			group.writeDelay = 0
		}

	}

	// send metasignal if an event is still running or if this is the end of a
	// writeDelay period. the second condition catches memory writes that do
	// not have an associated future.Event
	if (group.lastEvent != nil && !group.lastEvent.Completed()) || group.writeDelay == 1 {
		err := mon.Renderer.MetaSignal(group.signal)
		if err != nil {
			return err
		}
	}

	// count down write delay period. doing this after sending of metasignal so
	// that we don't clobber writeDelay too early
	if group.writeDelay >= 0 {
		group.writeDelay--
	}

	return nil
}
