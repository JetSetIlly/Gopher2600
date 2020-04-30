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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
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

	// subArea information for cartridge ram
	ramDetails []memorymap.SubArea
}

func newCBS(data []byte) (cartMapper, error) {
	const bankSize = 4096

	cart := &cbs{}
	cart.description = "CBS"
	cart.mappingID = "FA"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.mappingID))
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		copy(cart.banks[k], data[offset:offset+bankSize])
	}

	// 256 bytes of cartidge ram in all instances
	cart.ram = make([]uint8, 256)

	// prepare ram details
	cart.ramDetails = make([]memorymap.SubArea, 5)
	cart.ramDetails[0] = memorymap.SubArea{
		Label:       "CBS RAM+",
		Active:      true,
		ReadOrigin:  0x1080,
		ReadMemtop:  0x10ff,
		WriteOrigin: 0x1000,
		WriteMemtop: 0x107f,
	}

	cart.initialise()

	return cart, nil
}

func (cart cbs) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.mappingID, cart.bank)
}

func (cart cbs) id() string {
	return cart.mappingID
}

func (cart *cbs) initialise() {
	cart.bank = len(cart.banks) - 1
	for i := range cart.ram {
		cart.ram[i] = 0x00
	}
}

func (cart *cbs) read(addr uint16) (uint8, error) {
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

func (cart *cbs) write(addr uint16, data uint8) error {
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

func (cart *cbs) numBanks() int {
	return 3
}

func (cart cbs) getBank(addr uint16) int {
	// cbs cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return cart.bank
}

func (cart *cbs) setBank(addr uint16, bank int) error {
	if bank < 0 || bank > len(cart.banks) {
		return errors.New(errors.CartridgeError, fmt.Sprintf("%s: invalid bank [%d]", cart.mappingID, bank))
	}
	cart.bank = bank
	return nil
}

func (cart *cbs) saveState() interface{} {
	superchip := make([]uint8, len(cart.ram))
	copy(superchip, cart.ram)
	return []interface{}{cart.bank, superchip}
}

func (cart *cbs) restoreState(state interface{}) error {
	cart.bank = state.([]interface{})[0].(int)
	copy(cart.ram, state.([]interface{})[1].([]uint8))
	return nil
}

func (cart *cbs) poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *cbs) patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.mappingID)
}

func (cart *cbs) listen(addr uint16, data uint8) {
}

func (cart *cbs) step() {
}

func (cart cbs) getRAM() []memorymap.SubArea {
	if cart.ram == nil {
		return nil
	}
	return cart.ramDetails
}
