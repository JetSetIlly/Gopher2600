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

// atari8k variant (WF8):
//   - Smurf (prototype)
//
// discussion here:
//
//	https://forums.atariage.com/topic/367157-smurf-rescue-alternative-rom-with-wf8-bankswitch-format/
type wf8 struct {
	atari
}

func newWF8(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("WF8: %w", err)
	}

	cart := &wf8{
		atari: atari{
			env:            env,
			bankSize:       4096,
			mappingID:      "WF8",
			needsSuperchip: hasSuperchip(data),
			state:          newAtariState(),
		},
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("WF8: wrong number of bytes in the cartridge data")
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
func (cart *wf8) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *wf8) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *wf8) Reset() error {
	return cart.reset()
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *wf8) NumBanks() int {
	return 2
}

// Access implements the mapper.CartMapper interface.
func (cart *wf8) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if data, mask, ok := cart.atari.access(addr); ok {
		return data, mask, nil
	}

	// unlike normal F8

	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *wf8) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr, data) {
			return nil
		}
	}

	return cart.atari.accessVolatile(addr, data, poke)
}

// bankswitch on hotspot access
func (cart *wf8) bankswitch(addr uint16, data uint8) bool {
	// WF8 differs from in that there is only one hotspot address and that the
	// target bank is discerned from the data bus
	if addr == 0x0ff8 {
		if data&0x04 == 0x04 {
			cart.state.bank = 1
		} else {
			cart.state.bank = 0
		}
		return true
	}
	return false
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *wf8) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *wf8) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}
