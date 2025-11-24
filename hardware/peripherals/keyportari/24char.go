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

package keyportari

import (
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

type Keyportari24char struct {
	keyportari
}

func NewKeyportari24char(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	return &Keyportari24char{
		keyportari: newKeyportari(env, port, bus),
	}
}

func (kp *Keyportari24char) Snapshot() ports.Peripheral {
	n := *kp
	if kp.periph != nil {
		n.periph = kp.periph.Snapshot()
	}
	return &n
}

func (kp *Keyportari24char) Protocol() string {
	return "24char"
}

func (kp *Keyportari24char) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.KeyportariUp:
		kp.keydown = false

		d := data.(ports.EventDataKeyportari)
		switch d.Key {
		case "Up":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Up, ports.DataStickFalse)
			}
		case "Down":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Down, ports.DataStickFalse)
			}
		case "Left":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Left, ports.DataStickFalse)
			}
		case "Right":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Right, ports.DataStickFalse)
			}
		}

		kp.bus.WriteSWCHx(kp.port, 0xf0)
		return true, nil

	case ports.KeyportariDown:
		kp.keydown = true

		var v uint8
		d := data.(ports.EventDataKeyportari)
		switch d.Key {
		case ",":
			v = 0x00
		case ".":
			v = 0x01
		case "Backspace", "Delete":
			v = 0x02
		case "Return":
			v = 0x03
		case "Up":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Up, ports.DataStickTrue)
			}
			return true, nil
		case "Down":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Down, ports.DataStickTrue)
			}
			return true, nil
		case "Left":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Left, ports.DataStickTrue)
			}
			return true, nil
		case "Right":
			if kp.periph != nil {
				return kp.periph.HandleEvent(ports.Right, ports.DataStickTrue)
			}
			return true, nil
		default:
			// default is space
			v = 0x04

			if len(d.Key) == 1 {
				c := d.Key[0]
				if c > 64 && c < 91 {
					// upper chase characters (A-Z)
					v = (c - 63) * 4
				} else if c > 96 && c < 123 {
					// lower chase characters (a-z)
					v = (c - 69) * 4
				} else if c > 47 && c < 58 {
					// digits (0-9)
					v = (c + 6) * 4
				}
			}
		}

		switch kp.port {
		case plugging.PortLeft:
			kp.bus.WriteSWCHx(plugging.PortLeft, v&0xf0)
		case plugging.PortRight:
			kp.bus.WriteSWCHx(plugging.PortRight, v<<4)
		}

		return true, nil
	default:
		kp.keyportari.HandleEvent(event, data)
	}

	return false, nil
}
