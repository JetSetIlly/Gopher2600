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

package supercharger

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

type TapeLoaded func(*cpu.CPU, *vcs.RAM, *timer.Timer) error

func (er TapeLoaded) Error() string {
	return fmt.Sprintf("supercharger tape loaded, preparing VCS")
}

type Tape struct {
	cart *Supercharger
	data []byte
}

func (tap *Tape) load() error {
	gameData := tap.data[0:0x1eff]

	// only 8448 .bin format is supported currently
	gameHeader := tap.data[0x2000:0x2008]

	// PC address to jump to once loading has finished
	startAddress := (uint16(gameHeader[1]) << 8) | uint16(gameHeader[0])

	// RAM config to be set adter tape load
	configByte := gameHeader[2]

	// number of pages to load
	numPages := int(gameHeader[3])

	// not using the following in any meaningful way
	checksum := gameHeader[4]
	multiLoad := gameHeader[5]
	progressCounter := (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])

	fmt.Printf("start address: %#04x\n", startAddress)
	fmt.Printf("config byte: %#08b\n", configByte)
	fmt.Printf("num pages: %d\n", numPages)
	fmt.Printf("checksum: %#02x\n", checksum)
	fmt.Printf("multi load: %#02x\n", multiLoad)
	fmt.Printf("progress counter: %#02x\n", progressCounter)
	fmt.Println("")

	// data is loaded accoring to page table
	pageTable := tap.data[0x2010:0x2028]
	fmt.Printf("page-table\n----------\n%v\n\n", pageTable)

	// copy data to RAM banks
	for i := 0; i < numPages; i++ {
		bank := pageTable[i] & 0x3
		page := pageTable[i] >> 2
		bankOffset := int(page) * 0x100
		binOffset := i * 0x100

		data := gameData[binOffset : binOffset+0x100]
		copy(tap.cart.ram[bank][bankOffset:bankOffset+0x100], data)

		fmt.Printf("copying %#04x:%#04x to bank %d page %d, offset %#04x\n", binOffset, binOffset+0x100, bank, page, bankOffset)
	}

	// setup cartridge according to tape instructions. we do this by returning
	// a function disguised as an error type. The VCS knows how to interpret
	// this error and will call the function
	return TapeLoaded(func(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
		tap.cart.registers.setConfigByte(configByte)

		err := mc.LoadPC(startAddress)
		if err != nil {
			return errors.New(errors.SuperchargerError, err)
		}

		// clear RAM
		for a := uint16(0x80); a <= 0xff; a++ {
			ram.Poke(a, 0x00)
		}

		// poke some values into RAM
		ram.Poke(0xfa, 0xcd)
		ram.Poke(0xfb, 0xf8)
		ram.Poke(0xfc, 0xff)
		ram.Poke(0xfd, 0x4c)
		ram.Poke(0xfe, uint8(startAddress))
		ram.Poke(0xff, uint8(startAddress>>8))

		// RAM address 0x80 seems to be sensitive to the specific cartridge. it
		// seems to be used for bank-switching
		//
		// for Frogger and Communist Mutants it must 0x0b
		// for Killer Instinct it must be 0x0f
		ram.Poke(0x80, 0x0b)

		// reset timer. rabbit transit requires this
		tmr.SetInterval("T1024T")
		tmr.SetValue(0x4c)

		return nil
	})
}
