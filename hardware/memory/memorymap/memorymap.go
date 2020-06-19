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

import "fmt"

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

// Cartridge memory is mirrored in a number of places in the address space. The
// most useful mirror is the Fxxx mirror which many programmers use when
// writing assembly programs. The following constants are used by the
// disassembly package to reference the disassembly to the Fxxx mirror.
//
// Be extra careful when looping with MemtopCartFxxxMirror because it is at the
// very edge of uint16. Limit detection may need to consider the overflow
// conditions.
const (
	OriginCartFxxxMirror = uint16(0xf000)
	MemtopCartFxxxMirror = uint16(0xffff)
)

// Memtop is the top most address of memory in the VCS. It is the same as the
// cartridge memtop.
const Memtop = uint16(0x1fff)

// Within RIOT and TIA mirrors there are smaller mirrors (for the want of a
// better phrase). MaskRIOT and MaskTIA keep only the relevent bits of a RIOT
// and TIA address. Should only be applied to addresses that are definately
// either a RIOT or TIA address.
const (
	MaskRIOT = uint16(0x02f7)
	MaskTIA  = uint16(0x000f)
)

// CartridgeBits identifies the bits in an address that are relevent to the
// cartridge address. Useful for discounting those bits that determine the
// cartridge mirror. For example, the following will be true:
//
//	0x1123 & CartridgeBits == 0xf123 & CartridgeBits
//
// Alternatively, the following is an effective way to index an array:
//
//  addr := 0xf000
//  mem[addr & CartridgeBits] = 0xff
//
// In the example, index zero of the mem array is assigned the value 0xff.
const (
	CartridgeBits = OriginCart ^ MemtopCart
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
			return address & MemtopRIOT & MaskRIOT, RIOT
		}
		return address & MemtopRIOT, RIOT
	}

	// RAM addresses
	if address&OriginRAM == OriginRAM {
		return address & MemtopRAM, RAM
	}

	// everything else is in TIA space
	if read {
		return address & MemtopTIA & MaskTIA, TIA
	}

	return address & MemtopTIA, TIA
}

// IsArea returns true if the address is in the specificied area
func IsArea(address uint16, area Area) bool {
	_, a := MapAddress(address, true)
	return area == a
}

// BankDetails is used to identify a cartridge bank. In some contexts bank is
// represented by an integer only. The Bank type is used when more information
// about a bank is required.
type BankDetails struct {
	Number  int
	IsRAM   bool
	NonCart bool
	Segment int
}

func (b BankDetails) String() string {
	if b.NonCart {
		return "-"
	}
	if b.IsRAM {
		return fmt.Sprintf("%dR", b.Number)
	}
	return fmt.Sprintf("%d", b.Number)
}
