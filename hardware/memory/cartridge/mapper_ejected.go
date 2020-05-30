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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package cartridge

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const ejectedName = "ejected"
const ejectedHash = "nohash"
const ejectedMethod = "ejected"

// ejected implements the cartMapper interface.

type ejected struct {
	description string
}

func newEjected() *ejected {
	cart := &ejected{}
	cart.description = ejectedName
	cart.Initialise()
	return cart
}

func (cart ejected) String() string {
	return cart.description
}

func (cart ejected) ID() string {
	return "-"
}

func (cart *ejected) Initialise() {
}

func (cart *ejected) Read(addr uint16) (uint8, error) {
	return 0, errors.New(errors.CartridgeEjected)
}

func (cart *ejected) Write(addr uint16, data uint8) error {
	return errors.New(errors.CartridgeEjected)
}

func (cart ejected) NumBanks() int {
	return 0
}

func (cart *ejected) SetBank(addr uint16, bank int) error {
	return errors.New(errors.CartridgeError, fmt.Sprintf("ejected cartridge"))
}

func (cart ejected) GetBank(addr uint16) int {
	return 0
}

func (cart *ejected) SaveState() interface{} {
	return nil
}

func (cart *ejected) RestoreState(state interface{}) error {
	return nil
}

func (cart *ejected) Poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *ejected) Patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.description)
}

func (cart *ejected) Listen(addr uint16, data uint8) {
}

func (cart *ejected) Step() {
}

func (cart ejected) GetRAM() []memorymap.SubArea {
	return nil
}
