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
)

// Input implements the input/output part of the RIOT (the IO in RIOT).
type Ports struct {
	riot bus.ChipBus
	tia  bus.ChipBus

	Panel       Peripheral
	LeftPlayer  Peripheral
	RightPlayer Peripheral

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

	// the swchaFromCPU field is a copy of the SWCHA register as it was written
	// by the CPU. it is not necessarily the value of SWCHA as written by the
	// RIOT.
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

	// port B equivalents of the above. there is no swchbMux field because only
	// one peripheral uses port B at a time.
	//
	// there is a swchbFromPeriph however. this is the value as written by the
	// peripheral (the panel) before SWBCNT has been applied to it
	swbcnt          uint8
	swchb           uint8
	swchbFromCPU    uint8
	swchbFromPeriph uint8
}

// NewPorts is the preferred method of initialisation of the Ports type.
func NewPorts(riotMem bus.ChipBus, tiaMem bus.ChipBus) *Ports {
	p := &Ports{
		riot:         riotMem,
		tia:          tiaMem,
		swchaFromCPU: 0x00,
		swacnt:       0x00,
		latch:        false,
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
		p.swcha = ^p.swacnt | p.swchaFromCPU
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)

		// mask value with SWACNT bits before passing to peripheral
		data.Value &= p.swacnt
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case "SWACNT":
		p.swacnt = data.Value
		p.riot.ChipWrite(addresses.SWACNT, p.swacnt)

		// peripheral update for SWACNT
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

		// i/o bits have changed so change the data in the SWCHA register
		p.swcha = ^p.swacnt | p.swchaFromCPU
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)

		// adjusting SWACNT also affects the SWCHA lines to the peripheral.
		// adjust SWCHA lines and update peripheral with new SWCHA data
		data = bus.ChipData{
			Name:  "SWCHA",
			Value: p.swcha,
		}
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case "SWCHB":
		p.swchbFromCPU = data.Value
		p.swchb = (p.swchbFromCPU & p.swbcnt) | (p.swchbFromPeriph & ^p.swbcnt)
		p.riot.ChipWrite(addresses.SWCHB, p.swchb)

	case "SWBCNT":
		p.swbcnt = data.Value
		p.riot.ChipWrite(addresses.SWBCNT, p.swbcnt)

		p.swchb = (p.swchbFromCPU & p.swbcnt) | (p.swchbFromPeriph & ^p.swbcnt)
		p.riot.ChipWrite(addresses.SWCHB, p.swchb)
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

// PeripheralID returns the ID of the peripheral in the identified port.
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

// WriteSWCHx implements the peripheral.PeripheralBus interface.
func (p *Ports) WriteSWCHx(id plugging.PortID, data uint8) {
	switch id {
	case plugging.PortLeftPlayer:
		data &= 0xf0              // keep only the bits for player 0
		data |= p.swchaMux & 0x0f // combine with the existing player 1 bits
		p.swchaMux = data
		p.swcha = data & ^p.swacnt
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case plugging.PortRightPlayer:
		data = (data & 0xf0) >> 4 // move bits into the player 1 nibble
		data |= p.swchaMux & 0xf0 // combine with the existing player 0 bits
		p.swchaMux = data
		p.swcha = data & ^p.swacnt
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case plugging.PortPanel:
		p.swchbFromPeriph = data
		p.swchb = (p.swchbFromCPU & p.swbcnt) | (p.swchbFromPeriph & ^p.swbcnt)
		p.riot.ChipWrite(addresses.SWCHB, p.swchb)
	default:
		return
	}
}

// WriteINPTx implements the peripheral.PeripheralBus interface.
func (p *Ports) WriteINPTx(inptx addresses.ChipRegister, data uint8) {
	// write memory if button is pressed or it is not and the button latch
	// is false
	if data != 0x80 || !p.latch {
		p.tia.ChipWrite(inptx, data)
	}
}

// HandleInputEvent forwards the InputEvent to the perupheral in the correct
// port. Returns true if the event was handled and false if not.
func (p *Ports) HandleInputEvent(inp InputEvent) (bool, error) {
	var handled bool
	var err error

	switch inp.Port {
	case plugging.PortPanel:
		handled, err = p.Panel.HandleEvent(inp.Ev, inp.D)
	case plugging.PortLeftPlayer:
		handled, err = p.LeftPlayer.HandleEvent(inp.Ev, inp.D)
	case plugging.PortRightPlayer:
		handled, err = p.RightPlayer.HandleEvent(inp.Ev, inp.D)
	}

	// if error was because of an unhandled event then return without error
	if err != nil {
		return handled, curated.Errorf("ports: %v", err)
	}

	return handled, nil
}
