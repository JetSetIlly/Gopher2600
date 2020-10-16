// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be usdful,
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
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type df struct {
	mappingID   string
	description string

	// df cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// identifies the currently selected bank
	bank int

	// df cartridges always have a RAM area
	ram []uint8
}

func newDF(data []byte) (mapper.CartMapper, error) {
	cart := &df{
		mappingID:   "DF",
		description: "128KB",
		bankSize:    4096,
		ram:         make([]uint8, 256),
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("%s: wrong number of bytes in the cartridge data", cart.mappingID)
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	cart.Initialise()

	return cart, nil
}

func (cart df) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.description, cart.bank)
}

// ID implements the mapper.CartMapper interface
func (cart df) ID() string {
	return cart.mappingID
}

// Initialise implements the mapper.CartMapper interface
func (cart *df) Initialise() {
	cart.bank = 15

	for i := range cart.ram {
		cart.ram[i] = 0x00
	}
}

// Read implements the mapper.CartMapper interface
func (cart *df) Read(addr uint16, passive bool) (uint8, error) {
	if addr >= 0x0080 && addr <= 0x00ff {
		return cart.ram[addr-0x80], nil
	}

	cart.bankswitch(addr, passive)

	return cart.banks[cart.bank][addr], nil
}

// Write implements the mapper.CartMapper interface
func (cart *df) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	if addr <= 0x007f {
		cart.ram[addr] = data
		return nil
	}

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return curated.Errorf(bus.AddressError, addr)
}

