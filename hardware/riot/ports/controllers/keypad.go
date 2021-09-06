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

// Keypad represents the VCS keypad type.
type Keypad struct {
	port   plugging.PortID
	bus    ports.PeripheralBus
	column [3]addresses.ChipRegister
	key    rune
}

// the value of keypad.key when nothing is being pressed.
const noKey = ' '

// NewKeypad is the preferred method of initialisation for the Keyboard type
// Satisifies the ports.NewPeripheral interface and can be used as an argument
// to ports.AttachPlayer0() and ports.AttachPlayer1().
func NewKeypad(port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	key := &Keypad{
		port: port,
		bus:  bus,
	}

	switch port {
	case plugging.PortLeftPlayer:
		key.column = [3]addresses.ChipRegister{addresses.INPT0, addresses.INPT1, addresses.INPT4}
	case plugging.PortRightPlayer:
		key.column = [3]addresses.ChipRegister{addresses.INPT2, addresses.INPT3, addresses.INPT5}
	}

	key.Reset()
	return key
}

// Plumb implements the ports.Peripheral interface.
func (key *Keypad) Plumb(bus ports.PeripheralBus) {
	key.bus = bus
}

// String implements the ports.Peripheral interface.
func (key *Keypad) String() string {
	return fmt.Sprintf("keypad: key=%v", key.key)
}

// PortID implements the ports.Peripheral interface.
func (key *Keypad) PortID() plugging.PortID {
	return key.port
}

// ID implements the ports.Peripheral interface.
func (key *Keypad) ID() plugging.PeripheralID {
	return plugging.PeriphKeypad
}

// HandleEvent implements the ports.Peripheral interface.
func (key *Keypad) HandleEvent(event ports.Event, data ports.EventData) error {
	switch event {
	case ports.NoEvent:
		return nil

	case ports.KeypadDown:
		var k rune

		switch d := data.(type) {
		case rune:
			k = d
		case ports.EventDataPlayback:
			n, err := strconv.ParseInt(string(d), 10, 64)
			if err != nil {
				return curated.Errorf("keypad: %v: unexpected event data", event)
			}
			k = rune(n)

		default:
			return curated.Errorf("keypad: %v: unexpected event data", event)
		}

		if k != '1' && k != '2' && k != '3' &&
			k != '4' && k != '5' && k != '6' &&
			k != '7' && k != '8' && k != '9' &&
			k != '*' && k != '0' && k != '#' {
			return curated.Errorf("keypad: unrecognised rune (%v)", k)
		}

		// note key for use by readKeyboard()
		key.key = k

	case ports.KeypadUp:
		switch d := data.(type) {
		case nil:
			// expected data
		case ports.EventDataPlayback:
			if len(string(d)) > 0 {
				return curated.Errorf("keypad: %v: unexpected event data", event)
			}
		}
		key.key = noKey

	default:
		// silently ignore unhandled event
		return nil
	}

	return nil
}

// Update implements the ports.Peripheral interface.
func (key *Keypad) Update(data bus.ChipData) bool {
	switch data.Name {
	case "SWCHA":
		var column int
		var v uint8

		switch key.port {
		case plugging.PortLeftPlayer:
			v = data.Value & 0xf0
		case plugging.PortRightPlayer:
			v = (data.Value & 0x0f) << 4
		}

		switch key.key {
		// row 0
		case '1':
			if v&0xe0 == v {
				column = 1
			}
		case '2':
			if v&0xe0 == v {
				column = 2
			}
		case '3':
			if v&0xe0 == v {
				column = 3
			}

			// row 2
		case '4':
			if v&0xd0 == v {
				column = 1
			}
		case '5':
			if v&0xd0 == v {
				column = 2
			}
		case '6':
			if v&0xd0 == v {
				column = 3
			}

			// row 3
		case '7':
			if v&0xb0 == v {
				column = 1
			}
		case '8':
			if v&0xb0 == v {
				column = 2
			}
		case '9':
			if v&0xb0 == v {
				column = 3
			}

			// row 4
		case '*':
			if v&0x70 == v {
				column = 1
			}
		case '0':
			if v&0x70 == v {
				column = 2
			}
		case '#':
			if v&0x70 == v {
				column = 3
			}
		}

		// The Stella Programmer's Guide says that: "a delay of 400 microseconds is
		// necessary between writing to this port and reading the TIA input ports.".
		// We're not emulating this here because as far as I can tell there is no need
		// to. More over, I'm not sure what's supposed to happen if the 400ms is not
		// adhered to.
		//
		// !!TODO: Consider adding 400ms delay for SWACNT settings to take effect.
		switch column {
		case 1:
			key.bus.WriteINPTx(key.column[0], 0x00)
			key.bus.WriteINPTx(key.column[1], 0x80)
			key.bus.WriteINPTx(key.column[2], 0x80)
		case 2:
			key.bus.WriteINPTx(key.column[0], 0x80)
			key.bus.WriteINPTx(key.column[1], 0x00)
			key.bus.WriteINPTx(key.column[2], 0x80)
		case 3:
			key.bus.WriteINPTx(key.column[0], 0x80)
			key.bus.WriteINPTx(key.column[1], 0x80)
			key.bus.WriteINPTx(key.column[2], 0x00)
		default:
			key.bus.WriteINPTx(key.column[0], 0x80)
			key.bus.WriteINPTx(key.column[1], 0x80)
			key.bus.WriteINPTx(key.column[2], 0x80)
		}
	}

	return false
}

// Step implements the ports.Peripheral interface.
func (key *Keypad) Step() {
	// keypad does not write to SWCHx so unlike the Stick and Paddle
	// controller types there is no need to ensure the SWCHx register retains
	// its state if it is active
}

// Reset implements the ports.Peripheral interface.
func (key *Keypad) Reset() {
	key.key = noKey
}