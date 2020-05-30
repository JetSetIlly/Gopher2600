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

package cartridge

import "github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

// cartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument recieve that address
// normalised to a range of 0x0000 to 0x0fff
type cartMapper interface {
	Initialise()
	ID() string
	Read(addr uint16) (data uint8, err error)
	Write(addr uint16, data uint8) error
	NumBanks() int
	GetBank(addr uint16) (bank int)
	SetBank(addr uint16, bank int) error
	SaveState() interface{}
	RestoreState(interface{}) error

	// poke new value anywhere into currently selected bank of cartridge memory
	// (including ROM).
	Poke(addr uint16, data uint8) error

	// patch differs from poke in that it alters the data as though it was
	// being read from disk
	Patch(offset uint16, data uint8) error

	// see the commentary for the Listen() function in the Cartridge type for
	// an explanation for what this does
	Listen(addr uint16, data uint8)

	// some cartridge mappings have indpendent clocks that tick and change
	// internal cartridge state. the step() function is called every cpu cycle
	// at a rate of 1.19. cartridges with slower clocks need to handle the rate
	// change.
	Step()

	// some cartridge mappings have additional RAM. getRAM() returns an array of
	// memorymap.SubArea, or nil if the cartridge has no RAM
	GetRAM() []memorymap.SubArea
}

// optionalSuperchip are implemented by cartMappers that have an optional
// superchip
type optionalSuperchip interface {
	addSuperchip() bool
}
