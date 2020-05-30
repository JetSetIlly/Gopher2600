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

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// cdfj implements the cartMapper interface.
type cdfj struct {
	mappingID   string
	description string

	bank int
}

func newDPC(data []byte) (*cdfj, error) {
	const bankSize = 4096
	const gfxSize = 2048

	cart := &cdfj{}
	cart.mappingID = "CDFJ"
	cart.description = "CDFJ (Harmony emulation)"

	cart.Initialise()

	return cart, nil
}

func (cart cdfj) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.mappingID, cart.bank)
}

func (cart cdfj) ID() string {
	return cart.mappingID
}

func (cart *cdfj) Initialise() {
	cart.bank = 0
}

func (cart *cdfj) Read(addr uint16) (uint8, error) {
	return 0, nil
}

func (cart *cdfj) Write(addr uint16, data uint8) error {
	return nil
}

func (cart cdfj) NumBanks() int {
	return 2
}

func (cart *cdfj) SetBank(addr uint16, bank int) error {
	cart.bank = bank
	return nil
}

func (cart cdfj) GetBank(addr uint16) int {
	return cart.bank
}

func (cart *cdfj) SaveState() interface{} {
	return nil
}

func (cart *cdfj) RestoreState(state interface{}) error {
	return nil
}

func (cart *cdfj) Poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *cdfj) Patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.description)
}

func (cart *cdfj) Listen(addr uint16, data uint8) {
}

func (cart *cdfj) Step() {
}

func (cart cdfj) GetRAM() []memorymap.SubArea {
	return nil
}

// StaticRead implements the StaticArea interface
func (cart cdfj) StaticRead(addr uint16) (uint8, error) {
	return 0, nil
}

// StaticWrite implements the StaticArea interface
func (cart *cdfj) StaticWrite(addr uint16, data uint8) error {
	return nil
}

// StaticSize implements the StaticArea interface
func (cart cdfj) StaticSize() int {
	return 0
}
