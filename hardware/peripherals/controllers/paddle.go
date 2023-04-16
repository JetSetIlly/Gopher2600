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
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

const (
	// paddleChargeRate governs the rate at which the controller capacitor fills.
	// the tick value is increased by the sensitivity value every cycle; once
	// it reaches or exceeds the resistance value, the charge value is
	// increased. the charge value is the value written to the INPTx register.
	//
	// the following value is found by 1/N where N is the number of ticks
	// required for the charge value to increase by 1. eg 1/150==0.0067.
	paddleChargeRate = 0.0067

	// sets the fire button for both paddles to nofire. WriteSWCHx() will shift
	// the bits into the correct position for the right port
	paddleNoFire = 0xf0
)

type paddle struct {
	// register to write puck charge to
	inptx chipbus.Register

	// button data is always written to SWCHA but which bit depends on the paddle
	buttonMask uint8

	// values indicating paddle state
	charge     uint8
	resistance float32
	ticks      float32

	// the state of the fire button
	fire bool
}

// Paddles represents a pair of paddles inserted into the same port
type Paddles struct {
	port plugging.PortID
	bus  ports.PeripheralBus

	// maximum of two paddles per paddles pair
	paddles [2]paddle
}

// NewPaddlePair is the preferred method of initialisation for the PaddlePair
// type Satisifies the ports.NewPeripheral interface and can be used as an
// argument to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewPaddlePair(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	pdl := &Paddles{
		port: port,
		bus:  bus,
	}

	switch port {
	case plugging.PortLeft:
		// paddle player 0 and 1
		pdl.paddles[0].inptx = chipbus.INPT0
		pdl.paddles[1].inptx = chipbus.INPT1
	case plugging.PortRight:
		// paddle player 2 and 3
		pdl.paddles[0].inptx = chipbus.INPT2
		pdl.paddles[1].inptx = chipbus.INPT3
	}

	// button masks are the same for left and right ports. WriteSWCHx() will
	// shift them into the correct position for the right port
	pdl.paddles[0].buttonMask = 0x80
	pdl.paddles[1].buttonMask = 0x40

	return pdl
}

// Unplug implements the Peripheral interface.
func (pdl *Paddles) Unplug() {
	// no need to go through the paddle pair specific writeSWCHx()
	pdl.bus.WriteSWCHx(pdl.port, paddleNoFire)

	// write no charge value to inptx
	for i := range pdl.paddles {
		pdl.bus.WriteINPTx(pdl.paddles[i].inptx, 0x00)
	}
}

// Snapshot implements the Peripheral interface.
func (pdl *Paddles) Snapshot() ports.Peripheral {
	n := *pdl
	return &n
}

// Plumb implements the ports.Peripheral interface.
func (pdl *Paddles) Plumb(bus ports.PeripheralBus) {
	pdl.bus = bus
}

// String implements the ports.Peripheral interface.
func (pdl *Paddles) String() string {
	var s strings.Builder
	for i := range pdl.paddles {
		s.WriteString(fmt.Sprintf("paddle: button=%v charge=%v resistance=%.02f\n", pdl.paddles[i].fire, pdl.paddles[i].charge, pdl.paddles[i].resistance))
	}
	return s.String()
}

// PortID implements the ports.Peripheral interface.
func (pdl *Paddles) PortID() plugging.PortID {
	return pdl.port
}

// ID implements the ports.Peripheral interface.
func (pdl *Paddles) ID() plugging.PeripheralID {
	return plugging.PeriphPaddles
}

func (pdl *Paddles) setFire() {
	var fire uint8
	for i := range pdl.paddles {
		if pdl.paddles[i].fire {
			fire |= uint8(pdl.paddles[i].buttonMask)
		}
	}
	pdl.bus.WriteSWCHx(pdl.port, ^fire)
}

// HandleEvent implements the ports.Peripheral interface.
func (pdl *Paddles) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.NoEvent:
		return false, nil

	case ports.Fire:
		switch d := data.(type) {
		case bool:
			pdl.paddles[0].fire = d
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return false, fmt.Errorf("paddle: %v: unexpected event data", event)
			}
			pdl.paddles[0].fire = b
		default:
			return false, fmt.Errorf("paddle: %v: unexpected event data", event)
		}

		pdl.setFire()

	case ports.SecondFire:
		switch d := data.(type) {
		case bool:
			pdl.paddles[1].fire = d
		case ports.EventDataPlayback:
			b, err := strconv.ParseBool(string(d))
			if err != nil {
				return false, fmt.Errorf("paddle: %v: unexpected event data", event)
			}
			pdl.paddles[1].fire = b
		default:
			return false, fmt.Errorf("paddle: %v: unexpected event data", event)
		}

		pdl.setFire()

	case ports.PaddleSet:
		switch d := data.(type) {
		case ports.EventDataPaddle:
			for i := range pdl.paddles {
				pdl.paddles[i].resistance = 1.0 - d[i]
			}
		case ports.EventDataPlayback:
			var vals ports.EventDataPaddle
			vals.FromString(string(d))
			for i := range pdl.paddles {
				pdl.paddles[i].resistance = 1.0 - vals[i]
			}
		default:
			return false, fmt.Errorf("paddle: %v: unexpected event data", event)
		}

	default:
		return false, nil
	}

	return true, nil
}

// Update implements the ports.Peripheral interface.
func (pdl *Paddles) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.VBLANK:
		// ground paddle's puck when the high bit of VBLANK is set
		if data.Value&0x80 == 0x80 {
			for i := range pdl.paddles {
				pdl.paddles[i].charge = 0x00
				pdl.paddles[i].ticks = 0.0
				pdl.bus.WriteINPTx(pdl.paddles[i].inptx, 0x00)
			}
		}

	default:
		return true
	}

	return false
}

// Step implements the ports.Peripheral interface.
func (pdl *Paddles) Step() {
	for i := range pdl.paddles {
		if pdl.paddles[i].charge < 255 {
			pdl.paddles[i].ticks += paddleChargeRate
			if pdl.paddles[i].ticks >= pdl.paddles[i].resistance {
				pdl.paddles[i].ticks = 0.0
				pdl.paddles[i].charge++
				pdl.bus.WriteINPTx(pdl.paddles[i].inptx, pdl.paddles[i].charge)
			}
		}

	}

	pdl.setFire()
}

// Reset implements the ports.Peripheral interface.
func (pdl *Paddles) Reset() {
	for i := range pdl.paddles {
		pdl.paddles[i].charge = 0
		pdl.paddles[i].ticks = 0.0
		pdl.paddles[i].resistance = 0.0
	}
}

// IsActive implements the ports.Peripheral interface.
func (pdl *Paddles) IsActive() bool {
	return false
}
