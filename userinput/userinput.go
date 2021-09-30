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

package userinput

import (
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// HandleInput conceptualises data being sent to the console ports.
type HandleInput interface {
	// HandleEvent forwards the Event and EventData to the device connected to the
	// specified PortID.
	HandleEvent(id plugging.PortID, ev ports.Event, d ports.EventData) error

	// PeripheralID identifies the device currently attached to the port.
	PeripheralID(id plugging.PortID) plugging.PeripheralID
}
