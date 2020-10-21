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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// Auto handles the automatic switching between controller types.
type Auto struct {
	id         ports.PortID
	bus        ports.PeripheralBus
	controller ports.Peripheral

	paddleTouchLeft  int
	paddleTouchRight int
}

// NewAuto is the preferred method of initialisation for the Auto type.
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewAuto(id ports.PortID, bus ports.PeripheralBus) ports.Peripheral {
	aut := &Auto{
		id:  id,
		bus: bus,
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
		aut.toStick()

	case ports.PaddleFire:

	case ports.PaddleSet:
		// count the number of times the paddle controller has touched the
		// extremes (or near the extremes). this is really to prevent the
		// paddle from accidentally be triggered. there maybe should be some
		// time limit
		if _, ok := aut.controller.(*Paddle); !ok {
			v := data.(float32)
			if v < 0.1 {
				aut.paddleTouchLeft++
			} else if v > 0.9 {
				aut.paddleTouchRight++
			}
			if aut.paddleTouchLeft >= 3 && aut.paddleTouchRight >= 3 {
				aut.toPaddle()
				aut.paddleTouchLeft = 0
				aut.paddleTouchRight = 0
			}
		}

	case ports.KeyboardDown:
		aut.toKeyboard()
	case ports.KeyboardUp:
		aut.toKeyboard()
	}

	err := aut.controller.HandleEvent(event, data)

	// if error was because of an unhandled event then return without error
	if err != nil && curated.Is(err, UnhandledEvent) {
		return nil
	}

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
	aut.controller = NewStick(aut.id, aut.bus)
}

func (aut *Auto) toStick() {
	if _, ok := aut.controller.(*Stick); !ok {
		aut.controller = NewStick(aut.id, aut.bus)
	}
}

func (aut *Auto) toPaddle() {
	if _, ok := aut.controller.(*Paddle); !ok {
		aut.controller = NewPaddle(aut.id, aut.bus)
	}
}

func (aut *Auto) toKeyboard() {
	if _, ok := aut.controller.(*Keyboard); !ok {
		aut.controller = NewKeyboard(aut.id, aut.bus)
	}
}
