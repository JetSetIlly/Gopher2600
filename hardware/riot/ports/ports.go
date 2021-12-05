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

package ports

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// TV defines the television functions required by the Ports type.
type TV interface {
	GetCoords() coords.TelevisionCoords
}

// Input implements the input/output part of the RIOT (the IO in RIOT).
type Ports struct {
	riot bus.ChipBus
	tia  bus.ChipBus

	Panel       Peripheral
	LeftPlayer  Peripheral
	RightPlayer Peripheral

	playback EventPlayback
	recorder []EventRecorder

	monitor plugging.PlugMonitor

	// local copies of key chip memory registers

	// the latch bit represents the value of bit 6 of the VBLANK register. used
	// to affect how INPTx registers are written. see WriteINPTx() function
	latch bool

	// the swacnt field is the local copy of the SWACNT register. used to mask
	// bits in the SWCHA register. a 1 bit indicates the corresponding SWCHA
	// bit is used for output from the VCS, while a 0 bit indicates that it is
	// used for input to the VCS.
	swacnt uint8

	// the swcha field is the local copy of SWCHA register. note that we use
	// this only for reference purporses (particular the String() function).
	// the two swcha* derived fields below are of more use to the emulation
	// itself.
	swcha uint8

	// local copy of the SWCHB register. used exclusively for reference
	// purposes
	swchb uint8

	// the swcha field is a copy of the SWCHA register as it was written by the
	// CPU. it is not necessarily the value of SWCHA as written by the RIOT.
	//
	// we need this so that changing the SWACNT (by the CPU) will cause the
	// correct value to be written to be written to the SWCHA register.
	//
	// we can think of these as the input lines that are used in conjunction
	// with the SWACNT bits to create the SWCHA register
	swchaFromCPU uint8

	// swchaMux is the value that has most recently been written to the SWCHA
	// register by the RIOT
	//
	// the value has *not* been masked by the swacnt value
	//
	// we use it to mux the Player0 and Player 1 nibbles into the single register
	swchaMux uint8

	// events pushed onto the input queue
	pushed chan InputEvent

	// the following fields all relate to driven input, for either the driver
	// or for the passenger (the driven)
	fromDriver      chan InputEvent
	toPassenger     chan InputEvent
	checkForDriven  bool
	drivenInputData InputEvent

	// the time of driven events are measured by television coordinates
	//
	// not used except to synchronise driver and passenger emulations
	tv TV
}

// NewPorts is the preferred method of initialisation of the Ports type.
func NewPorts(riotMem bus.ChipBus, tiaMem bus.ChipBus) *Ports {
	p := &Ports{
		riot:         riotMem,
		tia:          tiaMem,
		recorder:     make([]EventRecorder, 0),
		swchaFromCPU: 0x00,
		swacnt:       0x00,
		latch:        false,
		pushed:       make(chan InputEvent, 64),
	}
	return p
}

// Snapshot returns a copy of the RIOT Ports sub-system in its current state.
func (p *Ports) Snapshot() *Ports {
	n := *p
	n.Panel = p.Panel.Snapshot()
	n.LeftPlayer = p.LeftPlayer.Snapshot()
	n.RightPlayer = p.RightPlayer.Snapshot()
	return &n
}

// Plumb new ChipBusses into the Ports sub-system. Depending on context it
// might be advidable for ResetPeripherals() to be called after plumbing has
// succeeded.
func (p *Ports) Plumb(riotMem bus.ChipBus, tiaMem bus.ChipBus) {
	p.riot = riotMem
	p.tia = tiaMem
	if p.Panel != nil {
		p.Panel.Plumb(p)
	}
	if p.LeftPlayer != nil {
		p.LeftPlayer.Plumb(p)
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Plumb(p)
	}
}

// Plug connects a peripheral to a player port.
func (p *Ports) Plug(port plugging.PortID, c NewPeripheral) error {
	periph := c(port, p)

	// notify monitor of plug event
	if p.monitor != nil {
		p.monitor.Plugged(port, periph.ID())
	}

	// attach any existing monitors to the new player peripheral
	if a, ok := periph.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(p.monitor)
	}

	switch port {
	case plugging.PortPanel:
		p.Panel = periph
	case plugging.PortLeftPlayer:
		p.LeftPlayer = periph
	case plugging.PortRightPlayer:
		p.RightPlayer = periph
	default:
		return fmt.Errorf("can't attach peripheral to port (%v)", port)
	}

	return nil
}

func (p *Ports) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("SWACNT: %#02x", p.swacnt))
	s.WriteString(fmt.Sprintf("  SWCHA: %#02x", p.swcha))
	s.WriteString(fmt.Sprintf("  SWCHA (from CPU): %#02x", p.swchaFromCPU))
	s.WriteString(fmt.Sprintf("  SWCHB: %#02x", p.swchb))
	return s.String()
}

// ResetPeripherals to an initial state.
func (p *Ports) ResetPeripherals() {
	if p.LeftPlayer != nil {
		p.LeftPlayer.Reset()
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Reset()
	}
	if p.Panel != nil {
		p.Panel.Reset()
	}
}