// bankswitch on hotspot access
func (cart *df) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0fc0 && addr <= 0xfdf {
		if passive {
			return true
		}

		// looking at this switch, I'm now thinking hotspots could be done
		// programmatically. for now though, we'll keep it like this.
		if addr == 0x0fc0 {
			cart.bank = 0
		} else if addr == 0x0fc1 {
			cart.bank = 1
		} else if addr == 0x0fc2 {
			cart.bank = 2
		} else if addr == 0x0fc3 {
			cart.bank = 3
		} else if addr == 0x0fc4 {
			cart.bank = 4
		} else if addr == 0x0fc5 {
			cart.bank = 5
		} else if addr == 0x0fc6 {
			cart.bank = 6
		} else if addr == 0x0fc7 {
			cart.bank = 7
		} else if addr == 0x0fc8 {
			cart.bank = 8
		} else if addr == 0x0fc9 {
			cart.bank = 9
		} else if addr == 0x0fca {
			cart.bank = 10
		} else if addr == 0x0fcb {
			cart.bank = 11
		} else if addr == 0x0fcc {
			cart.bank = 12
		} else if addr == 0x0fcd {
			cart.bank = 13
		} else if addr == 0x0fce {
			cart.bank = 14
		} else if addr == 0x0fcf {
			cart.bank = 15
		} else if addr == 0x0fd0 {
			cart.bank = 16
		} else if addr == 0x0fd1 {
			cart.bank = 17
		} else if addr == 0x0fd2 {
			cart.bank = 18
		} else if addr == 0x0fd3 {
			cart.bank = 19
		} else if addr == 0x0fd4 {
			cart.bank = 20
		} else if addr == 0x0fd5 {
			cart.bank = 21
		} else if addr == 0x0fd6 {
			cart.bank = 22
		} else if addr == 0x0fd7 {
			cart.bank = 23
		} else if addr == 0x0fd8 {
			cart.bank = 24
		} else if addr == 0x0fd9 {
			cart.bank = 25
		} else if addr == 0x0fda {
			cart.bank = 26
		} else if addr == 0x0fdb {
			cart.bank = 27
		} else if addr == 0x0fdc {
			cart.bank = 28
		} else if addr == 0x0fdd {
			cart.bank = 29
		} else if addr == 0x0fde {
			cart.bank = 30
		} else if addr == 0x0fdf {
			cart.bank = 31
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface
func (cart df) NumBanks() int {
	return 32
}

// GetBank implements the mapper.CartMapper interface
func (cart df) GetBank(addr uint16) mapper.BankInfo {
	// df cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.bank, IsRAM: addr <= 0x00ff}
}

// Patch implements the mapper.CartMapper interface
func (cart *df) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("%s: patch offset too high (%v)", cart.ID(), offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface
func (cart *df) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface
func (cart *df) Step() {
}

// GetRAM implements the mapper.CartRAMBus interface
func (cart df) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "df+RAM",
		Origin: 0x1080,
		Data:   make([]uint8, len(cart.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface
func (cart *df) PutRAM(_ int, idx int, data uint8) {
	cart.ram[idx] = data
}

// IterateBank implements the mapper.CartMapper interface
func (cart df) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// ReadHotspots implements the mapper.CartHotspotsBus interface
func (cart df) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x0fc0: mapper.CartHotspotInfo{Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x0fc1: mapper.CartHotspotInfo{Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x0fc2: mapper.CartHotspotInfo{Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x0fc3: mapper.CartHotspotInfo{Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x0fc4: mapper.CartHotspotInfo{Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x0fc5: mapper.CartHotspotInfo{Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x0fc6: mapper.CartHotspotInfo{Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
		0x0fc7: mapper.CartHotspotInfo{Symbol: "BANK7", Action: mapper.HotspotBankSwitch},
		0x0fc8: mapper.CartHotspotInfo{Symbol: "BANK8", Action: mapper.HotspotBankSwitch},
		0x0fc9: mapper.CartHotspotInfo{Symbol: "BANK9", Action: mapper.HotspotBankSwitch},
		0x0fca: mapper.CartHotspotInfo{Symbol: "BANK10", Action: mapper.HotspotBankSwitch},
		0x0fcb: mapper.CartHotspotInfo{Symbol: "BANK11", Action: mapper.HotspotBankSwitch},
		0x0fcc: mapper.CartHotspotInfo{Symbol: "BANK12", Action: mapper.HotspotBankSwitch},
		0x0fcd: mapper.CartHotspotInfo{Symbol: "BANK13", Action: mapper.HotspotBankSwitch},
		0x0fce: mapper.CartHotspotInfo{Symbol: "BANK14", Action: mapper.HotspotBankSwitch},
		0x0fcf: mapper.CartHotspotInfo{Symbol: "BANK15", Action: mapper.HotspotBankSwitch},
		0x0fd0: mapper.CartHotspotInfo{Symbol: "BANK16", Action: mapper.HotspotBankSwitch},
		0x0fd1: mapper.CartHotspotInfo{Symbol: "BANK17", Action: mapper.HotspotBankSwitch},
		0x0fd2: mapper.CartHotspotInfo{Symbol: "BANK18", Action: mapper.HotspotBankSwitch},
		0x0fd3: mapper.CartHotspotInfo{Symbol: "BANK19", Action: mapper.HotspotBankSwitch},
		0x0fd4: mapper.CartHotspotInfo{Symbol: "BANK20", Action: mapper.HotspotBankSwitch},
		0x0fd5: mapper.CartHotspotInfo{Symbol: "BANK21", Action: mapper.HotspotBankSwitch},
		0x0fd6: mapper.CartHotspotInfo{Symbol: "BANK22", Action: mapper.HotspotBankSwitch},
		0x0fd7: mapper.CartHotspotInfo{Symbol: "BANK23", Action: mapper.HotspotBankSwitch},
		0x0fd8: mapper.CartHotspotInfo{Symbol: "BANK24", Action: mapper.HotspotBankSwitch},
		0x0fd9: mapper.CartHotspotInfo{Symbol: "BANK25", Action: mapper.HotspotBankSwitch},
		0x0fda: mapper.CartHotspotInfo{Symbol: "BANK26", Action: mapper.HotspotBankSwitch},
		0x0fdb: mapper.CartHotspotInfo{Symbol: "BANK27", Action: mapper.HotspotBankSwitch},
		0x0fdc: mapper.CartHotspotInfo{Symbol: "BANK28", Action: mapper.HotspotBankSwitch},
		0x0fdd: mapper.CartHotspotInfo{Symbol: "BANK29", Action: mapper.HotspotBankSwitch},
		0x0fde: mapper.CartHotspotInfo{Symbol: "BANK30", Action: mapper.HotspotBankSwitch},
		0x0fdf: mapper.CartHotspotInfo{Symbol: "BANK31", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface
func (cart df) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}
