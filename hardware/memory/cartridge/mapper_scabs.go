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
)

// scabs implements the mapper.CartMapper interface.
//
// the function of the scheme is specified European Patent 84300730.3
// https://worldwide.espacenet.com/patent/search/family/023848640/publication/EP0116455A2?q=84300730.3
type scabs struct {
	env       *environment.Environment
	mappingID string
	bankSize  int
	banks     [2][]uint8
	state     *scabsState
}

func newSCABS(env *environment.Environment, data []byte) (mapper.CartMapper, error) {
	cart := &scabs{
		env:       env,
		mappingID: "FE",
		bankSize:  4096,
		state:     &scabsState{},
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("FE: wrong number of bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *scabs) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *scabs) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *scabs) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *scabs) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *scabs) Reset() {
	cart.state.bank = 0
}

// Access implements the mapper.CartMapper interface.
func (cart *scabs) Access(addr uint16, peek bool) (uint8, uint8, error) {
	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *scabs) AccessVolatile(_ uint16, _ uint8, _ bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *scabs) NumBanks() int {
	return 2
}

// GetBank implements the mapper.CartMapper interface.
func (cart *scabs) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *scabs) Patch(_ int, _ uint8) error {
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *scabs) AccessPassive(addr uint16, data uint8) {
	// "[...] it will be noted that JSR instruction is always followed by an
	// address 01FE on the address bus. Once cycle thereafter the most
	// significant 8bits of the new memory location appears on the data bus.
	// Thus by monitoring the address bus for 01FE and then latching the most
	// significant bit on the data bus cycle thereafter, memory bank selection
	// can be implemented"
	//
	// Article 30 of European Patent 84300730.3

	switch cart.state.bankSwitch {
	case 2:
		cart.state.bankSwitch = 1
	case 1:
		switch data >> 5 {
		case 0b111:
			cart.state.bank = 0
		case 0b110:
			cart.state.bank = 1
		}
		cart.state.bankSwitch = 0
	default:
		if addr == 0x01fe {
			cart.state.bankSwitch = 2
		}
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *scabs) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *scabs) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

type scabsState struct {
	bank       int
	bankSwitch int
}

// Snapshot implements the mapper.CartMapper interface.
func (s *scabsState) Snapshot() *scabsState {
	n := *s
	return &n
}
