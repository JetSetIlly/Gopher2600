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

package cartridge

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// from bankswitch_sizes.txt:
//
// -E7: Only M-Network used this scheme.  This has to be the most complex
// method used in any cart! :-) It allows for the capability of 2K of RAM;
// although it doesn't have to be used (in fact, only one cart used it-
// Burgertime).  This is similar to the 3F type with a few changes.  There are
// now 8 2K banks, instead of 4.
//
// The last 2K in the cart always points to the last 2K of the ROM image, while
// the first 2K is selectable.  You access 1FE0 to 1FE6 to select which 2K
// bank. Note that you cannot select the last 2K of the ROM image into the
// lower 2K of the cart!
//
// Accessing 1FE7 selects 1K of RAM at 1000-17FF instead of ROM!  The 2K of RAM
// is broken up into two 1K sections.  One 1K section is mapped in at 1000-17FF
// if 1FE7 has been accessed.  1000-13FF is the write port, while 1400-17FF is
// the read port.
//
// The second 1K of RAM appears at 1800-19FF.  1800-18FF is the
// write port while 1900-19FF is the read port.  You select which 256 byte
// block appears here by accessing 1FF8 to 1FFB.
//
//
// from the same document, more detail about M-Network RAM:
//
// OK, the RAM setup in these carts is very complex.  There is a total of 2K
// of RAM broken up into 2 1K pieces.  One 1K piece goes into 1000-17FF
// if the bankswitch is set to $1FE7.  The other is broken up into 4 256-byte
// parts.
//
// You select which part to use by issuing a fake read to 1FE8-1FEB.  The
// RAM is then available for use by all banks at 1800-19FF.
//
// Similar to other schemes, 1800-18FF is write while 1900-19FF is read.
// Low RAM uses 1000-13FF for write and 1400-17FF for read.
//
// Note that the 256-byte banks and the large 1K bank are seperate entities.
// The M-Network carts are about as complex as it gets.

const num256ByteRAMbanks = 4

type mnetwork struct {
	mappingID   string
	description string

	// mnetwork cartridges have 8 banks of 2048 bytes
	bankSize int
	banks    [][]uint8

	// identifies the currently selected bank
	bank int

	ram256byte    [num256ByteRAMbanks][]uint8
	ram256byteIdx int

	//  o ram1k is read through addresses 0x1000 to 0x13ff and written
	//  through addresses 0x1400 to 0x17ff * when use1kRAM is true
	//
	//  o ram256byte is read through addresses 0x1900 to 0x19fd and written
	//  through address 0x1800 to 0x18ff in all cases
	//
	// (addresses quoted above are of course masked so that they fall into the
	// allocation range)
	ram1k []uint8

	// use1kRAM is set to true when hotspot 0x0fe7 has been triggered. it's not
	// clear when, if ever, the flag should be set to false. we have taken the
	// view that is is when any of hotspots 0x0fe0 to 0x0fe6 are triggered
	use1kRAM bool
}

func newMnetwork(data []byte) (mapper.CartMapper, error) {
	cart := &mnetwork{
		description: "mnetwork",
		mappingID:   "E7",
		bankSize:    2048,
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("%s: wrong number of bytes in the cartridge data", cart.mappingID)
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	// not all m-network cartridges have any RAM but we'll allocate it for all
	// instances because there's no way of detecting if it does or not.
	cart.ram1k = make([]uint8, 1024)
	for i := range cart.ram256byte {
		cart.ram256byte[i] = make([]uint8, 256)
	}

	cart.Initialise()

	return cart, nil
}

func (cart mnetwork) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s [%s]", cart.mappingID, cart.description))
	s.WriteString(fmt.Sprintf(" Bank: %d ", cart.bank))
	s.WriteString(fmt.Sprintf(" RAM: %d", cart.ram256byteIdx))
	if cart.use1kRAM {
		s.WriteString(" [+1k RAM]")
	}
	return s.String()
}

// ID implements the mapper.CartMapper interface
func (cart mnetwork) ID() string {
	return cart.mappingID
}

// Initialise implements the mapper.CartMapper interface
func (cart *mnetwork) Initialise() {
	cart.bank = 0
	cart.ram256byteIdx = 0

	for i := range cart.ram1k {
		cart.ram1k[i] = 0x00
	}

	for i := range cart.ram256byte {
		for j := range cart.ram256byte[i] {
			cart.ram256byte[i][j] = 0x00
		}
	}
}

// Read implements the mapper.CartMapper interface
func (cart *mnetwork) Read(addr uint16, passive bool) (uint8, error) {
	if cart.hotspot(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	var data uint8

	if addr >= 0x0000 && addr <= 0x07ff {
		if cart.use1kRAM && addr >= 0x0400 {
			data = cart.ram1k[addr&0x03ff]
		} else {
			data = cart.banks[cart.bank][addr&0x07ff]
		}
	} else if addr >= 0x0800 && addr <= 0x0fff {
		if addr >= 0x0900 && addr <= 0x09ff {
			// access upper 1k of ram if cart.segment is pointing to ram and
			// the address is in the write range
			data = cart.ram256byte[cart.ram256byteIdx][addr&0x00ff]
		} else {
			// if address is not in ram space then read from the last rom bank
			data = cart.banks[cart.NumBanks()-1][addr&0x07ff]
		}
	} else {
		return 0, curated.Errorf(bus.AddressError, addr)
	}

	return data, nil
}

// Write implements the mapper.CartMapper interface
func (cart *mnetwork) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.hotspot(addr, passive) {
		return nil
	}

	if addr >= 0x0000 && addr <= 0x07ff {
		if addr <= 0x03ff && cart.use1kRAM {
			cart.ram1k[addr&0x03ff] = data
			return nil
		}
	} else if addr >= 0x0800 && addr <= 0x08ff {
		cart.ram256byte[cart.ram256byteIdx][addr&0x00ff] = data
		return nil
	}

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return curated.Errorf(bus.AddressError, addr)
}

