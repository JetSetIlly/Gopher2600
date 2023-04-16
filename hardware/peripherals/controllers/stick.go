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
	"strconv"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
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
	port plugging.PortID
	bus  ports.PeripheralBus

	axis uint8

	button      uint8
	buttonInptx chipbus.Register
}

// NewStick is the preferred method of initialisation for the Stick type
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewStick(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	stk := &Stick{
		port:   port,
		bus:    bus,
		axis:   axisCenter,
		button: stickNoFire,
	}

	switch port {
	case plugging.PortLeft:
		stk.buttonInptx = chipbus.INPT4
	case plugging.PortRight:
		stk.buttonInptx = chipbus.INPT5
	}

	return stk
}

// Unplug implements the Peripheral interface.
func (stk *Stick) Unplug() {
	stk.bus.WriteSWCHx(stk.port, axisCenter)
	stk.bus.WriteINPTx(stk.buttonInptx, stickFire)
}

// Snapshot implements the Peripheral interface.
func (stk *Stick) Snapshot() ports.Peripheral {
	n := *stk
	return &n
}

// Plumb implements the ports.Peripheral interface.
func (stk *Stick) Plumb(bus ports.PeripheralBus) {
	stk.bus = bus
}

// String implements the ports.Peripheral interface.
func (stk *Stick) String() string {
	return fmt.Sprintf("stick: axis=%02x fire=%02x", stk.axis, stk.button)
}

// PortID implements the ports.Peripheral interface.
func (stk *Stick) PortID() plugging.PortID {
	return stk.port
}

// ID implements the ports.Peripheral interface.
func (stk *Stick) ID() plugging.PeripheralID {
	return plugging.PeriphStick
}

// HandleEvent implements the ports.Peripheral interface.
func (stk *Stick) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.NoEvent:
		return false, nil

	case ports.Fire:
		switch d := data.(type) {
		case bool:
			// support for playback file versions before v1.1
			if d {
				stk.button = stickFire
			} else {
				stk.button = stickNoFire
			}
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return false, fmt.Errorf("stick: %v: unexpected event data", event)
			}
			if b {
				stk.button = stickFire
			} else {
				stk.button = stickNoFire
			}
		default:
			return false, fmt.Errorf("stick: %v: unexpected event data", event)
		}
		stk.bus.WriteINPTx(stk.buttonInptx, stk.button)
		return true, nil

	case ports.Centre:
		switch d := data.(type) {
		case nil:
			// ideal path
		case ports.EventDataPlayback:
			if len(d) > 0 {
				return false, fmt.Errorf("stick: %v: unexpected event data", event)
			}
		default:
			return false, fmt.Errorf("stick: %v: unexpected event data", event)
		}
		stk.axis = axisCenter
		stk.bus.WriteSWCHx(stk.port, stk.axis)
		return true, nil
	}

	var axis uint8

	switch event {
	case ports.Left:
		axis = axisLeft
	case ports.Right:
		axis = axisRight
	case ports.Up:
		axis = axisUp
	case ports.Down:
		axis = axisDown
	case ports.LeftUp:
		axis = axisLeft | axisUp
	case ports.LeftDown:
		axis = axisLeft | axisDown
	case ports.RightUp:
		axis = axisRight | axisUp
	case ports.RightDown:
		axis = axisRight | axisDown
	default:
		return false, nil
	}

	var e ports.EventDataStick

	// other stick events can be treated the same (although note the default case)
	switch d := data.(type) {
	case ports.EventDataStick:
		e = d
	case ports.EventDataPlayback:
		e = ports.EventDataStick(d)
	default:
		return false, fmt.Errorf("stick: %v: unexpected event data", event)
	}

	// set/unset bits according to the event data
	if e == ports.DataStickTrue {
		stk.axis ^= axis
	} else if e == ports.DataStickFalse {
		stk.axis |= axis
	} else if e == ports.DataStickSet {
		stk.axis = axisCenter
		stk.axis ^= axis
	} else {
		return false, fmt.Errorf("stick: %v: unexpected event data (%v)", event, e)
	}

	// update register
	stk.bus.WriteSWCHx(stk.port, stk.axis)

	return true, nil
}

// Update implements the ports.Peripheral interface.
func (stk *Stick) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.VBLANK:
		if data.Value&0x40 != 0x40 {
			if stk.button == stickNoFire {
				stk.bus.WriteINPTx(stk.buttonInptx, stk.button)
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
		stk.bus.WriteSWCHx(stk.port, stk.axis)
	}
}

// Reset implements the ports.Peripheral interface.
func (stk *Stick) Reset() {
	stk.axis = axisCenter
	stk.button = stickNoFire
	stk.bus.WriteSWCHx(stk.port, stk.axis)
	stk.bus.WriteINPTx(stk.buttonInptx, stk.button)
}

// IsActive implements the ports.Peripheral interface.
func (stk *Stick) IsActive() bool {
	return stk.button == stickFire || stk.axis != axisCenter
}
