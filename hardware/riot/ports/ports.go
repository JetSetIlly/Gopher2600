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

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// Input implements the input/output part of the RIOT (the IO in RIOT)
type Ports struct {
	riot   bus.InputDeviceBus
	tia    bus.InputDeviceBus
	swacnt uint8
	latch  bool

	Panel   Peripheral
	Player0 Peripheral
	Player1 Peripheral

	playback EventPlayback
	recorder EventRecorder
}

// NewPorts is the preferred method of initialisation of the Ports type
func NewPorts(riotMem bus.ChipBus, tiaMem bus.ChipBus) (*Ports, error) {
	p := &Ports{
		riot:   riotMem.(bus.InputDeviceBus),
		tia:    tiaMem.(bus.InputDeviceBus),
		swacnt: 0x00,
		latch:  false,
	}

	p.Panel = NewPanel(p)
	if p.Panel == nil {
		return nil, fmt.Errorf("can't create control panel")
	}

	return p, nil
}

// AttachPlayer attaches a peripheral (represented by a PeripheralConstructor) to a port
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

// Reset peripherals to an initial state
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

	case "SWCHA":
		// SWCHA is filtered by SWACNT. this maybe should be done in the memory
		// sub-system before it is propagated
		//
		// !!TODO: think about moving SWACNT filtering to the memory sub-system
		data.Value &= p.swacnt

		p.riot.InputDeviceWrite(addresses.SWCHA, data.Value, 0x00)

	case "SWACNT":
		p.swacnt = data.Value
		p.riot.InputDeviceWrite(addresses.SWACNT, data.Value, 0x00)

	default:
		return true
	}

	_ = p.Player0.Update(data)
	_ = p.Player1.Update(data)
	_ = p.Panel.Update(data)

	return false
}

// Step input state forward one cycle
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
// implement RecordablePort
func (p *Ports) AttachPlayback(b EventPlayback) {
	p.playback = b
}

// AttachEventRecorder attaches an EventRecorder implementation to all ports
// that implement RecordablePort
func (p *Ports) AttachEventRecorder(r EventRecorder) {
	p.recorder = r
}

// GetPlayback requests playback events from all attached and eligible peripherals
func (p *Ports) GetPlayback() error {
	if p.playback == nil {
		return nil
	}

	id, ev, v, err := p.playback.GetPlayback()
	if err != nil {
		return err
	}

	if id == NoPortID || ev == NoEvent {
		return nil
	}

	return p.HandleEvent(id, ev, v)
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
		return errors.New(errors.InputError, err)
	}

	// record event with the EventRecorder
	if p.recorder != nil {
		return p.recorder.RecordEvent(id, ev, d)
	}

	return nil
}

// WriteSWCHx implements the MemoryAccess interface
func (p *Ports) WriteSWCHx(id PortID, data uint8) {
	switch id {
	case Player0ID:
		p.riot.InputDeviceWrite(addresses.SWCHA, data&(p.swacnt^0xff), 0xf0)
	case Player1ID:
		p.riot.InputDeviceWrite(addresses.SWCHA, (data>>4)&(p.swacnt^0xff), 0x0f)
	case PanelID:
		p.riot.InputDeviceWrite(addresses.SWCHB, data, 0xff)
	default:
		return
	}
}

// WriteINPTx implements the MemoryAccess interface
func (p *Ports) WriteINPTx(inptx addresses.ChipRegister, data uint8) {
	// write memory if button is pressed or it is not and the button latch
	// is false
	if data != 0x80 || !p.latch {
		p.tia.InputDeviceWrite(inptx, data, 0x80)
	}
}
