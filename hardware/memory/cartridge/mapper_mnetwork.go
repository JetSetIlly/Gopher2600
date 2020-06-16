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

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
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

func fingerprintMnetwork(b []byte) bool {
	threshold := 2
	for i := 0; i < len(b)-3; i++ {
		if b[i] == 0x7e && b[i+1] == 0x66 && b[i+2] == 0x66 && b[i+3] == 0x66 {
			threshold--
		}
		if threshold == 0 {
			return true
		}
	}

	return false
}

const num256ByteRAMbanks = 4

type mnetwork struct {
	mappingID   string
	description string

	bankSize int

	banks [][]uint8
	bank  int

	ram256byte    [num256ByteRAMbanks][]uint8
	ram256byteIdx int

	//  o ram1k is read through addresses 0x1000 to 0x13ff and written
	//  through addresses 0x1400 to 0x17ff * when bank == 7 *
	//
	//  o ram256byte is read through addresses 0x1900 to 0x19fd and written
	//  through address 0x1800 to 0x18ff in all cases
	//
	// (addresses quoted above are of course masked so that they fall into the
	// allocation range)
	ram1k []uint8
}

func newMnetwork(data []byte) (cartMapper, error) {
	cart := &mnetwork{
		description: "mnetwork",
		mappingID:   "E7",
		bankSize:    2048,
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.mappingID))
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
	if cart.bank == 7 {
		s.WriteString(" [+1k RAM]")
	}
	return s.String()
}

// ID implements the cartMapper interface
func (cart mnetwork) ID() string {
	return cart.mappingID
}

// Initialise implements the cartMapper interface
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

// Read implements the cartMapper interface
func (cart *mnetwork) Read(addr uint16, active bool) (uint8, error) {
	var data uint8

	if addr >= 0x0000 && addr <= 0x07ff {
		if cart.bank == 7 && addr >= 0x0400 {
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
			cart.bankSwitchOnAccess(addr)
		}
	} else {
		return 0, errors.New(errors.MemoryBusError, addr)
	}

	return data, nil
}

// Write implements the cartMapper interface
func (cart *mnetwork) Write(addr uint16, data uint8, active bool, poke bool) error {
	if addr >= 0x0000 && addr <= 0x07ff {
		if addr <= 0x03ff && cart.bank == 7 {
			cart.ram1k[addr&0x03ff] = data
			return nil
		}
	} else if addr >= 0x0800 && addr <= 0x08ff {
		cart.ram256byte[cart.ram256byteIdx][addr&0x00ff] = data
		return nil
	} else if cart.bankSwitchOnAccess(addr) {
		return nil
	}

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return errors.New(errors.MemoryBusError, addr)
}

func (cart *mnetwork) bankSwitchOnAccess(addr uint16) bool {
	switch addr {
	case 0x0fe0:
		cart.bank = 0
	case 0x0fe1:
		cart.bank = 1
	case 0x0fe2:
		cart.bank = 2
	case 0x0fe3:
		cart.bank = 3
	case 0x0fe4:
		cart.bank = 4
	case 0x0fe5:
		cart.bank = 5
	case 0x0fe6:
		cart.bank = 6

		// from bankswitch_sizes.txt: "Note that you cannot select the last 2K
		// of the ROM image into the lower 2K of the cart!  Accessing 1FE7
		// selects 1K of RAM at 1000-17FF instead of ROM!"
		//
		// we're using bank number -1 to indicate the use of RAM
	case 0x0fe7:
		cart.bank = 7

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

	default:
		return false
	}

	return true
}

// NumBanks implements the cartMapper interface
func (cart *mnetwork) NumBanks() int {
	return 8 // eight banks of 2k
}

// GetBank implements the cartMapper interface
func (cart *mnetwork) GetBank(addr uint16) (bank int) {
	if addr >= 0x0000 && addr <= 0x07ff {
		return cart.bank
	}
	return cart.ram256byteIdx
}

// SetBank implements the cartMapper interface
func (cart *mnetwork) SetBank(addr uint16, bank int) error {
	if addr >= 0x0000 && addr <= 0x07ff {
		cart.bank = bank
	} else if addr >= 0x0800 && addr <= 0x0fff {
		// last segment always points to the last bank
	} else {
		return errors.New(errors.CartridgeError, fmt.Sprintf("%s: invalid address [%#04x bank %d]", cart.mappingID, addr, bank))
	}

	return nil
}

// SaveState implements the cartMapper interface
func (cart *mnetwork) SaveState() interface{} {
	ram1k := make([]uint8, len(cart.ram1k))
	copy(ram1k, cart.ram1k)

	ram256byte := [4][]uint8{}
	for i := range ram256byte {
		ram256byte[i] = make([]uint8, len(cart.ram256byte[i]))
		copy(ram256byte[i], cart.ram256byte[i])
	}

	return []interface{}{cart.bank, cart.ram256byteIdx, ram1k, ram256byte}
}

// RestoreState implements the cartMapper interface
func (cart *mnetwork) RestoreState(state interface{}) error {
	cart.bank = state.([]interface{})[0].(int)
	cart.ram256byteIdx = state.([]interface{})[1].(int)

	ram1k := state.([]interface{})[2].([]uint8)
	copy(cart.ram1k, ram1k)

	ram256byte := state.([]interface{})[3].([4][]uint8)
	for i := range cart.ram256byte {
		copy(cart.ram256byte[i], ram256byte[i])
	}

	return nil
}

// Patch implements the cartMapper interface
func (cart *mnetwork) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return errors.New(errors.CartridgePatchOOB, offset)
	}

	bank := int(offset) / cart.bankSize
	offset = offset % cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the cartMapper interface
func (cart *mnetwork) Listen(_ uint16, _ uint8) {
}

// Step implements the cartMapper interface
func (cart *mnetwork) Step() {
}

// GetRAM implements the bus.CartRAMBus interface
func (cart mnetwork) GetRAM() []bus.CartRAM {
	r := make([]bus.CartRAM, num256ByteRAMbanks+1)

	r[0] = bus.CartRAM{
		Label:  "1k",
		Origin: 0x1000,
		Data:   make([]uint8, len(cart.ram1k)),
	}
	copy(r[0].Data, cart.ram1k)

	for i := 0; i < num256ByteRAMbanks; i++ {
		r[i+1] = bus.CartRAM{
			Label:  fmt.Sprintf("256B [%d]", i),
			Origin: 0x1900,
			Data:   make([]uint8, len(cart.ram256byte[i])),
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
