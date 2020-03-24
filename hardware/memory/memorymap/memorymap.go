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

package memorymap

// Area represents the different areas of memory
type Area int

func (a Area) String() string {
	switch a {
	case TIA:
		return "TIA"
	case RAM:
		return "RAM"
	case RIOT:
		return "RIOT"
	case Cartridge:
		return "Cartridge"
	}

	return "undefined"
}

// The different memory areas in the VCS
const (
	Undefined Area = iota
	TIA
	RAM
	RIOT
	Cartridge
)

// The origin and memory top for each area of memory. Checking which area an
// address falls within and forcing the address into the normalised range is
// all handled by the MapAddress() function.
//
// Implementations of the different memory areas may need to drag the address
// down into the the range of an array. This can be done by with elegantly with
// (address^origin) rather than subtraction.
const (
	OriginTIA  = uint16(0x0000)
	MemtopTIA  = uint16(0x003f)
	OriginRAM  = uint16(0x0080)
	MemtopRAM  = uint16(0x00ff)
	OriginRIOT = uint16(0x0280)
	MemtopRIOT = uint16(0x0297)
	OriginCart = uint16(0x1000)
	MemtopCart = uint16(0x1fff)
)

// Memtop is the top most address of memory in the VCS. It is the same as the
// cartridge memtop.
const Memtop = uint16(0x1fff)

// Adressess in the RIOT and TIA areas that are being used to read from from
// memory require an additional transformation
const (
	AddressMaskRIOT = uint16(0x02f7)
	AddressMaskTIA  = uint16(0x000f)
)

// The top nibble of a cartridge address can be anything. AddressMaskCart takes
// away the uninteresting bits
//
// Good way of normalising cartridge addresses to start from 0x000 (useful for
// arrays) *if* you know that the address is for certain a cartridge address
const (
	AddressMaskCart = MemtopCart ^ OriginCart
)

// MapAddress translates the address argument from mirror space to primary
// space.  Generally, an address should be passed through this function before
// accessing memory.
func MapAddress(address uint16, read bool) (uint16, Area) {
	// note that the order of these filters is important

	// cartridge addresses
	if address&OriginCart == OriginCart {
		return address & MemtopCart, Cartridge
	}

	// RIOT addresses
	if address&OriginRIOT == OriginRIOT {
		if read {
			return address & MemtopRIOT & AddressMaskRIOT, RIOT
		}
		return address & MemtopRIOT, RIOT
	}

	// RAM addresses
	if address&OriginRAM == OriginRAM {
		return address & MemtopRAM, RAM
	}

	// everything else is in TIA space
	if read {
		return address & MemtopTIA & AddressMaskTIA, TIA
	}

	return address & MemtopTIA, TIA
}

func IsArea(address uint16, area Area) bool {
	_, a := MapAddress(address, true)
	return area == a
}
