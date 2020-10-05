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

// Package bus is used to define access patterns for different areas of the
// emulation to the VCS memory. For example, the VCS chips (the TIA and the
// RIOT) access memory differently to the CPU. By restricting access to memory
// from the chip to the ChipBus interface, we can prevent
//
// The DebugBus is for the exclusive use of debuggers and exposes a Peek() and
// Poke() function.
package bus
