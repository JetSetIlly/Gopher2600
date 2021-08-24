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
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// paddle values.
const (
	paddleFire   = 0x00
	paddleNoFire = 0xf0

	// paddleChargeRate governs the rate at which the controller capacitor fills.
	// the tick value is increased by the sensitivity value every cycle; once
	// it reaches or exceeds the resistance value, the charge value is
	// increased. the charge value is the value written to the INPTx register.
	//
	// the following value is found by 1/N where N is the number of ticks
	// required for the charge value to increase by 1. eg 1/150==0.0067.
	paddleChargeRate = 0.0067
)

// Paddle represents the VCS paddle controller type.
type Paddle struct {
	port plugging.PortID
	bus  ports.PeripheralBus

	// register to write puck charge to
	inptx addresses.ChipRegister

	// button data is always written to SWCHA but which bit depends on the player
	buttonMask uint8

	// values indicating paddle state
	charge     uint8
	resistance float32
	ticks      float32

	// the state of the fire button
	fire uint8
}

// NewPaddle is the preferred method of initialisation for the Paddle type
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewPaddle(port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	pdl := &Paddle{
		port: port,
		bus:  bus,
	}

	// !!TODO: support for paddle player 3 and paddle player 4
	switch port {
	case plugging.PortLeftPlayer:
		pdl.inptx = addresses.INPT0
		pdl.buttonMask = 0x80
	case plugging.PortRightPlayer:
		pdl.inptx = addresses.INPT1
		pdl.buttonMask = 0x40
	}

	return pdl
}

// Plumb implements the ports.Peripheral interface.
func (pdl *Paddle) Plumb(bus ports.PeripheralBus) {
	pdl.bus = bus
}

// String implements the ports.Peripheral interface.
func (pdl *Paddle) String() string {
	return fmt.Sprintf("paddle: button=%02x charge=%v resistance=%.02f", pdl.fire, pdl.charge, pdl.resistance)
}

// PortID implements the ports.Peripheral interface.
func (pdl *Paddle) PortID() plugging.PortID {
	return pdl.port
}

// ID implements the ports.Peripheral interface.
func (pdl *Paddle) ID() plugging.PeripheralID {
	return plugging.PeriphPaddle
}

// HandleEvent implements the ports.Peripheral interface.
func (pdl *Paddle) HandleEvent(event ports.Event, data ports.EventData) error {
	switch event {
	case ports.NoEvent:
		return nil

	case ports.Fire:
		switch d := data.(type) {
		case bool:
			if d {
				pdl.fire = paddleFire
			} else {
				pdl.fire = paddleNoFire
			}
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return curated.Errorf("paddle: %v: unexpected event data", event)
			}
			if b {
				pdl.fire = paddleFire
			} else {
				pdl.fire = paddleNoFire
			}
		default:
			return curated.Errorf("paddle: %v: unexpected event data", event)
		}

		pdl.bus.WriteSWCHx(pdl.port, pdl.fire)

	case ports.PaddleSet:
		var r float32

		switch d := data.(type) {
		case float32:
			r = d
		case ports.EventDataPlayback:
			f, err := strconv.ParseFloat(string(d), 32)
			if err != nil {
				return curated.Errorf("paddle: %v: unexpected event data", event)
			}
			r = float32(f)
		default:
			return curated.Errorf("paddle: %v: unexpected event data", event)
		}

		// reverse value so that we left and right are the correct way around (for a mouse)
		pdl.resistance = 1.0 - r

	default:
		// silently ignore unhandled event
		return nil
	}

	return nil
}

// Update implements the ports.Peripheral interface.
func (pdl *Paddle) Update(data bus.ChipData) bool {
	switch data.Name {
	case "VBLANK":
		if data.Value&0x80 == 0x80 {
			// ground paddle's puck
			pdl.charge = 0x00
			pdl.ticks = 0.0
			pdl.bus.WriteINPTx(pdl.inptx, 0x00)
		}

	default:
		return true
	}

	return false
}

// Step implements the ports.Peripheral interface.
func (pdl *Paddle) Step() {
	if pdl.charge < 255 {
		pdl.ticks += paddleChargeRate
		if pdl.ticks >= pdl.resistance {
			pdl.ticks = 0.0
			pdl.charge++
			pdl.bus.WriteINPTx(pdl.inptx, pdl.charge)
		}
	}

	// like with the stick we should make sure the fire button retains it's
	// depressed state. see Stick.Step() function for commentary
	if pdl.fire != paddleNoFire {
		pdl.bus.WriteSWCHx(pdl.port, pdl.fire)
	}
}

// Reset implements the ports.Peripheral interface.
func (pdl *Paddle) Reset() {
	pdl.charge = 0
	pdl.ticks = 0.0
	pdl.resistance = 0.0
}
