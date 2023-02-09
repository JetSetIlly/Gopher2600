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

package memorymap

// Area represents the different areas of memory.
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

// The different memory areas in the VCS.
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
// down further into the the range of the memory array. This is best done with
// (address^origin) rather than subtraction.
const (
	OriginTIA = uint16(0x0000)
	MemtopTIA = uint16(0x003f)
	OriginRAM = uint16(0x0080)
	MemtopRAM = uint16(0x00ff)

	OriginRIOT = uint16(0x0280)
	MemtopRIOT = uint16(0x0297)

	MemtopChipRegisters = MemtopRIOT

	OriginCart     = uint16(0x1000)
	MemtopCart     = uint16(0x1fff)
	OriginAbsolute = uint16(0x0000)
	MemtopAbsolute = uint16(0xffff)
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

// CartridgeBits identifies the bits in an address that are relevant to the
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

// The masks to apply to an address to bring any address into the primary
// range. Prefer to use MapAddress() for ease of use.
const (
	maskCart = MemtopCart
	maskRAM  = MemtopRAM

	maskWriteTIA = MemtopTIA
	maskReadTIA  = uint16(0x000f)

	maskWriteRIOT = MemtopRIOT
	maskReadRIOT  = uint16(0x0287)

	// read registers specific to the RIOT timer start at 0x284. the two bit of
	// the address is masked out leaving 4 addresses mapping to just two
	// registers - INTIM and TIMINT
	maskReadRIOT_timer            = uint16(0x284)
	maskReadRIOT_timer_correction = uint16(0x285)
)

// MapAddress translates the address argument from mirror space to primary
// space.  Generally, an address should be passed through this function before
// accessing memory.
func MapAddress(address uint16, read bool) (uint16, Area) {
	// note that the order of these filters is important

	// cartridge addresses
	if address&OriginCart == OriginCart {
		return address & maskCart, Cartridge
	}

	// RIOT addresses
	if address&OriginRIOT == OriginRIOT {
		if read {
			if address&maskReadRIOT_timer == maskReadRIOT_timer {
				// additional masking for RIOT timer addresses
				return address & maskReadRIOT_timer_correction, RIOT
			}
			return address & maskReadRIOT, RIOT
		}
		return address & maskWriteRIOT, RIOT
	}

	// RAM addresses
	if address&OriginRAM == OriginRAM {
		return address & maskRAM, RAM
	}

	// everything else is in TIA space
	if read {
		return address & maskReadTIA, TIA
	}

	return address & maskWriteTIA, TIA
}

// IsArea returns true if the address is in the specificied area.
func IsArea(address uint16, area Area) bool {
	_, a := MapAddress(address, true)
	return area == a
}