// Update checks to see if ChipData applies to the Input type and updates the
// internal controller/panel states accordingly. Returns true if ChipData
// requires more attention.
func (p *Ports) Update(data bus.ChipData) bool {
	switch data.Name {
	case "VBLANK":
		p.latch = data.Value&0x40 == 0x40

		// peripheral update
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case "SWCHA":
		p.swchaFromCPU = data.Value

		// mask value and set SWCHA register. some peripherals may call
		// WriteSWCHx() as part of the Update() function which will write over
		// this value.
		//
		// we should think of this write as the default event in case the
		// peripheral chooses to do nothing with the new value
		p.swcha = (p.swacnt ^ 0xff) | p.swchaFromCPU
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)

		// mask value with SWACNT bits before passing to peripheral
		data.Value &= p.swacnt

		// peripheral update for SWCHA
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case "SWACNT":
		p.swacnt = data.Value
		p.riot.ChipWrite(addresses.SWACNT, p.swacnt)

		// i/o bits have changed so change the data in the SWCHA register
		p.swcha = (p.swacnt ^ 0xff) | p.swchaFromCPU
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)

		// peripheral update for SWACNT
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

		// adjusting SWACNT also affects the SWCHA lines to the peripheral.
		// adjust SWCHA lines and update peripheral with new SWCHA data
		data = bus.ChipData{
			Name:  "SWCHA",
			Value: p.swcha,
		}
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case "SWCHB":
		fallthrough

	case "SWBCNT":
		_ = p.Panel.Update(data)
	}

	return false
}

// Step input state forward one cycle.
func (p *Ports) Step() {
	// not much to do here because most input operations happen on demand.
	// recharging of the paddle capacitors however happens (a little bit) every
	// step. also savekey needs to be processed every cycle
	if p.LeftPlayer != nil {
		p.LeftPlayer.Step()
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Step()
	}
	p.Panel.Step()
}

// SynchroniseWithDriver implies that the emulation will receive driven events
// from another emulation.
func (p *Ports) SynchroniseWithDriver(driver chan InputEvent, tv TV) error {
	if p.toPassenger != nil {
		return curated.Errorf("ports: cannot sync with driver: emulation already defined as a driver of input")
	}
	if p.playback != nil {
		return curated.Errorf("ports: cannot sync with driver: emulation is already receiving input from a playback")
	}
	p.tv = tv
	p.fromDriver = driver
	return nil
}

// SynchroniseWithPassenger connects the emulation to a second emulation (the
// passenger) to which user input events will be "driven".
func (p *Ports) SynchroniseWithPassenger(passenger chan InputEvent, tv TV) error {
	if p.fromDriver != nil {
		return curated.Errorf("ports: cannot sync with passenger: emulation already defined as being driven")
	}
	p.tv = tv
	p.toPassenger = passenger
	return nil
}

// AttachPlayback attaches an EventPlayback implementation.
func (p *Ports) AttachPlayback(b EventPlayback) error {
	if p.fromDriver != nil {
		return curated.Errorf("ports: cannot attach playback: emulation already defined as being driven")
	}
	p.playback = b
	return nil
}

// AttachEventRecorder attaches an EventRecorder implementation.
func (p *Ports) AttachEventRecorder(r EventRecorder) {
	p.recorder = append(p.recorder, r)
}

// AttchPlugMonitor implements the plugging.Monitorable interface.
func (p *Ports) AttachPlugMonitor(m plugging.PlugMonitor) {
	p.monitor = m

	// make sure any already attached peripherals know about the new monitor
	if a, ok := p.LeftPlayer.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}
	if a, ok := p.RightPlayer.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}
	if a, ok := p.Panel.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}

	// notify monitor of currently plugged peripherals
	if p.monitor != nil {
		p.monitor.Plugged(plugging.PortLeftPlayer, p.LeftPlayer.ID())
		p.monitor.Plugged(plugging.PortRightPlayer, p.RightPlayer.ID())
	}
}

// PeripheralID implements userinput.HandleInput interface.
//
// Consider using PeripheralID() function in the VCS type rather than this
// function directly.
func (p *Ports) PeripheralID(id plugging.PortID) plugging.PeripheralID {
	switch id {
	case plugging.PortPanel:
		return p.Panel.ID()
	case plugging.PortLeftPlayer:
		return p.LeftPlayer.ID()
	case plugging.PortRightPlayer:
		return p.RightPlayer.ID()
	}

	return plugging.PeriphNone
}

// WriteSWCHx implements the MemoryAccess interface.
func (p *Ports) WriteSWCHx(id plugging.PortID, data uint8) {
	switch id {
	case plugging.PortLeftPlayer:
		data &= 0xf0              // keep only the bits for player 0
		data |= p.swchaMux & 0x0f // combine with the existing player 1 bits
		p.swchaMux = data
		p.swcha = data & (p.swacnt ^ 0xff)
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case plugging.PortRightPlayer:
		data = (data & 0xf0) >> 4 // move bits into the player 1 nibble
		data |= p.swchaMux & 0xf0 // combine with the existing player 0 bits
		p.swchaMux = data
		p.swcha = data & (p.swacnt ^ 0xff)
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case plugging.PortPanel:
		p.swchb = data
		p.riot.ChipWrite(addresses.SWCHB, p.swchb)
	default:
		return
	}
}

// WriteINPTx implements the MemoryAccess interface.
func (p *Ports) WriteINPTx(inptx addresses.ChipRegister, data uint8) {
	// write memory if button is pressed or it is not and the button latch
	// is false
	if data != 0x80 || !p.latch {
		p.tia.ChipWrite(inptx, data)
	}
}
