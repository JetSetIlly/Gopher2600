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
	"math/rand"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type superbank struct {
	mappingID   string
	description string

	// superbank cartridges can have 32 banks (128k) or 64 banks (256k)
	bankSize int
	banks    [][]uint8

	// the mask we use to decide whether the listen() address will trigger a
	// bankswitch. this will change depending on the exact size of the
	// cartridge.
	bankSwitchMask uint16

	// rewindable state
	state *superbankState
}

func newSuperbank(data []byte) (mapper.CartMapper, error) {
	cart := &superbank{
		mappingID:   "SB",
		description: "Superbank",
		bankSize:    4096,
		state:       newSuperbankState(),
	}

	if len(data)%cart.bankSize != 0 {
		return nil, curated.Errorf("SB: %v", "wrong number of bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, len(data)/cart.bankSize)
	cart.bankSwitchMask = uint16(cart.NumBanks() - 1)

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// Mapping implements the mapper.CartMapper interface.
func (cart *superbank) Mapping() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *superbank) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *superbank) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *superbank) Plumb() {
}

// Reset implements the cartMapper interface.
func (cart *superbank) Reset(randSrc *rand.Rand) {
	cart.state.bank = len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *superbank) Read(addr uint16, passive bool) (uint8, error) {
	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *superbank) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("SB: %v", curated.Errorf(bus.AddressError, addr))
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *superbank) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *superbank) GetBank(addr uint16) mapper.BankInfo {
	// superbank cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *superbank) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("SB: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *superbank) Listen(addr uint16, _ uint8) {
	// return with no side effect if address is not a TIA mirror and address
	// within that mirror is too large for the size of the cartridge.
	if addr&0x0800 != 0x0800 || addr&(cart.bankSwitchMask^0xff) != 0x0000 {
		return
	}

	// switch banks if address is in range of the number of banks
	bank := int(addr & cart.bankSwitchMask)

	// belt and braces check (probably not necessary)
	if bank < len(cart.banks) {
		cart.state.bank = bank
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *superbank) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *superbank) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *superbank) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *superbank) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for the superbank cartridge.
type superbankState struct {
	// identifies the currently selected bank
	bank int
}

func newSuperbankState() *superbankState {
	return &superbankState{}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *superbankState) Snapshot() *superbankState {
	n := *s
	return &n
}
