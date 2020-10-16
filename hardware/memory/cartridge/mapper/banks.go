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

package mapper

import (
	"fmt"
)

// BankContent contains data and ID of a cartridge bank. Used by CopyBanks()
// and helps the disassembly process.
type BankContent struct {
	Number int

	// copy of the bank data
	Data []uint8

	// the segment origins that this data is allowed to be mapped to. most
	// cartridges will have one entry. values in the array will refer to
	// addresses in the cartridge address space. by convention the mappers will
	// refer to the primary mirror.
	//
	//	memorymap.OriginCart <= origins[n] <= memorymap.MemtopCart
	//
	// to index the Data field, transform the origin and any address derived
	// from it, with memorymap.CartridgeBits
	//
	//	idx := Origins[0] & memorymap.CartridgeBits
	//	v := Data[idx]
	//
	// address values are supplied by the mapper implementation and must be
	// cartridge addresses and should in the primary cartridge mirror range
	// (ie. 0x1000 to 0x1fff)j
	Origins []uint16
}

// BankInfo is used to identify a cartridge bank. In some instance a bank can
// be identified by it's bank number only. In other contexts more detail is
// required and so BankInfo is used isntead.
type BankInfo struct {
	Number  int
	Segment int

	// is cartridge bank writable
	IsRAM bool

	// if the address used to generate the Details is not a cartridge address.
	// this happens deliberately for example, during the Supercharger load
	// procedure, where execution happens (briefly) inside the main VCS RAM
	NonCart bool
}

func (b BankInfo) String() string {
	if b.NonCart {
		return "-"
	}
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}
