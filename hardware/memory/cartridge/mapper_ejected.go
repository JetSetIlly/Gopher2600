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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// ejected implements the mapper.CartMapper interface.
type ejected struct {
}

func newEjected() *ejected {
	return &ejected{}
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *ejected) MappedBanks() string {
	return "ejected"
}

// ID implements the mapper.CartMapper interface.
func (cart *ejected) ID() string {
	return "-"
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *ejected) Snapshot() mapper.CartMapper {
	return &ejected{}
}

// Plumb implements the mapper.CartMapper interface.
func (cart *ejected) Plumb() {
}

// Reset implements the mapper.CartMapper interface.
func (cart *ejected) Reset() {
}

// Access implements the mapper.CartMapper interface.
func (cart *ejected) Access(_ uint16, _ bool) (uint8, uint8, error) {
	// return undriven pins
	return 0, 0, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *ejected) AccessVolatile(_ uint16, _ uint8, _ bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *ejected) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *ejected) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *ejected) Patch(_ int, _ uint8) error {
	return Ejected
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *ejected) AccessPassive(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *ejected) Step(_ float32) {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *ejected) CopyBanks() []mapper.BankContent {
	return nil
}
