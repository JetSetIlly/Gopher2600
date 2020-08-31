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
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// PortID differentiates the different ports peripherals can be attached to
type PortID int

// List of defined IDs
const (
	Player0ID PortID = iota
	Player1ID
	PanelID
	NumPortIDs
)

// Peripheral represents a (input or output) device that can attached to the
// VCS ports.
type Peripheral interface {
	String() string

	// ID is the name of the peripheral
	ID() string

	// handle an incoming input event
	HandleEvent(Event, EventData) error

	// memory has been updated. peripherals are notified.
	Update(bus.ChipData) bool

	// step is called every CPU clock. important for paddle devices
	Step()

	// reset state of peripheral. this has nothing to do with the reset switch
	// on the VCS panel
	Reset()
}

// NewPeripheral defines the function signature for a creating a new
// peripheral, suitable for use with AttachPloyer0() and AttachPlayer1()
type NewPeripheral func(PortID, MemoryAccess) Peripheral

// MemoryAccess defines the memory operations required by peripherals
type MemoryAccess interface {
	WriteSWCHx(id PortID, data uint8)
	WriteINPTx(inptx addresses.ChipRegister, data uint8)
}
