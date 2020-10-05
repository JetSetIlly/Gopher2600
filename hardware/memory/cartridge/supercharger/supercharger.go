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

package supercharger

import (
	"fmt"
	"path"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const MappingID = "AR"

// supercharger has 6k of RAM in total
const numRamBanks = 4
const bankSize = 2048

// tape defines the operations required by the $fff9 tape loader. With this
// interface, the Supercharger implementation supports both fast-loading
// from a Stella bin file, and "slow" loading from a sound file.
type tape interface {
	load() (uint8, error)
	step()
}

// Supercharger represents a supercharger cartridge
type Supercharger struct {
	mappingID   string
	description string

	tape      tape
	registers Registers

	bankSize int
	bios     []uint8
	ram      [3][]uint8

	onLoaded func(cart mapper.CartMapper) error
}

// NewSupercharger is the preferred method of initialisation for the
// Supercharger type
func NewSupercharger(cartload cartridgeloader.Loader) (mapper.CartMapper, error) {
	cart := &Supercharger{
		mappingID:   MappingID,
		description: "supercharger",
		bankSize:    2048,
	}

	var err error

	// set up tape
	if cartload.IsSoundData {
		cart.tape, err = NewSoundLoad(cart, cartload)
	} else {
		cart.tape, err = NewFastLoad(cart, cartload)
	}
	if err != nil {
		return nil, curated.Errorf("supercharger: %v", err)
	}

	// allocate ram
	for i := range cart.ram {
		cart.ram[i] = make([]uint8, bankSize)
	}

	// load bios and activate
	cart.bios, err = loadBIOS(path.Dir(cartload.Filename))
	if err != nil {
		return nil, curated.Errorf("supercharger: %v", err)
	}

	// prepare onLoaded function
	if cartload.OnLoaded == nil {
		cart.onLoaded = func(cart mapper.CartMapper) error { return nil }
	} else {
		cart.onLoaded = cartload.OnLoaded
	}

	cart.Initialise()

	return cart, nil
}

func (cart Supercharger) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s [%s] ", cart.mappingID, cart.description))
	s.WriteString(cart.registers.BankString())
	return s.String()
}

// ID implements the cartMapper interface
func (cart Supercharger) ID() string {
	return cart.mappingID
}

// Initialise implements the cartMapper interface
func (cart *Supercharger) Initialise() {
	cart.registers.WriteDelay = 0
	cart.registers.BankingMode = 0
	cart.registers.ROMpower = true
	cart.registers.RAMwrite = true
}

// Read implements the cartMapper interface
func (cart *Supercharger) Read(fullAddr uint16, passive bool) (uint8, error) {
	addr := fullAddr & memorymap.CartridgeBits

	// what bank to read. bank zero refers to the BIOS. bank 1 to 3 refer to
	// one of the RAM banks
	bank := cart.GetBank(addr).Number

	bios := false
	switch bank {
	case 0:
		bios = true
	default:
		// RAM banks are indexed from 0 to 2
		bank--
	}

	// control register has been read. I've opted to return the value at the
	// address before the bank switch. I think this is correct but I'm not
	// sure.
	if addr == 0x0ff8 {
		b := cart.ram[bank][addr&0x07ff]
		if !passive {
			cart.registers.setConfigByte(cart.registers.Value)
			cart.registers.Delay = 0
		}
		return b, nil
	}

	if addr == 0x0ff9 {
		// call load() whenever address is touched, although do not allow
		// it if RAMwrite is false
		if passive || !cart.registers.RAMwrite {
			return 0, nil
		}

		return cart.tape.load()
	}

	// note address to be used as the next value in the control register
	if !passive {
		if fullAddr&0xf000 == 0xf000 && fullAddr <= 0xf0ff {
			if cart.registers.Delay == 0 {
				cart.registers.Value = uint8(fullAddr & 0x00ff)
				cart.registers.Delay = 6
			}
		}
	}

	if bios {
		if cart.registers.ROMpower {
			// trigger onLoaded() function whenever BIOS address $fa1a
			// (specifically) is touched. note that this method means that the
			// onLoaded() function will be called whatever the context the
			// address is read and not just when the PC is at the address.
			if fullAddr == 0xfa1a {
				err := cart.onLoaded(cart)
				if err != nil {
					return 0, curated.Errorf("supercharger: %v", err)
				}
			}

			return cart.bios[addr&0x07ff], nil
		}

		return 0, curated.Errorf("supercharger: ROM is powered off")
	}

	if !passive && cart.registers.Delay == 1 {
		if cart.registers.RAMwrite {
			cart.ram[bank][addr&0x07ff] = cart.registers.Value
			cart.registers.LastWriteAddress = fullAddr
			cart.registers.LastWriteValue = cart.registers.Value
		}
	}

	return cart.ram[bank][addr&0x07ff], nil
}

