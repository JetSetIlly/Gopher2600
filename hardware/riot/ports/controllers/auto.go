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

package controllers

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Auto handles the automatic switching between controller types.
type Auto struct {
	port       plugging.PortID
	bus        ports.PeripheralBus
	controller ports.Peripheral
	monitor    plugging.PlugMonitor

	paddleTouchCt int
}

// NewAuto is the preferred method of initialisation for the Auto type.
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewAuto(port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	aut := &Auto{
		port: port,
		bus:  bus,
	}

	aut.Reset()
	return aut
}

// Plumb implements the Peripheral interface.
func (aut *Auto) Plumb(bus ports.PeripheralBus) {
	aut.bus = bus
	aut.controller.Plumb(bus)
}

// String implements the ports.Peripheral interface.
func (aut *Auto) String() string {
	return aut.controller.String()
}

// PortID implements the ports.Peripheral interface.
func (aut *Auto) PortID() plugging.PortID {
	return aut.port
}

// Name implements the ports.Peripheral interface.
func (aut *Auto) Name() string {
	return aut.controller.Name()
}

// HandleEvent implements the ports.Peripheral interface.
func (aut *Auto) HandleEvent(event ports.Event, data ports.EventData) error {
	switch event {
	case ports.Left:
		aut.toStick()
	case ports.Right:
		aut.toStick()
	case ports.Up:
		aut.toStick()
	case ports.Down:
		aut.toStick()
	case ports.Fire:
		// no auto switch for fire events
	case ports.PaddleSet:
		aut.toPaddle()
	case ports.KeyboardDown:
		aut.toKeyboard()
	case ports.KeyboardUp:
		aut.toKeyboard()
	}

	err := aut.controller.HandleEvent(event, data)

	return err
}

// Update implements the ports.Peripheral interface.
func (aut *Auto) Update(data bus.ChipData) bool {
	switch data.Name {
	case "SWACNT":
		if data.Value&0xf0 == 0xf0 {
			aut.toKeyboard()
		} else if data.Value&0xf0 == 0x00 {
			aut.toStick()
		}
	}

	return aut.controller.Update(data)
}

// Step implements the ports.Peripheral interface.
func (aut *Auto) Step() {
	aut.controller.Step()
}

// Reset implements the ports.Peripheral interface.
func (aut *Auto) Reset() {
	aut.controller = NewStick(aut.port, aut.bus)
}

func (aut *Auto) toStick() {
	aut.paddleTouchCt = 0
	if _, ok := aut.controller.(*Stick); !ok {
		aut.controller = NewStick(aut.port, aut.bus)
		aut.plug()
	}
}

func (aut *Auto) toPaddle() {
	if _, ok := aut.controller.(*Paddle); !ok {
		const autoPaddleSensitivity = 20

		if aut.paddleTouchCt < autoPaddleSensitivity {
			aut.paddleTouchCt++
			if aut.paddleTouchCt < autoPaddleSensitivity {
				return
			}
		}

		aut.controller = NewPaddle(aut.port, aut.bus)
		aut.plug()
	}
}

func (aut *Auto) toKeyboard() {
	aut.paddleTouchCt = 0
	if _, ok := aut.controller.(*Keyboard); !ok {
		aut.controller = NewKeyboard(aut.port, aut.bus)
		aut.plug()
	}
}

// plug is called by toStick(), toPaddle() and toKeyboard() and handles the
// plug monitor.
func (aut *Auto) plug() {
	// notify any peripheral monitors
	if aut.monitor != nil {
		aut.monitor.Plugged(aut.port, aut.controller.Name())
	}

	// attach any monitors to newly plugged controllers
	if a, ok := aut.controller.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(aut.monitor)
	}
}

// AttachPlugMonitor implements the plugging.Monitorable interface.
func (aut *Auto) AttachPlugMonitor(m plugging.PlugMonitor) {
	aut.monitor = m

	if a, ok := aut.controller.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}
}
