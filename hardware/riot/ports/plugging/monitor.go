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

package plugging

// PlugMonitor interface implementations will be notified of newly plugged
// peripherals.
type PlugMonitor interface {
	Plugged(port PortID, description string)
}

// Moniterable implementations are capable of having a PlugMonitor attached to
// it. The VCS Ports themselves are monitorable but we also use this mechanism
// in the "auto" controller type and in the future, devices like the Quadtari
// will be implemented similarly.
//
// It is expected that such implementations will call PlugMonitor.Plugged() as
// required. Note that PlugMonitor.Plugged() should not be called on the event
// of AttachPlugMonitor except in the special case of the ports.Ports type
//
// Implementations of Monitorable should also test peripherals that are
// daisy-chained and call AttachPlusMonitor() as appropriate.
//
//		if a, ok := periph.daisychain.(pluggin.Monitorable); ok {
//			a.AttachPlugMonitor(m)
//		}
//
type Monitorable interface {
	AttachPlugMonitor(m PlugMonitor)
}