// bankswitch on hotspot access
func (cart *mnetwork) hotspot(addr uint16, passive bool) bool {
	if (addr >= 0xfe0 && addr <= 0xfe7) || (addr >= 0xff8 && addr <= 0x0ffb) {
		if passive {
			return true
		}

		switch addr {
		case 0x0fe0:
			cart.bank = 0
			cart.use1kRAM = false
		case 0x0fe1:
			cart.bank = 1
			cart.use1kRAM = false
		case 0x0fe2:
			cart.bank = 2
			cart.use1kRAM = false
		case 0x0fe3:
			cart.bank = 3
			cart.use1kRAM = false
		case 0x0fe4:
			cart.bank = 4
			cart.use1kRAM = false
		case 0x0fe5:
			cart.bank = 5
			cart.use1kRAM = false
		case 0x0fe6:
			cart.bank = 6
			cart.use1kRAM = false

			// from bankswitch_sizes.txt: "Note that you cannot select the last 2K
			// of the ROM image into the lower 2K of the cart!  Accessing 1FE7
			// selects 1K of RAM at 1000-17FF instead of ROM!"
			//
			// we're using bank number -1 to indicate the use of RAM
		case 0x0fe7:
			cart.use1kRAM = true

			// from bankswitch_sizes.txt: "You select which 256 byte block appears
			// here by accessing 1FF8 to 1FFB."
			//
			// "here" refers to the read range 0x0900 to 0x09ff and the write range
			// 0x0800 to 0x08ff
		case 0x0ff8:
			cart.ram256byteIdx = 0
		case 0x0ff9:
			cart.ram256byteIdx = 1
		case 0x0ffa:
			cart.ram256byteIdx = 2
		case 0x0ffb:
			cart.ram256byteIdx = 3
		}

		return true
	}

	return false
}

// NumBanks implements the mapper.CartMapper interface
func (cart mnetwork) NumBanks() int {
	return 8 // eight banks of 2k
}

// GetBank implements the mapper.CartMapper interface
func (cart *mnetwork) GetBank(addr uint16) banks.Details {
	if addr >= 0x0000 && addr <= 0x07ff {
		if cart.use1kRAM {
			return banks.Details{Number: cart.bank, IsRAM: true, Segment: 0}
		}
		return banks.Details{Number: cart.bank, IsRAM: false, Segment: 0}
	}

	if addr >= 0x0800 && addr <= 0x08ff {
		return banks.Details{Number: cart.ram256byteIdx, IsRAM: true, Segment: 1}
	}

	return banks.Details{Number: 7, IsRAM: false, Segment: 1}
}

// Patch implements the mapper.CartMapper interface
func (cart *mnetwork) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("%s: patch offset too high (%v)", cart.ID(), offset)
	}

	bank := int(offset) / cart.bankSize
	offset = offset % cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface
func (cart *mnetwork) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface
func (cart *mnetwork) Step() {
}

// GetRAM implements the bus.CartRAMBus interface
func (cart mnetwork) GetRAM() []bus.CartRAM {
	r := make([]bus.CartRAM, num256ByteRAMbanks+1)

	r[0] = bus.CartRAM{
		Label:  "1k",
		Origin: 0x1000,
		Data:   make([]uint8, len(cart.ram1k)),
		Mapped: cart.use1kRAM,
	}
	copy(r[0].Data, cart.ram1k)

	for i := 0; i < num256ByteRAMbanks; i++ {
		r[i+1] = bus.CartRAM{
			Label:  fmt.Sprintf("256B [%d]", i),
			Origin: 0x1900,
			Data:   make([]uint8, len(cart.ram256byte[i])),
			Mapped: cart.ram256byteIdx == i,
		}
		copy(r[i+1].Data, cart.ram256byte[i])
	}

	return r
}

// PutRAM implements the bus.CartRAMBus interface
func (cart *mnetwork) PutRAM(bank int, idx int, data uint8) {
	if bank == 0 {
		cart.ram1k[idx] = data
		return
	}
	cart.ram256byte[bank-1][idx] = data
}

// IterateBank implemnts the disassemble interface
func (cart mnetwork) IterateBanks(prev *banks.Content) *banks.Content {
	b := prev.Number + 1
	if b >= 0 && b <= 6 {
		// includes 1k ram section
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart,
			},
		}
	} else if b == 7 {
		// includes 256B ram section
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart + uint16(cart.bankSize),
			},
		}
	}
	return nil
}
