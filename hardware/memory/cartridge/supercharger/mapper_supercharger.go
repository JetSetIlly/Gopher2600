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
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// supercharger has 6k of RAM in total
const numRamBanks = 4
const bankSize = 2048

// Supercharger represents a supercharger cartridge
type Supercharger struct {
	mappingID   string
	description string

	registers SuperChargerRegisters

	bios []uint8
	ram  [3][]uint8
	bank int

	tape        []byte
	tapeCtr     int
	tapeByteCtr int
}

// SuperChargerRegisters implements the bus.CartRegisters interface
type SuperChargerRegisters struct {
	value uint8

	WriteDelay  int
	BankingMode int
	RAMwrite    bool
	ROMpower    bool

	Transitions int // 0=off, 1=trigger
}

func (r SuperChargerRegisters) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("wd: %03b  bbb: %03b  ram: %v  rom: %v", r.WriteDelay, r.BankingMode, r.RAMwrite, r.ROMpower))
	return s.String()
}

func (r *SuperChargerRegisters) tick() {
	if r.Transitions > 0 {
		r.Transitions--
	}
}

// NewSupercharger is the preferred method of initialisation for the
// Supercharger type
func NewSupercharger(data []byte) (*Supercharger, error) {
	cart := &Supercharger{
		mappingID:   "AR",
		description: "supercharger",
		tape:        data,
	}

	// allocate ram
	for i := range cart.ram {
		cart.ram[i] = make([]uint8, bankSize)
	}

	var err error

	// load bios and activate
	cart.bios, err = loadBIOS()
	if err != nil {
		return nil, err
	}

	cart.Initialise()

	return cart, nil
}

func (cart Supercharger) String() string {
	s := strings.Builder{}
	s.WriteString(cart.description)
	s.WriteString(" ")
	s.WriteString(cart.registers.String())
	return s.String()
}

// ID implements the cartMapper interface
func (cart Supercharger) ID() string {
	return "-"
}

// Initialise implements the cartMapper interface
func (cart *Supercharger) Initialise() {
	cart.bank = 0
	cart.registers.ROMpower = true
}

// Read implements the cartMapper interface
func (cart *Supercharger) Read(addr uint16, active bool) (uint8, error) {
	cart.registers.tick()

	switch addr {
	case 0x0ff8:
		// control register
		cart.registers.ROMpower = cart.registers.value&0x01 != 0x01
		cart.registers.RAMwrite = cart.registers.value&0x02 == 0x02
		cart.registers.BankingMode = int((cart.registers.value >> 2) & 0x07)
		cart.registers.WriteDelay = int((cart.registers.value >> 5) & 0x07)
		return 0, nil

	case 0x0ff9:
		b := (cart.tape[cart.tapeCtr] >> cart.tapeByteCtr) & 0x01
		cart.tapeByteCtr++
		if cart.tapeByteCtr > 7 {
			cart.tapeByteCtr = 0
			cart.tapeCtr++
		}

		return b, nil
	}

	// note address to be used as the next value in the control register
	if active && addr <= 0x00ff {
		cart.registers.value = uint8(addr & 0xff)
		cart.registers.Transitions = 4
	}

	bios := false
	bank := cart.GetBank(addr).Number

	switch bank {
	case 0:
		bios = true
	default:
		bank--
	}

	if bios {
		if cart.registers.ROMpower {
			return cart.bios[addr&0x7ff], nil
		}
		return 0, errors.New(errors.SuperchargerError, "ROM is powered off")
	}

	if cart.registers.Transitions == 1 {
		if cart.registers.RAMwrite {
			cart.ram[bank][addr&0x7ff] = cart.registers.value
		}
	}

	return cart.ram[bank][addr&0x7ff], nil
}

// Write implements the cartMapper interface
func (cart *Supercharger) Write(addr uint16, data uint8, active bool, poke bool) error {
	cart.registers.tick()
	return nil
}

// NumBanks implements the cartMapper interface
func (cart Supercharger) NumBanks() int {
	return numRamBanks
}

// SetBank implements the cartMapper interface
func (cart *Supercharger) SetBank(_ uint16, _ int) error {
	return nil
}

// GetBank implements the cartMapper interface
func (cart Supercharger) GetBank(addr uint16) memorymap.BankDetails {
	switch cart.registers.BankingMode {
	case 0:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 0, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 3, IsRAM: true, Segment: 1}

	case 1:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 0, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 1, IsRAM: true, Segment: 1}

	case 2:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 1, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 3, IsRAM: true, Segment: 1}

	case 3:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 3, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 1, IsRAM: true, Segment: 1}

	case 4:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 0, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 3, IsRAM: true, Segment: 1}

	case 5:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 0, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 2, IsRAM: true, Segment: 1}

	case 6:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 2, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 1, IsRAM: true, Segment: 1}

	case 7:
		if addr >= 0x800 {
			return memorymap.BankDetails{Number: 3, IsRAM: true, Segment: 0}
		}
		return memorymap.BankDetails{Number: 2, IsRAM: true, Segment: 1}
	}
	panic("unknown banking method")
}

// Patch implements the cartMapper interface
func (cart *Supercharger) Patch(_ int, _ uint8) error {
	return nil
}

// Listen implements the cartMapper interface
func (cart *Supercharger) Listen(_ uint16, _ uint8) {
	cart.registers.tick()
}

// Step implements the cartMapper interface
func (cart *Supercharger) Step() {
}
