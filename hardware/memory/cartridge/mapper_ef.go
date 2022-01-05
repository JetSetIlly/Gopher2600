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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type ef struct {
	instance *instance.Instance

	mappingID   string
	description string

	// ef cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *efState
}

func newEF(instance *instance.Instance, data []byte) (mapper.CartMapper, error) {
	cart := &ef{
		instance:    instance,
		mappingID:   "EF",
		description: "16 bank 64k",
		bankSize:    4096,
		state:       newEFstate(),
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("EF: %v", "wrong number of bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *ef) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *ef) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *ef) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *ef) Plumb() {
}

// Reset implements the cartMapper interface.
func (cart *ef) Reset() {
	cart.state.bank = 0 //len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *ef) Read(addr uint16, passive bool) (uint8, error) {
	cart.bankswitch(addr, passive)

	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *ef) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("EF: %v", curated.Errorf(cpubus.AddressError, addr))
}

// bankswitch on hotspot access.
func (cart *ef) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0fe0 && addr <= 0x0fef {
		if passive {
			return true
		}
		cart.state.bank = int(addr & 0x000f)
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *ef) NumBanks() int {
	return 16
}

// GetBank implements the mapper.CartMapper interface.
func (cart *ef) GetBank(addr uint16) mapper.BankInfo {
	// ef cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: addr <= 0x00ff}
}

// Patch implements the mapper.CartMapper interface.
func (cart *ef) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("EF: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *ef) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *ef) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *ef) CopyBanks() []mapper.BankContent {
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
func (cart *ef) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1fe0: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1fe1: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1fe2: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1fe3: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1fe4: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1fe5: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1fe6: {Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
		0x1fe7: {Symbol: "BANK7", Action: mapper.HotspotBankSwitch},
		0x1fe8: {Symbol: "BANK8", Action: mapper.HotspotBankSwitch},
		0x1fe9: {Symbol: "BANK9", Action: mapper.HotspotBankSwitch},
		0x1fea: {Symbol: "BANK10", Action: mapper.HotspotBankSwitch},
		0x1feb: {Symbol: "BANK11", Action: mapper.HotspotBankSwitch},
		0x1fec: {Symbol: "BANK12", Action: mapper.HotspotBankSwitch},
		0x1fed: {Symbol: "BANK13", Action: mapper.HotspotBankSwitch},
		0x1fee: {Symbol: "BANK14", Action: mapper.HotspotBankSwitch},
		0x1fef: {Symbol: "BANK15", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *ef) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for the CBS cartridge.
type efState struct {
	// identifies the currently selected bank
	bank int
}

func newEFstate() *efState {
	return &efState{}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *efState) Snapshot() *efState {
	n := *s
	return &n
}
