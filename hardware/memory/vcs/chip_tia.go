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

package vcs

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
)

// NewTIA is the preferred method of initialisation for the TIA memory chip.
func NewTIA(prefs *preferences.Preferences) *ChipMemory {
	chip := &ChipMemory{
		prefs:  prefs,
		origin: memorymap.OriginTIA,
		memtop: memorymap.MemtopTIA,
	}

	// allocate the minimal amount of memory
	chip.memory = make([]uint8, chip.memtop-chip.origin+1)

	// initial values
	chip.memory[addresses.INPT1] = 0x00
	chip.memory[addresses.INPT2] = 0x00
	chip.memory[addresses.INPT3] = 0x00
	chip.memory[addresses.INPT4] = 0x80
	chip.memory[addresses.INPT5] = 0x80

	return chip
}
