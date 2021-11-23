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
// Emulations can share user input through the DrivenEvent mechanism. The driver
// emulation should call SynchroniseWithPassenger() and the passenger emulation
// should call SynchroniseWithDriver().
//
// With the DrivenEvent mechanism, the driver sends events to the passenger.
// Both emulations will receive the same user input at the same time, relative
// to television coordinates, so it is important that the driver is running
// ahead of the passenger at all time. See comparison package for model
// implementation.
package ports
