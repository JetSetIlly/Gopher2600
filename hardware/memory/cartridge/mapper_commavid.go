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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// from http://www.taswegian.com/WoodgrainWizard/tiki-index.php?page=CV
//
// $F000-$F3FF read from RAM
// $F400-$F7FF write to RAM
// $F800-$FFFF ROM
//
// this seems to be the same principle as other simple RAM cartridges, like the
// Atari Superchip or CBS cartridges
//
// from bankswitch_sizes.txt:
//
// cartridges:
//   - MagiCard
//   - Video Lfe
//
// and the reason why I implemented this format for Gopher2600:
//
// https://forums.atariage.com/topic/342021-new-512-bytes-demo-released-gar-nix/
type commavid struct {
	env *environment.Environment

	mappingID string

	bankSize int
	bankData []uint8

	// rewindable state
	state *commavidState
}

func newCommaVid(env *environment.Environment, data []byte) (mapper.CartMapper, error) {
	cart := &commavid{
		env:       env,
		bankSize:  4096,
		mappingID: "CV",
		state:     newCommaVidState(),
	}

	cart.bankData = make([]uint8, cart.bankSize)
	if len(data) < 2048 {
		// place undersized binaries at the end of memory
		copy(cart.bankData[4096-len(data):], data)
		logger.Logf("CV", "placing undersized commavid data at end of cartridge memory")
	} else if len(data) == 2048 {
		copy(cart.bankData[2048:], data[:2048])
		logger.Logf("CV", "placing 2k commavid data at end of cartridge memory")
	} else {
		return nil, fmt.Errorf("CV: unhandled size for commavid cartridges (%d)", len(data))
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *commavid) MappedBanks() string {
	return fmt.Sprintf("Bank: 0")
}

// ID implements the mapper.CartMapper interface.
func (cart *commavid) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *commavid) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *commavid) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *commavid) Reset() {
	// always starting with random state. this is because Video Life, one of
	// the few original CommaVid cartridges, expects it for the opening effect
	for i := range cart.state.ram {
		cart.state.ram[i] = uint8(cart.env.Random.NoRewind(0xff))
	}
}

// Access implements the mapper.CartMapper interface.
func (cart *commavid) Access(addr uint16, _ bool) (uint8, uint8, error) {
	if addr >= 0x0000 && addr <= 0x03ff {
		return cart.state.ram[addr], mapper.CartDrivenPins, nil
	}
	if addr >= 0x0400 && addr <= 0x07ff {
		return 0, 0, nil
	}

	return cart.bankData[addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *commavid) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if addr >= 0x0400 && addr <= 0x07ff {
		cart.state.ram[addr&0x03ff] = data
		return nil
	}

	if poke {
		cart.bankData[addr] = data
		return nil
	}

	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *commavid) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *commavid) GetBank(addr uint16) mapper.BankInfo {
	// commavid cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: 0, IsRAM: addr <= 0x07ff}
}

// Patch implements the mapper.CartPatchable interface
func (cart *commavid) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize {
		return fmt.Errorf("CV: patch offset too high (%d)", offset)
	}
	cart.bankData[offset] = data
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *commavid) AccessPassive(addr uint16, data uint8) error {
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *commavid) Step(_ float32) {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *commavid) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "CommaVid",
		Origin: 0x0,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *commavid) PutRAM(_ int, idx int, data uint8) {
	cart.state.ram[idx] = data
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *commavid) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, 1)
	c[0] = mapper.BankContent{Number: 0,
		Data:    cart.bankData,
		Origins: []uint16{memorymap.OriginCart},
	}
	return c
}

// rewindable state for commavid cartridges
type commavidState struct {
	// commavid cartridges are distinguished for having 1k of onboard ram
	ram []uint8
}

func newCommaVidState() *commavidState {
	const commavidRAMsize = 1024

	return &commavidState{
		ram: make([]uint8, commavidRAMsize),
	}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *commavidState) Snapshot() *commavidState {
	n := *s
	n.ram = make([]uint8, len(s.ram))
	copy(n.ram, s.ram)
	return &n
}
