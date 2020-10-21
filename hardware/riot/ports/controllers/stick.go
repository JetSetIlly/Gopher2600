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
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// stick values.
const (
	stickFire   = 0x00
	stickNoFire = 0x80
	axisRight   = 0x80
	axisLeft    = 0x40
	axisDown    = 0x20
	axisUp      = 0x10
	axisCenter  = 0xf0
)

// Stick represents the VCS digital joystick controller.
type Stick struct {
	id  ports.PortID
	bus ports.PeripheralBus

	axis   uint8
	button uint8

	inptx addresses.ChipRegister
}

// NewStick is the preferred method of initialisation for the Stick type
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewStick(id ports.PortID, bus ports.PeripheralBus) ports.Peripheral {
	stk := &Stick{
		id:     id,
		bus:    bus,
		axis:   axisCenter,
		button: stickNoFire,
	}

	switch id {
	case ports.Player0ID:
		stk.inptx = addresses.INPT4
	case ports.Player1ID:
		stk.inptx = addresses.INPT5
	}

	stk.Reset()
	return stk
}

// Plumb implements the ports.Peripheral interface.
func (stk *Stick) Plumb(bus ports.PeripheralBus) {
	stk.bus = bus
}

// String implements the ports.Peripheral interface.
func (stk *Stick) String() string {
	return fmt.Sprintf("stick: axis=%02x fire=%02x", stk.axis, stk.button)
}

// Name implements the ports.Peripheral interface.
func (stk *Stick) Name() string {
	return "Stick"
}

// HandleEvent implements the ports.Peripheral interface.
func (stk *Stick) HandleEvent(event ports.Event, data ports.EventData) error {
	switch event {
	default:
		return curated.Errorf(UnhandledEvent, stk.Name(), event)

	case ports.NoEvent:

	case ports.Left:
		if data.(bool) {
			stk.axis ^= axisLeft
		} else {
			stk.axis |= axisLeft
		}
		stk.bus.WriteSWCHx(stk.id, stk.axis)

	case ports.Right:
		if data.(bool) {
			stk.axis ^= axisRight
		} else {
			stk.axis |= axisRight
		}
		stk.bus.WriteSWCHx(stk.id, stk.axis)

	case ports.Up:
		if data.(bool) {
			stk.axis ^= axisUp
		} else {
			stk.axis |= axisUp
		}
		stk.bus.WriteSWCHx(stk.id, stk.axis)

	case ports.Down:
		if data.(bool) {
			stk.axis ^= axisDown
		} else {
			stk.axis |= axisDown
		}
		stk.bus.WriteSWCHx(stk.id, stk.axis)

	case ports.Fire:
		if data.(bool) {
			stk.button = stickFire
		} else {
			stk.button = stickNoFire
		}
		stk.bus.WriteINPTx(stk.inptx, stk.button)
	}

	return nil
}

// Update implements the ports.Peripheral interface.
func (stk *Stick) Update(data bus.ChipData) bool {
	switch data.Name {
	case "VBLANK":
		if data.Value&0x40 != 0x40 {
			if stk.button == stickNoFire {
				stk.bus.WriteINPTx(stk.inptx, stk.button)
			}
		}

	default:
		return true
	}

	return false
}

// Step implements the ports.Peripheral interface.
func (stk *Stick) Step() {
	// if axis is deflected from the centre then make sure the SWCHA is set
	// correctly every cycle. this isn't necessary in all situations but ROMs
	// in which SWACNT is changed, axis state can be "forgotten". for example,
	// we can see this in the HeMan ROM.
	if stk.axis != 0xf0 {
		stk.bus.WriteSWCHx(stk.id, stk.axis)
	}
}

// Reset implements the ports.Peripheral interface.
func (stk *Stick) Reset() {
	stk.bus.WriteSWCHx(stk.id, stk.axis)
	stk.bus.WriteINPTx(stk.inptx, stk.button)
}
