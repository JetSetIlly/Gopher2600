package cartridge

import (
	"fmt"
	"gopher2600/errors"
)

const ejectedName = "ejected"
const ejectedHash = "nohash"
const ejectedMethod = "ejected"

// ejected implements the cartMapper interface.

type ejected struct {
	method string
}

func newEjected() *ejected {
	cart := &ejected{method: ejectedMethod}
	cart.initialise()
	return cart
}

func (cart ejected) String() string {
	return cart.method
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
	return errors.New(errors.CartridgeError, fmt.Sprintf("invalid bank (%d) for cartridge type (%s)", bank, cart.method))
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

func (cart ejected) ram() []uint8 {
	return []uint8{}
}

func (cart *ejected) listen(addr uint16, data uint8) {
}

func (cart *ejected) poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *ejected) patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.method)
}
