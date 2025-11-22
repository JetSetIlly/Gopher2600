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
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

type Keyportari struct {
	port plugging.PortID
	bus  ports.PeripheralBus
}

func NewKeyportari(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	kp := &Keyportari{
		port: port,
		bus:  bus,
	}
	return kp
}

func (kp *Keyportari) String() string {
	return "keyportari: nothing to report"
}

func (kp *Keyportari) Unplug() {
	kp.bus.WriteSWCHx(kp.port, 0xf0)
}

func (kp *Keyportari) Snapshot() ports.Peripheral {
	n := *kp
	return &n
}

func (kp *Keyportari) Plumb(bus ports.PeripheralBus) {
	kp.bus = bus
}

func (kp *Keyportari) PortID() plugging.PortID {
	return kp.port
}

func (kp *Keyportari) ID() plugging.PeripheralID {
	return plugging.PeriphKeyportari
}

func (kp *Keyportari) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	switch event {
	case ports.KeyportariUp:
		kp.bus.WriteSWCHx(kp.port, 0xf0)
		return true, nil
	case ports.KeyportariDown:
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
	}

	return false, nil
}

func (kp *Keyportari) Update(chipbus.ChangedRegister) bool {
	return false
}

func (kp *Keyportari) Step() {
}

func (kp *Keyportari) Reset() {
	kp.bus.WriteSWCHx(kp.port, 0xf0)
}

func (kp *Keyportari) IsActive() bool {
	return true
}
