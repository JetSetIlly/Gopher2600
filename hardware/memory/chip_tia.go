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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package memory

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// newTIA is the preferred method of initialisation for the TIA memory area
func newTIA() *ChipMemory {
	area := &ChipMemory{
		origin:      memorymap.OriginTIA,
		memtop:      memorymap.MemtopTIA,
		cpuReadMask: memorymap.AddressMaskTIA,
	}

	// allocation the minimal amount of memory
	area.memory = make([]uint8, area.memtop-area.origin+1)

	// initial values
	area.memory[addresses.INPT1] = 0x00
	area.memory[addresses.INPT2] = 0x00
	area.memory[addresses.INPT3] = 0x00
	area.memory[addresses.INPT4] = 0x80
	area.memory[addresses.INPT5] = 0x80

	return area
}
