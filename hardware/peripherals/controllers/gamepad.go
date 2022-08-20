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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Gamepad represents a two button gamepad. It is based on and is virtually
// the same as the regular Stick type.
//
// It also consumes the same event types as the regular Stick type.
//
// Information about how 2-button gamepads are expected to be handled found in
// the AtariAge thread below:
//
// https://atariage.com/forums/topic/158596-2-button-games-in-bb-using-sega-genesis-pads-with-the-2600/?tab=comments
type Gamepad struct {
	port plugging.PortID
	bus  ports.PeripheralBus

	axis uint8

	button      uint8
	buttonInptx chipbus.Register

	second      uint8
	secondInptx chipbus.Register

	// the gamepad is wired such that INPT0 (or INPT2 for the right player) is
	// wired to the VCC rail. the register is set to high on creation or reset
	// and set to low when unplugged.
	insertedInptx chipbus.Register
}

const (
	secondFire   = 0x00
	secondNoFire = 0x80
	inserted     = 0x80
	notInserted  = 0x00
)

// NewGamepad is the preferred method of initialisation for the Gamepad type
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewGamepad(instance *instance.Instance, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	pad := &Gamepad{
		port:   port,
		bus:    bus,
		axis:   axisCenter,
		button: stickNoFire,
		second: secondNoFire,
	}

	switch port {
	case plugging.PortLeftPlayer:
		pad.buttonInptx = chipbus.INPT4
		pad.secondInptx = chipbus.INPT1
		pad.insertedInptx = chipbus.INPT0
	case plugging.PortRightPlayer:
		pad.buttonInptx = chipbus.INPT5
		pad.secondInptx = chipbus.INPT3
		pad.insertedInptx = chipbus.INPT2
	}

	return pad
}

// Unplug implements the Peripheral interface.
func (pad *Gamepad) Unplug() {
	pad.bus.WriteSWCHx(pad.port, axisCenter)
	pad.bus.WriteINPTx(pad.buttonInptx, stickFire)
	pad.bus.WriteINPTx(pad.secondInptx, secondFire)
	pad.bus.WriteINPTx(pad.insertedInptx, notInserted)
}

// Snapshot implements the Peripheral interface.
func (pad *Gamepad) Snapshot() ports.Peripheral {
	n := *pad
	return &n
}

// Plumb implements the ports.Peripheral interface.
func (pad *Gamepad) Plumb(bus ports.PeripheralBus) {
	pad.bus = bus
}

// String implements the ports.Peripheral interface.
func (pad *Gamepad) String() string {
	return fmt.Sprintf("gamepad: axis=%02x fire=%02x", pad.axis, pad.button)
}

// PortID implements the ports.Peripheral interface.
func (pad *Gamepad) PortID() plugging.PortID {
	return pad.port
}

// ID implements the ports.Peripheral interface.
func (pad *Gamepad) ID() plugging.PeripheralID {
	return plugging.PeriphGamepad
}

// HandleEvent implements the ports.Peripheral interface.
func (pad *Gamepad) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.SecondFire:
		switch d := data.(type) {
		case bool:
			// support for playback file versions before v1.1
			if d {
				pad.second = secondFire
			} else {
				pad.second = secondNoFire
			}
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return false, curated.Errorf("gamepad: %v: unexpected event data", event)
			}
			if b {
				pad.second = secondFire
			} else {
				pad.second = secondNoFire
			}
		default:
			return false, curated.Errorf("gamepad: %v: unexpected event data", event)
		}
		pad.bus.WriteINPTx(pad.secondInptx, pad.second)
		return true, nil
	}

	switch event {
	case ports.NoEvent:
		return false, nil

	case ports.Fire:
		switch d := data.(type) {
		case bool:
			// support for playback file versions before v1.1
			if d {
				pad.button = stickFire
			} else {
				pad.button = stickNoFire
			}
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return false, curated.Errorf("gamepad: %v: unexpected event data", event)
			}
			if b {
				pad.button = stickFire
			} else {
				pad.button = stickNoFire
			}
		default:
			return false, curated.Errorf("gamepad: %v: unexpected event data", event)
		}
		pad.bus.WriteINPTx(pad.buttonInptx, pad.button)
		return true, nil

	case ports.Centre:
		switch d := data.(type) {
		case nil:
			// ideal path
		case ports.EventDataPlayback:
			if len(d) > 0 {
				return false, curated.Errorf("gamepad: %v: unexpected event data", event)
			}
		default:
			return false, curated.Errorf("gamepad: %v: unexpected event data", event)
		}
		pad.axis = axisCenter
		pad.bus.WriteSWCHx(pad.port, pad.axis)
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
		return false, curated.Errorf("gamepad: %v: unexpected event data", event)
	}

	// set/unset bits according to the event data
	if e == ports.DataStickTrue {
		pad.axis ^= axis
	} else if e == ports.DataStickFalse {
		pad.axis |= axis
	} else if e == ports.DataStickSet {
		pad.axis = axisCenter
		pad.axis ^= axis
	} else {
		return false, curated.Errorf("gamepad: %v: unexpected event data (%v)", event, e)
	}

	// update register
	pad.bus.WriteSWCHx(pad.port, pad.axis)

	return true, nil
}

// Update implements the ports.Peripheral interface.
func (pad *Gamepad) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.VBLANK:
		if data.Value&0x40 != 0x40 {
			if pad.button == stickNoFire {
				pad.bus.WriteINPTx(pad.buttonInptx, pad.button)
			}
			if pad.second == secondFire {
				pad.bus.WriteINPTx(pad.secondInptx, pad.second)
			}
		}

	default:
		return true
	}

	return false
}

// Step implements the ports.Peripheral interface.
func (pad *Gamepad) Step() {
	// if axis is deflected from the centre then make sure the SWCHA is set
	// correctly every cycle. this isn't necessary in all situations but ROMs
	// in which SWACNT is changed, axis state can be "forgotten". for example,
	// we can see this in the HeMan ROM.
	if pad.axis != 0xf0 {
		pad.bus.WriteSWCHx(pad.port, pad.axis)
	}
}

// Reset implements the ports.Peripheral interface.
func (pad *Gamepad) Reset() {
	pad.axis = axisCenter
	pad.button = stickNoFire
	pad.second = secondNoFire
	pad.bus.WriteSWCHx(pad.port, pad.axis)
	pad.bus.WriteINPTx(pad.buttonInptx, pad.button)
	pad.bus.WriteINPTx(pad.secondInptx, pad.second)
	pad.bus.WriteINPTx(pad.insertedInptx, inserted)
}

// IsActive implements the ports.Peripheral interface.
func (pad *Gamepad) IsActive() bool {
	return pad.button == stickFire || pad.axis != axisCenter || pad.second == secondFire
}
