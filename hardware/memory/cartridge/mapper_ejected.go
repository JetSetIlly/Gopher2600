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
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
)

const ejectedName = "ejected"
const ejectedHash = "nohash"
const ejectedMethod = "ejected"

// ejected implements the cartMapper interface.

type ejected struct {
	description string
}

func newEjected() *ejected {
	cart := &ejected{
		description: "ejected",
	}
	cart.Initialise()
	return cart
}

func (cart ejected) String() string {
	return cart.description
}

// ID implements the cartMapper interface
func (cart ejected) ID() string {
	return "-"
}

// Initialise implements the cartMapper interface
func (cart *ejected) Initialise() {
}

// Read implements the cartMapper interface
func (cart *ejected) Read(_ uint16, _ bool) (uint8, error) {
	return 0, errors.Errorf(Ejected)
}

// Write implements the cartMapper interface
func (cart *ejected) Write(_ uint16, _ uint8, _, _ bool) error {
	return errors.Errorf(Ejected)
}

// NumBanks implements the cartMapper interface
func (cart ejected) NumBanks() int {
	return 0
}

// GetBank implements the cartMapper interface
func (cart ejected) GetBank(_ uint16) banks.Details {
	return banks.Details{Number: 0, IsRAM: false}
}

// Patch implements the cartMapper interface
func (cart *ejected) Patch(_ int, _ uint8) error {
	return errors.Errorf(Ejected)
}

// Listen implements the cartMapper interface
func (cart *ejected) Listen(_ uint16, _ uint8) {
}

// Step implements the cartMapper interface
func (cart *ejected) Step() {
}

// IterateBank implemnts the disassemble interface
func (cart ejected) IterateBanks(prev *banks.Content) *banks.Content {
	return nil
}
