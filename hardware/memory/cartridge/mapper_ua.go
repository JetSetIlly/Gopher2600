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
	"crypto/sha1"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

type ua struct {
	env *environment.Environment

	mappingID string

	// ua cartridges are 8k in size and have two banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// identifies the currently selected bank
	bank int

	// the hotspot addresses are swapped
	swappedHotspots bool
}

func newUA(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("UA: %w", err)
	}

	cart := &ua{
		env:             env,
		mappingID:       "UA",
		bankSize:        4096,
		swappedHotspots: false,
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("UA: wrong number of bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	// only one cartridge dump is known to have swapped hotspots
	if fmt.Sprintf("%0x", sha1.Sum(data)) == "6d4a94c2348bbd8e9c73b73d8f3389196d42fd54" {
		cart.swappedHotspots = true
		logger.Logf(env, "UA", "swapping hotspot address for this cartridge (Sorcerer's Apprentice)")
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface
func (cart *ua) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.bank)
}

// ID implements the mapper.CartMapper interface
func (cart *ua) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface
func (cart *ua) Snapshot() mapper.CartMapper {
	n := *cart
	return &n
}

// Plumb implements the mapper.CartMapper interface
func (cart *ua) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface
func (cart *ua) Reset() {
	cart.bank = len(cart.banks) - 1
}

// Access implements the mapper.CartMapper interface
func (cart *ua) Access(addr uint16, _ bool) (uint8, uint8, error) {
	return cart.banks[cart.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface
func (cart *ua) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if poke {
		cart.banks[cart.bank][addr] = data
	}
	return nil
}

// NumBanks implements the mapper.CartMapper interface
func (cart *ua) NumBanks() int {
	return 2
}

// GetBank implements the mapper.CartMapper interface
func (cart *ua) GetBank(addr uint16) mapper.BankInfo {
	// ua cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.bank}
}

// AccessPassive implements the mapper.CartMapper interface
func (cart *ua) AccessPassive(addr uint16, data uint8) error {
	switch addr & 0x1260 {
	case 0x0220:
		if cart.swappedHotspots {
			cart.bank = 1
		} else {
			cart.bank = 0
		}
	case 0x0240:
		if cart.swappedHotspots {
			cart.bank = 0
		} else {
			cart.bank = 1
		}
	}
	return nil
}

// Step implements the mapper.CartMapper interface
func (cart *ua) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface
func (cart *ua) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// Patch implements the mapper.CartPatchable interface
func (cart *ua) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("UA: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}
