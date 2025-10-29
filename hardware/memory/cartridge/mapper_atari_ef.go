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
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// the EF mapper is really just a 64k standard atari cartridge
type ef struct {
	atari
}

// newEF is the preferred method of initialisation for the ef type
func newEF(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("EF: %w", err)
	}

	cart := &ef{
		atari: atari{
			env:            env,
			bankSize:       4096,
			mappingID:      "EF",
			needsSuperchip: hasSuperchip(data),
			state:          newAtariState(),
		},
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("EF: wrong number of bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, cart.NumBanks())
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *ef) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *ef) Plumb(env *environment.Environment) {
	cart.env = env
}

// Access implements the mapper.CartMapper interface.
func (cart *ef) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if data, mask, ok := cart.atari.access(addr); ok {
		return data, mask, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *ef) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	return cart.accessVolatile(addr, data, poke)
}

// bankswitch on hotspot access.
func (cart *ef) bankswitch(addr uint16) bool {
	if addr >= 0x0fe0 && addr <= 0x0fef {
		cart.state.bank = int(addr & 0x000f)
		return true
	}
	return false
}

// Reset implements the mapper.CartMapper interface.
func (cart *ef) Reset() error {
	return cart.reset()
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *ef) NumBanks() int {
	return 16
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
