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
	"unicode"
	"unicode/utf8"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/controllers"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

type keyportari struct {
	env     *environment.Environment
	port    plugging.PortID
	bus     ports.PeripheralBus
	periph  ports.Peripheral
	keydown bool
}

func newKeyportari(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) keyportari {
	kp := keyportari{
		env:  env,
		port: port,
		bus:  bus,
	}
	if port == plugging.PortLeft {
		kp.periph = controllers.NewStick(env, port, bus)
	}
	return kp
}

// Plug implements plugging.PeripheralShim
func (kp *keyportari) Plug(periph ports.Peripheral) {
	if kp.periph != nil {
		kp.Unplug()
	}
	kp.periph = periph
}

// Child implements plugging.PeripheralShim
func (kp *keyportari) Periph() ports.Peripheral {
	return kp.periph
}

// ShimID implements plugging.PeripheralShim
func (kp *keyportari) ShimID() plugging.PeripheralID {
	return plugging.PeriphKeyportari
}

func (kp *keyportari) String() string {
	if kp.periph != nil {
		return kp.periph.String()
	}
	return "keyportari"
}

func (kp *keyportari) Unplug() {
	if kp.periph != nil {
		kp.periph.Unplug()
	}
	kp.bus.WriteSWCHx(kp.port, 0xf0)
}

// keyportari does not have a Snapshot() function because it doesn't work well when it is embedded.
// a snapshot would just take a snapshot of the embedded type when want we want is a snapshot of the
// containing type too

func (kp *keyportari) Plumb(bus ports.PeripheralBus) {
	kp.bus = bus
	if kp.periph != nil {
		kp.periph.Plumb(bus)
	}
}

func (kp *keyportari) PortID() plugging.PortID {
	return kp.port
}

func (kp *keyportari) ID() plugging.PeripheralID {
	if kp.periph == nil {
		return plugging.PeriphNone
	}
	return kp.periph.ID()
}

func (kp *keyportari) HandleEvent(event ports.Event, data ports.EventData) (bool, error) {
	if kp.periph != nil {
		return kp.periph.HandleEvent(event, data)
	}
	return false, nil
}

func (kp *keyportari) Update(data chipbus.ChangedRegister) bool {
	if kp.periph != nil {
		return kp.periph.Update(data)
	}
	return false
}

func (kp *keyportari) Step() {
	if kp.periph != nil {
		kp.periph.Step()
	}
}

func (kp *keyportari) Reset() {
	if kp.periph != nil {
		kp.periph.Reset()
	}
	kp.keydown = false
	kp.bus.WriteSWCHx(kp.port, 0xf0)
}

func (kp *keyportari) IsActive() bool {
	if kp.periph != nil {
		return kp.keydown || kp.periph.IsActive()
	}
	return kp.keydown
}

func (kp *keyportari) isPrint(key string) (rune, bool) {
	if len(key) != 1 {
		return ' ', false
	}
	r, sz := utf8.DecodeRuneInString(key)
	if sz > 1 {
		return ' ', false
	}
	return r, unicode.IsPrint(r)
}

func (kp *keyportari) writeSWCHx(v uint8) {
	switch kp.port {
	case plugging.PortLeft:
		kp.bus.WriteSWCHx(plugging.PortLeft, v&0xf0)
	case plugging.PortRight:
		kp.bus.WriteSWCHx(plugging.PortRight, v<<4)
	}
}
