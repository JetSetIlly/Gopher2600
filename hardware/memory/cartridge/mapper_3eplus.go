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

func fingerprint3ePlus(b []byte) bool {
	// 3e is similar to tigervision, a key difference being that it uses 0x3e
	// to switch ram, in addition to 0x3f for switching banks.
	//
	// postulating that the fingerprint method can be the same except for the
	// write address.

	threshold3e := 5
	threshold3f := 5
	for i := range b {
		if b[i] == 0x85 && b[i+1] == 0x3e {
			threshold3e--
		}
		if b[i] == 0x85 && b[i+1] == 0x3f {
			threshold3f--
		}
		if threshold3e <= 0 && threshold3f <= 0 {
			return true
		}
	}
	return false
}

type mapper3ePlus struct {
	mappingID   string
	description string

	bankSize int
	banks    [][]uint8

	// 64 is the maximum number of banks possible under the 3e+ scheme
	ram [64][]uint8

	slot      [4]int
	slotIsRam [4]bool
}

// should work with any size cartridge that is a multiple of 1024
//  - tested with chess3E+20200519_3PQ6_SQ.bin
//	https://atariage.com/forums/topic/299157-chess/?do=findComment&comment=4541517
//
//	- specifciation:
//	https://atariage.com/forums/topic/300815-gopher2600-new-emulator/?do=findComment&comment=4562549
func new3ePlus(data []byte) (cartMapper, error) {
	const ramSize = 512

	cart := &mapper3ePlus{
		mappingID:   "3E+",
		description: "", // no description
		bankSize:    1024,
	}

	if len(data)%cart.bankSize != 0 {
		return nil, errors.New(errors.CartridgeError, "%s: cartridge size must be multiple of %d", cart.mappingID, cart.bankSize)
	}

	numBanks := len(data) / cart.bankSize
	cart.banks = make([][]uint8, numBanks)

	if len(data) != cart.bankSize*numBanks {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number bytes in the cartridge file", cart.mappingID))
	}

	// partition binary
	for k := 0; k < numBanks; k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	// allocate ram
	for k := 0; k < len(cart.ram); k++ {
		cart.ram[k] = make([]uint8, ramSize)
	}

	cart.Initialise()

	return cart, nil
}

func (cart mapper3ePlus) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s Slots: ", cart.mappingID))
	for i := range cart.slot {
		s.WriteString(fmt.Sprintf("%d", cart.slot[i]))
		if cart.slotIsRam[i] {
			s.WriteString("R ")
		} else {
			s.WriteString(" ")
		}
	}
	return s.String()
}

// ID implements the cartMapper interface
func (cart mapper3ePlus) ID() string {
	return cart.mappingID
}

// Initialise implements the cartMapper interface
func (cart *mapper3ePlus) Initialise() {
	// from spec:
	//
	// The last 1K ROM ($FC00-$FFFF) segment in the 6502 address space (ie: $1C00-$1FFF)
	// is initialised to point to the FIRST 1K of the ROM image, so the reset vectors
	// must be placed at the end of the first 1K in the ROM image.

	for i := range cart.slot {
		cart.slot[i] = 0
		cart.slotIsRam[i] = false
	}
}

// Read implements the cartMapper interface
func (cart *mapper3ePlus) Read(addr uint16, active bool) (uint8, error) {
	var slot int

	if addr >= 0x0000 && addr <= 0x03ff {
		slot = 0
	} else if addr >= 0x0400 && addr <= 0x07ff {
		slot = 1
	} else if addr >= 0x0800 && addr <= 0x0bff {
		slot = 2
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		slot = 3
	}

	var data uint8

	if cart.slotIsRam[slot] {
		data = cart.ram[cart.slot[slot]][addr&0x01ff]
	} else {
		bank := cart.slot[slot]
		if bank < len(cart.banks) {
			data = cart.banks[bank][addr&0x03ff]
		}
	}

	return data, nil
}

