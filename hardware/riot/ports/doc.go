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

// Package ports represents the input/output parts of the VCS (the IO in
// RIOT).
//
// Emulated peripherals are plugged into the VCS with the Plug() function.
// Input from "real" devices is handled by HandleEvent() which passes the event
// to peripherals in the specified PortID.
//
// Peripherals write back to the VCS through the PeripheralBus.
package ports
