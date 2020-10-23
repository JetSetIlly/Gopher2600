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
	"math/rand"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// ejected implements the cartMapper interface.
type ejected struct {
}

func newEjected() *ejected {
	return &ejected{}
}

func (cart ejected) String() string {
	return "ejected"
}

// ID implements the cartMapper interface.
func (cart ejected) ID() string {
	return "-"
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *ejected) Snapshot() mapper.CartSnapshot {
	return nil
}

// Plumb implements the mapper.CartMapper interface.
func (cart *ejected) Plumb(s mapper.CartSnapshot) {
}

// Reset implements the cartMapper interface.
func (cart *ejected) Reset(_ *rand.Rand) {
}

// Read implements the cartMapper interface.
func (cart *ejected) Read(_ uint16, _ bool) (uint8, error) {
	return 0, curated.Errorf(Ejected)
}

// Write implements the cartMapper interface.
func (cart *ejected) Write(_ uint16, _ uint8, _, _ bool) error {
	return curated.Errorf(Ejected)
}

// NumBanks implements the cartMapper interface.
func (cart ejected) NumBanks() int {
	return 0
}

// GetBank implements the cartMapper interface.
func (cart ejected) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the cartMapper interface.
func (cart *ejected) Patch(_ int, _ uint8) error {
	return curated.Errorf(Ejected)
}

// Listen implements the cartMapper interface.
func (cart *ejected) Listen(_ uint16, _ uint8) {
}

// Step implements the cartMapper interface.
func (cart *ejected) Step() {
}

// IterateBank implements the mapper.CartMapper interface.
func (cart ejected) CopyBanks() []mapper.BankContent {
	return nil
}
