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

// Package mapper contains the CartMapper interface. This interface abstracts
// the functions required of any cartridge format.
//
// In addition it defines other interfaces that a cartridge mapper may
// optionally implement for additional functionality for the rest of the
// emulation. For example, the CartHotspotBus interface can be used to reveal
// information about the special/hotspot addresses for a cartridge format.
//
// In addition to the interfaces, any additional types are defined. For
// instance, the CartHotspotInfo type the symbol name and action type for a
// every hotspot in the cartridge.
package mapper
