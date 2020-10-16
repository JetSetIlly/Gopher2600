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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// disasmMemory is a simplified memory model that allows the emulated CPU to
// read cartridge memory without touching the actual cartridge.
type disasmMemory struct {
	// if bank is not nil then the bank is read directly
	bank   mapper.BankContent
	origin uint16
}

func (dismem *disasmMemory) Read(address uint16) (uint8, error) {
	// map address
	address, area := memorymap.MapAddress(address, true)
	if area == memorymap.Cartridge {
		// bank field is set so we bypass the cartridge mapper's usual read
		// logic and access the bank directly
		address = (address - dismem.origin) & memorymap.CartridgeBits
		if address >= uint16(len(dismem.bank.Data)) {
			return 0, nil
		}
		return dismem.bank.Data[address], nil
	}

	// address outside of cartridge range return nothing
	return 0, nil
}

func (dismem *disasmMemory) ReadZeroPage(address uint8) (uint8, error) {
	return dismem.Read(uint16(address))
}

func (dismem *disasmMemory) Write(address uint16, data uint8) error {
	return nil
}