// Write implements the cartMapper interface
func (cart *Supercharger) Write(addr uint16, data uint8, passive bool, poke bool) error {
	return nil
}

// NumBanks implements the cartMapper interface
func (cart Supercharger) NumBanks() int {
	return numRamBanks
}

// GetBank implements the cartMapper interface
func (cart Supercharger) GetBank(addr uint16) mapper.BankInfo {
	switch cart.registers.BankingMode {
	case 0:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, IsRAM: false, Segment: 0}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 1:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, IsRAM: false, Segment: 0}
		}
		return mapper.BankInfo{Number: 1, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 2:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 1, IsRAM: cart.registers.RAMwrite, Segment: 0}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 3:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 0}
		}
		return mapper.BankInfo{Number: 1, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 4:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, IsRAM: false, Segment: 0}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 5:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, IsRAM: false, Segment: 0}
		}
		return mapper.BankInfo{Number: 2, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 6:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 2, IsRAM: cart.registers.RAMwrite, Segment: 0}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 1}

	case 7:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 3, IsRAM: cart.registers.RAMwrite, Segment: 0}
		}
		return mapper.BankInfo{Number: 2, IsRAM: cart.registers.RAMwrite, Segment: 1}
	}
	panic("unknown banking method")
}

// Patch implements the cartMapper interface
func (cart *Supercharger) Patch(_ int, _ uint8) error {
	return curated.Errorf("%s: not patchable")
}

// Listen implements the cartMapper interface
func (cart *Supercharger) Listen(addr uint16, _ uint8) {
	cart.registers.transitionCount(addr)
}

// Step implements the cartMapper interface
func (cart *Supercharger) Step() {
	cart.tape.step()
}

// IterateBank implements the mapper.CartMapper interface
func (cart Supercharger) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.ram)+1)

	c[0] = mapper.BankContent{Number: 0,
		Data: cart.bios,
		Origins: []uint16{
			memorymap.OriginCart,
			memorymap.OriginCart + uint16(cart.bankSize),
		},
	}

	for b := 0; b < len(cart.ram); b++ {
		c[b+1] = mapper.BankContent{Number: b + 1,
			Data: cart.ram[b],
			Origins: []uint16{
				memorymap.OriginCart,
				memorymap.OriginCart + uint16(cart.bankSize),
			},
		}
	}

	return c
}

// GetRAM implements the mapper.CartRAMBus interface
func (cart Supercharger) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, len(cart.ram))

	for i := 0; i < len(cart.ram); i++ {
		mapped := false
		origin := uint16(0x1000)

		// in the documentation and for presentation purporses, RAM banks are
		// counted from 1. when deciding if a bank is mapped or not, we'll use
		// this value rather than the i index; being consistent with the
		// documentation is clearer
		bank := i + 1

		switch cart.registers.BankingMode {
		case 0:
			mapped = bank == 3

		case 1:
			mapped = bank == 1

		case 2:
			mapped = bank == 1
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 3
			}

		case 3:
			mapped = bank == 3
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 1
			}

		case 4:
			mapped = bank == 3

		case 5:
			mapped = bank == 2

		case 6:
			mapped = bank == 2
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 3
			}

		case 7:
			mapped = bank == 3
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 2
			}
		}

		r[i] = mapper.CartRAM{
			Label:  fmt.Sprintf("2048k [%d]", bank),
			Origin: origin,
			Data:   make([]uint8, len(cart.ram[i])),
			Mapped: mapped,
		}
		copy(r[i].Data, cart.ram[i])
	}

	return r
}

// PutRAM implements the mapper.CartRAMBus interface
func (cart *Supercharger) PutRAM(bank int, idx int, data uint8) {
	if bank < len(cart.ram) {
		cart.ram[bank][idx] = data
		return
	}
}

// Rewind implements the mapper.CartTapeBus interface
//
// Whether this does anything meaningful depends on the interal implementation
// of the 'tape' interface.
func (cart *Supercharger) Rewind() bool {
	if tape, ok := cart.tape.(mapper.CartTapeBus); ok {
		return tape.Rewind()
	}
	return false
}

// GetTapeState implements the mapper.CartTapeBus interface
//
// Whether this does anything meaningful depends on the interal implementation
// of the 'tape' interface.
func (cart *Supercharger) GetTapeState() (bool, mapper.CartTapeState) {
	if tape, ok := cart.tape.(mapper.CartTapeBus); ok {
		return tape.GetTapeState()
	}
	return false, mapper.CartTapeState{}
}
