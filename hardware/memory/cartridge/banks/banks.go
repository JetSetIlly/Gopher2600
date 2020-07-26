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

package banks

import "fmt"

// Content contains data and ID of a cartridge bank. Used by IterateBanks()
// and helps the disassembly process.
type Content struct {
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
	Origins []uint16
}

// Details is used to identify a cartridge bank. In some contexts bank is
// represented by an integer only. The Bank type is used when more information
// about a bank is required.
type Details struct {
	Number  int
	IsRAM   bool
	NonCart bool
	Segment int
}

func (b Details) String() string {
	if b.NonCart {
		return "-"
	}
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}
