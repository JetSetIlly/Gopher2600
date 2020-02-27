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

// cartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument recieve that address
// normalised to a range of 0x0000 to 0x0fff
type cartMapper interface {
	initialise()
	read(addr uint16) (data uint8, err error)
	write(addr uint16, data uint8) error
	numBanks() int
	getBank(addr uint16) (bank int)
	setBank(addr uint16, bank int) error
	saveState() interface{}
	restoreState(interface{}) error

	// tigervision cartridges have a very wierd bank-switching method that
	// require a way of notifying the cartridge of writes to addresses outside
	// of cartridge space
	listen(addr uint16, data uint8)

	// poke new value anywhere into currently selected bank of cartridge memory
	// (including ROM).
	poke(addr uint16, data uint8) error

	// patch differs from poke in that it alters the data as though it was
	// being read from disk
	patch(offset uint16, data uint8) error

	getRAMinfo() []RAMinfo
}

// optionalSuperchip are implemented by cartMappers that have an optional
// superchip
type optionalSuperchip interface {
	addSuperchip() bool
}

// RAMinfo details the read/write addresses for any cartridge ram
type RAMinfo struct {
	Label       string
	Active      bool
	ReadOrigin  uint16
	ReadMemtop  uint16
	WriteOrigin uint16
	WriteMemtop uint16
}
