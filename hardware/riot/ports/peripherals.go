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

package ports

import (
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Peripheral represents a (input or output) device that can plugged into the
// ports of the VCS.
type Peripheral interface {
	// String should return information about the state of the peripheral
	String() string

	// Periperhal is to be removed
	Unplug()

	// Snapshot the instance of the Peripheral
	Snapshot() Peripheral

	// Plumb a new PeripheralBus into the Peripheral
	Plumb(PeripheralBus)

	// The port the peripheral is plugged into
	PortID() plugging.PortID

	// The ID of the peripheral being represented
	ID() plugging.PeripheralID

	// handle an incoming input event
	HandleEvent(Event, EventData) (bool, error)

	// memory has been updated. peripherals are notified.
	Update(chipbus.ChangedRegister) bool

	// step is called every CPU clock. important for paddle devices
	Step()

	// reset state of peripheral. this has nothing to do with the reset switch
	// on the VCS panel
	Reset()

	// whether the peripheral is currently "active"
	IsActive() bool
}

// RestartPeripheral is implemented by peripherals that can significantly
// change configuration. For example, the AtariVox can make use of an external
// program which might be changed during the emulation.
//
// Restarting is a special event and should not be called too often due to the
// possible nature of configuration changes.
type RestartPeripheral interface {
	Restart()
}

// DisablePeripheral is implemented by peripherals that can be disabled. This
// is useful for peripherals that do not act well during rewinding.
type DisablePeripheral interface {
	Disable(bool)
}

// NewPeripheral defines the function signature for a creating a new
// peripheral, suitable for use with AttachPloyer0() and AttachPlayer1().
type NewPeripheral func(*instance.Instance, plugging.PortID, PeripheralBus) Peripheral

// PeripheralBus defines the memory operations required by peripherals. We keep
// this bus definition here rather than the Bus package because it is very
// specific to this package and sub-packages.
type PeripheralBus interface {
	WriteINPTx(inptx chipbus.Register, data uint8)

	// the SWCHA register is logically divided into two nibbles. player 0
	// uses the upper nibble and player 1 uses the lower nibble. peripherals
	// attached to either player port *must* only use the upper nibble. this
	// write function will transparently shift the data into the lower nibble
	// for peripherals attached to the player 1 port.
	//
	// also note that peripherals do not need to worry about preserving bits
	// in the opposite nibble. the WriteSWCHx implementation will do that
	// transparently according to which port the peripheral is attached
	//
	// Peripherals attached to the panel port can use the entire byte of the
	// SWCHB register
	WriteSWCHx(id plugging.PortID, data uint8)
}
