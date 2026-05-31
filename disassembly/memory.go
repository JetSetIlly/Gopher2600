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

package disassembly

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper/banking"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// disasmMemory is a simplified memory model that allows the emulated CPU to
// read cartridge memory without touching the actual cartridge.
type disasmMemory struct {
	// the bank which the cartridge starts on
	startingBank int

	// current bank to index the banks array
	currentBank int
	banks       []banking.Content

	// the current origin for the mapped bank
	currentOrigin uint16

	// the relevant bits in a cartridge address. used to mask away mirror address information
	cartridgeBits uint16
}

func newDisasmMemory(startingBank int, copiedBanks []banking.Content, cartridgeBits uint16) *disasmMemory {
	dismem := &disasmMemory{
		startingBank:  startingBank,
		currentBank:   startingBank,
		banks:         copiedBanks,
		cartridgeBits: cartridgeBits,
	}
	return dismem
}

func (dismem *disasmMemory) Read(address uint16) (uint8, error) {
	// map address
	address, area := memorymap.MapAddress(address, true)
	if area == memorymap.Cartridge {
		address = (address - dismem.currentOrigin) & dismem.cartridgeBits
		if int(address) >= len(dismem.banks[dismem.currentBank].Data) {
			return 0, nil
		}
		return dismem.banks[dismem.currentBank].Data[address], nil
	}

	// address outside of cartridge range return nothing
	return 0, nil
}

func (dismem *disasmMemory) Write(address uint16, data uint8) error {
	return nil
}
