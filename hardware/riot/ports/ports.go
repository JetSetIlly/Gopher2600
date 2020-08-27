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

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// MemoryAccess encapsulates the input device buses to both riot and tia memory
type MemoryAccess struct {
	RIOT bus.InputDeviceBus
	TIA  bus.InputDeviceBus
}

// Input implements the input/output part of the RIOT (the IO in RIOT)
type Ports struct {
	mem MemoryAccess

	Panel   Peripheral
	Player0 Peripheral
	Player1 Peripheral
}

// NewPorts is the preferred method of initialisation of the Ports type
func NewPorts(riotMem bus.ChipBus, tiaMem bus.ChipBus) (*Ports, error) {
	p := &Ports{
		mem: MemoryAccess{
			RIOT: riotMem.(bus.InputDeviceBus),
			TIA:  tiaMem.(bus.InputDeviceBus),
		},
	}

	p.Panel = NewPanel(&p.mem)
	if p.Panel == nil {
		return nil, fmt.Errorf("can't create control panel")
	}

	return p, nil
}

func (p *Ports) Reset() {
	if p.Player0 != nil {
		p.Player0.Reset()
	}
	if p.Player1 != nil {
		p.Player1.Reset()
	}
}

// Update checks to see if ChipData applies to the Input type and updates the
// internal controller/panel states accordingly. Returns true if ChipData
// requires more attention.
func (p *Ports) Update(data bus.ChipData) bool {
	// we forward the Update() call to all Ports because they might all be
	// interested in the data
	r := false
	if p.Player0 != nil {
		r = r || p.Player0.Update(data)
	}
	if p.Player1 != nil {
		r = r || p.Player1.Update(data)
	}
	r = r || p.Panel.Update(data)
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
	if p, ok := p.Player0.(RecordablePeripheral); ok {
		p.AttachPlayback(b)
	}
	if p, ok := p.Player1.(RecordablePeripheral); ok {
		p.AttachPlayback(b)
	}
	if p, ok := p.Panel.(RecordablePeripheral); ok {
		p.AttachPlayback(b)
	}
}

// AttachEventRecorder attaches an EventRecorder implementation to all ports
// that implement RecordablePort
func (p *Ports) AttachEventRecorder(r EventRecorder) {
	if p, ok := p.Player0.(RecordablePeripheral); ok {
		p.AttachEventRecorder(r)
	}
	if p, ok := p.Player1.(RecordablePeripheral); ok {
		p.AttachEventRecorder(r)
	}
	if p, ok := p.Panel.(RecordablePeripheral); ok {
		p.AttachEventRecorder(r)
	}
}

// GetPlayback requests playback events from all attached and eligible peripherals
func (p *Ports) GetPlayback() error {
	if p, ok := p.Player0.(RecordablePeripheral); ok {
		err := p.GetPlayback()
		if err != nil {
			return err
		}
	}
	if p, ok := p.Player1.(RecordablePeripheral); ok {
		err := p.GetPlayback()
		if err != nil {
			return err
		}
	}
	if p, ok := p.Panel.(RecordablePeripheral); ok {
		err := p.GetPlayback()
		if err != nil {
			return err
		}
	}
	return nil
}

// AttachPlayer0 attaches a peripheral (represented by a PeripheralConstructor) to port 0
func (p *Ports) AttachPlayer0(c PeripheralConstructor) error {
	p.Player0 = c(&p.mem)
	if p.Player0 == nil {
		return fmt.Errorf("can't attach peripheral to player 0 port")
	}
	return nil
}

// AttachPlayer1 attaches a peripheral (represented by a PeripheralConstructor) to port 1
func (p *Ports) AttachPlayer1(c PeripheralConstructor) error {
	p.Player1 = c(&p.mem)
	if p.Player1 == nil {
		return fmt.Errorf("can't attach peripheral to player 1 port")
	}
	return nil
}
