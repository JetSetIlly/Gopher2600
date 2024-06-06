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

// bank switching used format for tarzan. very basic and we use the atari mapper
// as a base implementation
//
// discussion here:
//
//	https://forums.atariage.com/topic/367498-atari-2600-tarzan-released/page/3/#comment-5481003
type jane struct {
	atari
}

func newJANE(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("JANE: %w", err)
	}

	cart := &jane{
		atari: atari{
			env:       env,
			bankSize:  4096,
			mappingID: "JANE",
			state:     newAtariState(),
		},
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("JANE: wrong number of bytes in the cartridge data")
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
func (cart *jane) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *jane) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *jane) Reset() {
	cart.reset(cart.NumBanks())
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *jane) NumBanks() int {
	return 4
}

// Access implements the mapper.CartMapper interface.
func (cart *jane) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if data, mask, ok := cart.atari.access(addr); ok {
		return data, mask, nil
	}

	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *jane) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr, data) {
			return nil
		}
	}

	return cart.atari.accessVolatile(addr, data, poke)
}

// bankswitch on hotspot access
func (cart *jane) bankswitch(addr uint16, data uint8) bool {
	if addr == 0x0ff0 {
		cart.state.bank = 0
		return true
	} else if addr == 0x0ff1 {
		cart.state.bank = 1
		return true
	} else if addr == 0x0ff8 {
		cart.state.bank = 2
		return true
	} else if addr == 0x0ff9 {
		cart.state.bank = 3
		return true
	}
	return false
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *jane) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff0: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff1: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *jane) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}
