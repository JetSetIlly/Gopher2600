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
		kp.writeSWCHx(0xff)
		return true, nil

	case ports.KeyportariDown:
		kp.keydown = true
		var v uint8
		d := data.(ports.EventDataKeyportari)
		switch d.Key {
		case "return":
			v = 0x0a
		case "backspace":
			v = 0x7f
		case "space":
			v = 0x20
		default:
			return true, nil
		}
		kp.writeSWCHx(v)
		return true, nil

	case ports.KeyportariText:
		d := data.(ports.EventDataKeyportari)
		if r, ok := kp.isPrint(d.Key); ok {
			kp.writeSWCHx(uint8(r))
		}
		return true, nil

	default:
		kp.keyportari.HandleEvent(event, data)
	}

	return false, nil
}
