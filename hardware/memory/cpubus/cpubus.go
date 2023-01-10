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

package cpubus

// Memory defines the operations for the memory system when accessed from the CPU All memory areas
// implement this interface because they are all accessible from the CPU (compare to ChipBus). The
// VCSMemory type also implements this interface and maps the read/write address to the correct
// memory area -- meaning that CPU access need not care which part of memory it is writing to
//
// Addresses should be mapped to their primary mirror in all cases.
//
// In the case of cartridge implementations there should be no real distinction between Read and
// Write. This is because there is no R/W line to the cartridge. In the event that a cartridge has a
// RAM area, writing to the RAM is usually done through specific addreses.
type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}
