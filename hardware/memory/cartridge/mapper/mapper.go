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

// Package mapper contains the CartMapper interface. This interface abstracts
// the functions required of any cartridge format.
package mapper

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
)

// CartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument recieve that address
// normalised to a range of 0x0000 to 0x0fff
type CartMapper interface {
	Initialise()
	String() string
	ID() string
	Read(addr uint16, active bool) (data uint8, err error)
	Write(addr uint16, data uint8, active bool, poke bool) error
	NumBanks() int
	GetBank(addr uint16) banks.Details

	// patch differs from write/poke in that it alters the data as though it
	// was being read from disk. that is, the offset is measured from the start
	// of the file. the cartmapper must translate the offset and update the
	// correct data structure as appropriate.
	Patch(offset int, data uint8) error

	// see the commentary for the Listen() function in the Cartridge type for
	// an explanation for what this does
	Listen(addr uint16, data uint8)

	// some cartridge mappings have indpendent clocks that tick and change
	// internal cartridge state. the step() function is called every cpu cycle
	// at a rate of 1.19MHz. cartridges with slower clocks need to handle the
	// rate change.
	Step()

	// return all the banks in the cartridge in sequence. see commentary for
	// IterateBanks() function in the Cartridge type for details.
	IterateBanks(prev *banks.Content) *banks.Content
}

// OptionalSuperchip are implemented by cartMappers that have an optional
// superchip
type OptionalSuperchip interface {
	AddSuperchip() bool
}

// CartContainer is a special CartMapper type that wraps another CartMapper.
// For example, the PlusROM type
type CartContainer interface {
	CartMapper
	ContainerID() string
}
