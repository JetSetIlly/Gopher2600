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
)

// Input implements the input/output part of the RIOT (the IO in RIOT).
type Ports struct {
	riot bus.ChipBus
	tia  bus.ChipBus

	Panel   Peripheral
	Player0 Peripheral
	Player1 Peripheral

	playback EventPlayback
	recorder EventRecorder

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
	// we need this so that chancing the SWACNT (by the CPU) will cause the
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
	// we use it to mux the Player0 and Player 1 nibbles into the single
	// register
	swchaMux uint8
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
	p.Panel = NewPanel(p)
	return p
}

// Snapshot returns a copy of the RIOT Ports sub-system in its current state.
func (p *Ports) Snapshot() *Ports {
	n := *p
	return &n
}

// Plumb new ChipBusses into the Ports sub-system.
func (p *Ports) Plumb(riotMem bus.ChipBus, tiaMem bus.ChipBus) {
	p.riot = riotMem
	p.tia = tiaMem
	if p.Panel != nil {
		p.Panel.Plumb(p)
	}
	if p.Player0 != nil {
		p.Player0.Plumb(p)
	}
	if p.Player1 != nil {
		p.Player1.Plumb(p)
	}
}

// AttachPlayer attaches a peripheral (represented by a PeripheralConstructor) to a port.
func (p *Ports) AttachPlayer(id PortID, c NewPeripheral) error {
	switch id {
	case Player0ID:
		p.Player0 = c(Player0ID, p)
		if p.Player0 == nil {
			return fmt.Errorf("can't attach peripheral to player 0 port")
		}
	case Player1ID:
		p.Player1 = c(Player1ID, p)
		if p.Player1 == nil {
			return fmt.Errorf("can't attach peripheral to player 1 port")
		}
	default:
		return fmt.Errorf("can't attach peripheral to port (%v)", id)
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

// Reset peripherals to an initial state.
func (p *Ports) Reset() {
	if p.Player0 != nil {
		p.Player0.Reset()
	}
	if p.Player1 != nil {
		p.Player1.Reset()
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
		_ = p.Player0.Update(data)
		_ = p.Player1.Update(data)

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
		_ = p.Player0.Update(data)
		_ = p.Player1.Update(data)

	case "SWACNT":
		p.swacnt = data.Value
		p.riot.ChipWrite(addresses.SWACNT, p.swacnt)

		// i/o bits have changed so change the data in the SWCHA register
		p.swcha = (p.swacnt ^ 0xff) | p.swchaFromCPU
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)

		// peripheral update for SWACNT
		_ = p.Player0.Update(data)
		_ = p.Player1.Update(data)

		// adjusting SWACNT also affects the SWCHA lines to the peripheral.
		// adjust SWCHA lines and update peripheral with new SWCHA data
		data = bus.ChipData{
			Name:  "SWCHA",
			Value: p.swcha,
		}
		_ = p.Player0.Update(data)
		_ = p.Player1.Update(data)

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
	// step.
	if p.Player0 != nil {
		p.Player0.Step()
	}
	if p.Player1 != nil {
		p.Player1.Step()
	}
	p.Panel.Step()
}

// AttachPlayback attaches an EventPlayback implementation to all ports that
// implement RecordablePort.
func (p *Ports) AttachPlayback(b EventPlayback) {
	p.playback = b
}

// AttachEventRecorder attaches an EventRecorder implementation to all ports
// that implement RecordablePort.
func (p *Ports) AttachEventRecorder(r EventRecorder) {
	p.recorder = r
}

// GetPlayback requests playback events from all attached and eligible peripherals.
func (p *Ports) GetPlayback() error {
	if p.playback == nil {
		return nil
	}

	// loop with GetPlayback() until we encounter a NoPortID or NoEvent
	// condition. there might be more than one entry for a particular
	// frame/scanline/horizpas state so we need to make sure we've processed
	// them all.
	//
	// this happens in particular with recordings that were made of  ROMs with
	// panel setup configurations (see setup package) - where the switches are
	// set when the TV state is at fr=0 sl=0 hp=0
	morePlayback := true
	for morePlayback {
		id, ev, v, err := p.playback.GetPlayback()
		if err != nil {
			return err
		}

		morePlayback = id != NoPortID && ev != NoEvent
		if morePlayback {
			err := p.HandleEvent(id, ev, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Ports) HandleEvent(id PortID, ev Event, d EventData) error {
	var err error

	switch id {
	case PanelID:
		err = p.Panel.HandleEvent(ev, d)
	case Player0ID:
		err = p.Player0.HandleEvent(ev, d)
	case Player1ID:
		err = p.Player1.HandleEvent(ev, d)
	}

	if err != nil {
		return curated.Errorf("ports: %v", err)
	}

	// record event with the EventRecorder
	if p.recorder != nil {
		return p.recorder.RecordEvent(id, ev, d)
	}

	return nil
}

// WriteSWCHx implements the MemoryAccess interface.
func (p *Ports) WriteSWCHx(id PortID, data uint8) {
	switch id {
	case Player0ID:
		data &= 0xf0              // keep only the bits for player 0
		data |= p.swchaMux & 0x0f // combine with the existing player 1 bits
		p.swchaMux = data
		p.swcha = data & (p.swacnt ^ 0xff)
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case Player1ID:
		data = (data & 0xf0) >> 4 // move bits into the player 1 nibble
		data |= p.swchaMux & 0xf0 // combine with the existing player 0 bits
		p.swchaMux = data
		p.swcha = data & (p.swacnt ^ 0xff)
		p.riot.ChipWrite(addresses.SWCHA, p.swcha)
	case PanelID:
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
