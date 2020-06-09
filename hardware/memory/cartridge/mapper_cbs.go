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

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

type cbs struct {
	mappingID   string
	description string

	// cbs cartridges have 3 banks of 4096 bytes
	banks [][]uint8

	// identifies the currently selected bank
	bank int

	// CBS cartridges always have a RAM area
	ram []uint8
}

func newCBS(data []byte) (cartMapper, error) {
	const bankSize = 4096

	cart := &cbs{}
	cart.description = "CBS"
	cart.mappingID = "FA"
	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) != bankSize*cart.NumBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.mappingID))
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		copy(cart.banks[k], data[offset:offset+bankSize])
	}

	// 256 bytes of cartidge ram in all instances
	cart.ram = make([]uint8, 256)

	cart.Initialise()

	return cart, nil
}

func (cart cbs) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.mappingID, cart.bank)
}

// ID implements the cartMapper interface
func (cart cbs) ID() string {
	return cart.mappingID
}

// Initialise implements the cartMapper interface
func (cart *cbs) Initialise() {
	cart.bank = len(cart.banks) - 1
	for i := range cart.ram {
		cart.ram[i] = 0x00
	}
}

// Read implements the cartMapper interface
func (cart *cbs) Read(addr uint16) (uint8, error) {
	if addr >= 0x0100 && addr <= 0x01ff {
		return cart.ram[addr-0x100], nil
	}

	data := cart.banks[cart.bank][addr]

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else if addr == 0x0ffa {
		cart.bank = 2
	}

	return data, nil
}

// Write implements the cartMapper interface
func (cart *cbs) Write(addr uint16, data uint8) error {
	if addr <= 0x00ff {
		cart.ram[addr] = data
		return nil
	}

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else if addr == 0x0ffa {
		cart.bank = 2
	} else {
		return errors.New(errors.BusError, addr)
	}

	return nil
}

// NumBanks implements the cartMapper interface
func (cart *cbs) NumBanks() int {
	return 3
}

// GetBank implements the cartMapper interface
func (cart cbs) GetBank(addr uint16) int {
	// cbs cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return cart.bank
}

// SetBank implements the cartMapper interface
func (cart *cbs) SetBank(addr uint16, bank int) error {
	cart.bank = bank
	return nil
}

// SaveState implements the cartMapper interface
func (cart *cbs) SaveState() interface{} {
	superchip := make([]uint8, len(cart.ram))
	copy(superchip, cart.ram)
	return []interface{}{cart.bank, superchip}
}

// RestoreState implements the cartMapper interface
func (cart *cbs) RestoreState(state interface{}) error {
	cart.bank = state.([]interface{})[0].(int)
	copy(cart.ram, state.([]interface{})[1].([]uint8))
	return nil
}

// Poke implements the cartMapper interface
func (cart *cbs) Poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

// Patch implements the cartMapper interface
func (cart *cbs) Patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.mappingID)
}

// Listen implements the cartMapper interface
func (cart *cbs) Listen(addr uint16, data uint8) {
}

// Step implements the cartMapper interface
func (cart *cbs) Step() {
}

// GetRAM implements the bus.CartRAMBus interface
func (cart cbs) GetRAM() []bus.CartRAM {
	r := make([]bus.CartRAM, 1)
	r[0] = bus.CartRAM{
		Label:  "CBS+RAM",
		Origin: 0x1080,
		Data:   make([]uint8, len(cart.ram)),
	}
	copy(r[0].Data, cart.ram)
	return r
}

// PutRAM implements the bus.CartRAMBus interface
func (cart *cbs) PutRAM(_ int, idx int, data uint8) {
	cart.ram[idx] = data
}
