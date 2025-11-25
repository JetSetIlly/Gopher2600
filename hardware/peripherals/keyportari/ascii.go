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
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

type KeyportariASCII struct {
	keyportari
}

func NewKeyportariASCII(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	return &KeyportariASCII{
		keyportari: newKeyportari(env, port, bus),
	}
}

func (kp *KeyportariASCII) Snapshot() ports.Peripheral {
	n := *kp
	if kp.periph != nil {
		n.periph = kp.periph.Snapshot()
	}
	return &n
}

func (kp *KeyportariASCII) Protocol() string {
	return "ASCII"
}

func (kp *KeyportariASCII) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.KeyportariUp:
		kp.keydown = false
		kp.bus.WriteSWCHx(kp.port, 0xf0)
		return true, nil
	case ports.KeyportariDown:
		kp.keydown = true
		var v uint8
		d := data.(ports.EventDataKeyportari)
		switch d.Key {
		case "Return":
			v = 0x0a
		case "Backspace":
			v = 0x7f
		case "Space":
			v = 0x20
		case "Left Shift", "Right Shift":
			return true, nil
		case "Left Ctrl", "Right Ctrl":
			return true, nil
		case "Left Alt", "Right Alt":
			return true, nil
		case "Escape":
			return true, nil
		default:
			if len(d.Key) == 0 {
				return false, nil
			}

			v = strings.ToLower(d.Key)[0]
			if d.Shift {
				switch v {
				case '2':
					v = '"'
				default:
					v = strings.ToUpper(string(v))[0]
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
