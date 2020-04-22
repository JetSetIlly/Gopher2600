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
	cart.initialise()
	return cart
}

func (cart ejected) String() string {
	return cart.description
}

func (cart ejected) format() string {
	return "-"
}

func (cart *ejected) initialise() {
}

func (cart *ejected) read(addr uint16) (uint8, error) {
	return 0, errors.New(errors.CartridgeEjected)
}

func (cart *ejected) write(addr uint16, data uint8) error {
	return errors.New(errors.CartridgeEjected)
}

func (cart ejected) numBanks() int {
	return 0
}

func (cart *ejected) setBank(addr uint16, bank int) error {
	return errors.New(errors.CartridgeError, fmt.Sprintf("ejected cartridge"))
}

func (cart ejected) getBank(addr uint16) int {
	return 0
}

func (cart *ejected) saveState() interface{} {
	return nil
}

func (cart *ejected) restoreState(state interface{}) error {
	return nil
}

func (cart *ejected) listen(addr uint16, data uint8) {
}

func (cart *ejected) poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *ejected) patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.description)
}

func (cart ejected) getRAMinfo() []RAMinfo {
	return nil
}

func (cart *ejected) step() {
}
