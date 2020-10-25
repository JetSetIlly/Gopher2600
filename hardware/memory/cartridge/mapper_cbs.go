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

// from bankswitch_sizes.txt:
//
// 12K:
//
//  -FA: Used only by CBS.  Similar to F8, except you have three 4K banks
//  instead of two.  You select the desired bank via 1FF8, 1FF9, and 1FFA.
//  These carts also have 256 bytes of RAM mapped in at 1000-11FF.  1000-10FF
//  is the write port while 1100-11FF is the read port.
//
//
// cartridges:
//	- Omega Race
//	- Gorf
type cbs struct {
	mappingID   string
	description string

	// cbs cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *cbsState
}

func newCBS(data []byte) (mapper.CartMapper, error) {
	cart := &cbs{
		mappingID:   "FA",
		description: "CBS",
		bankSize:    4096,
		state:       newCbsState(),
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("FA: %v", "wrong number bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

func (cart cbs) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.description, cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart cbs) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *cbs) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *cbs) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*cbsState)
}

// Reset implements the cartMapper interface.
func (cart *cbs) Reset(randSrc *rand.Rand) {
	for i := range cart.state.ram {
		if randSrc != nil {
			cart.state.ram[i] = uint8(randSrc.Intn(0xff))
		} else {
			cart.state.ram[i] = 0
		}
	}

	cart.state.bank = len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *cbs) Read(addr uint16, passive bool) (uint8, error) {
	if addr >= 0x0100 && addr <= 0x01ff {
		return cart.state.ram[addr-0x100], nil
	}

	cart.bankswitch(addr, passive)

	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *cbs) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	if addr <= 0x00ff {
		cart.state.ram[addr] = data
		return nil
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("FA: %v", curated.Errorf(bus.AddressError, addr))
}

// bankswitch on hotspot access.
func (cart *cbs) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0ff8 && addr <= 0xffa {
		if passive {
			return true
		}
		if addr == 0x0ff8 {
			cart.state.bank = 0
		} else if addr == 0x0ff9 {
			cart.state.bank = 1
		} else if addr == 0x0ffa {
			cart.state.bank = 2
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart cbs) NumBanks() int {
	return 3
}

// GetBank implements the mapper.CartMapper interface.
func (cart cbs) GetBank(addr uint16) mapper.BankInfo {
	// cbs cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: addr <= 0x00ff}
}

// Patch implements the mapper.CartMapper interface.
func (cart *cbs) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("FA: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *cbs) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *cbs) Step() {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart cbs) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "CBS+RAM",
		Origin: 0x1080,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *cbs) PutRAM(_ int, idx int, data uint8) {
	cart.state.ram[idx] = data
}

// IterateBank implements the mapper.CartMapper interface.
func (cart cbs) CopyBanks() []mapper.BankContent {
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
func (cart cbs) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart cbs) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for the CBS cartridge.
type cbsState struct {
	// identifies the currently selected bank
	bank int

	// some atari ROMs support aditional RAM. this is sometimes referred to as
	// the superchip. ram is only added when it is detected
	ram []uint8
}

func newCbsState() *cbsState {
	const cbsRAMsize = 256

	return &cbsState{
		ram: make([]uint8, cbsRAMsize),
	}
}

// Snapshot implements the mapper.CartSnapshot interface.
func (s *cbsState) Snapshot() mapper.CartSnapshot {
	n := *s
	n.ram = make([]uint8, len(s.ram))
	copy(n.ram, s.ram)
	return &n
}