// Write implements the cartMapper interface
func (cart *mapper3ePlus) Write(addr uint16, data uint8, active bool, poke bool) error {
	var slot int
	if addr >= 0x0000 && addr <= 0x03ff {
		slot = 0
	} else if addr >= 0x0400 && addr <= 0x07ff {
		slot = 1
	} else if addr >= 0x0800 && addr <= 0x0bff {
		slot = 2
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		slot = 3
	}

	if cart.slotIsRam[slot] {
		cart.ram[cart.slot[slot]][addr&0x01ff] = data
		return nil
	} else if poke {
		cart.banks[cart.slot[slot]][addr&0x03ff] = data
		return nil
	}

	return errors.New(errors.MemoryBusError, addr)
}

// NumBanks implements the cartMapper interface
func (cart mapper3ePlus) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the cartMapper interface
func (cart *mapper3ePlus) GetBank(addr uint16) (bank int) {
	if addr >= 0x0000 && addr <= 0x03ff {
		return cart.slot[0]
	} else if addr >= 0x0400 && addr <= 0x07ff {
		return cart.slot[1]
	} else if addr >= 0x0800 && addr <= 0x0bff {
		return cart.slot[2]
	}

	// remaining address is between 0x0c00 and 0x0fff
	return cart.slot[3]
}

// SetBank implements the cartMapper interface
func (cart *mapper3ePlus) SetBank(addr uint16, bank int) error {
	if addr >= 0x0000 && addr <= 0x03ff {
		cart.slot[0] = bank
	} else if addr >= 0x0400 && addr <= 0x07ff {
		cart.slot[1] = bank
	} else if addr >= 0x0800 && addr <= 0x0bff {
		cart.slot[2] = bank
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		cart.slot[3] = bank
	}
	return nil
}

// SaveState implements the cartMapper interface
func (cart *mapper3ePlus) SaveState() interface{} {
	ram := [64][]uint8{}
	for i := range ram {
		ram[i] = make([]uint8, len(cart.ram[i]))
		copy(ram[i], cart.ram[i])
	}

	var slot [4]int
	for i := range slot {
		slot[i] = cart.slot[i]
	}

	var slotIsRam [4]bool
	for i := range slotIsRam {
		slotIsRam[i] = cart.slotIsRam[i]
	}

	return []interface{}{slot, slotIsRam, ram}
}

// RestoreState implements the cartMapper interface
func (cart *mapper3ePlus) RestoreState(state interface{}) error {
	slot := state.([]interface{})[0].([4]int)
	for i := range slot {
		cart.slot[i] = slot[i]
	}

	slotIsRam := state.([]interface{})[1].([4]bool)
	for i := range slotIsRam {
		cart.slotIsRam[i] = slotIsRam[i]
	}

	ram := state.([]interface{})[2].([64][]uint8)
	for i := range ram {
		copy(cart.ram[i], ram[i])
	}

	return nil
}

// Patch implements the cartMapper interface
func (cart *mapper3ePlus) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return errors.New(errors.CartridgePatchOOB, offset)
	}

	bank := int(offset) / cart.bankSize
	offset = offset % cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the cartMapper interface
func (cart *mapper3ePlus) Listen(addr uint16, data uint8) {
	// mapper 3e+ is a derivative of tigervision and so uses the same Listen()
	// mechanism

	if addr == 0x3f {
		slot := data >> 6
		bank := data & 0x3f
		cart.slot[slot] = int(bank)
		cart.slotIsRam[slot] = false
	} else if addr == 0x3e {
		slot := data >> 6
		bank := data & 0x3f
		cart.slot[slot] = int(bank)
		cart.slotIsRam[slot] = true
	}
}

// Step implements the cartMapper interface
func (cart *mapper3ePlus) Step() {
}

// GetRAM implements the bus.CartRAMBus interface
func (cart mapper3ePlus) GetRAM() []bus.CartRAM {
	r := make([]bus.CartRAM, len(cart.ram))

	for i := range cart.ram {
		r[i] = bus.CartRAM{
			Label:  fmt.Sprintf("%d", i),
			Origin: 0x1000,
			Data:   make([]uint8, len(cart.ram[i])),
		}
		copy(r[i].Data, cart.ram[i])
	}

	return r
}

// PutRAM implements the bus.CartRAMBus interface
func (cart *mapper3ePlus) PutRAM(bank int, idx int, data uint8) {
	cart.ram[bank][idx] = data
}
