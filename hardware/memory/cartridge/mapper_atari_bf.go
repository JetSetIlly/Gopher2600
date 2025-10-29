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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// the BF mapper is an concpetual extension of the EF mapper. therefore, it is
// really just a 256k standard atari cartridge
type bf struct {
	atari
}

// newBF is the preferred method of initialisation for the ef type
func newBF(env *environment.Environment) (mapper.CartMapper, error) {
	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("BF: %w", err)
	}

	cart := &bf{
		atari: atari{
			env:            env,
			bankSize:       4096,
			mappingID:      "BF",
			needsSuperchip: hasSuperchip(data),
			state:          newAtariState(),
		},
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("BF: wrong number of bytes in the cartridge data")
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
func (cart *bf) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *bf) Plumb(env *environment.Environment) {
	cart.env = env
}

// Access implements the mapper.CartMapper interface.
func (cart *bf) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if data, mask, ok := cart.atari.access(addr); ok {
		return data, mask, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *bf) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	return cart.accessVolatile(addr, data, poke)
}

// bankswitch on hotspot access.
func (cart *bf) bankswitch(addr uint16) bool {
	if addr >= 0x0f80 && addr <= 0x0fbf {
		cart.state.bank = int(addr - 0x0f80)
		return true
	}
	return false
}

// Reset implements the mapper.CartMapper interface.
func (cart *bf) Reset() error {
	return cart.reset()
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *bf) NumBanks() int {
	return 64
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *bf) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *bf) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}
