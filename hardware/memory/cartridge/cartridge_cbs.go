package cartridge

import (
	"fmt"
	"gopher2600/errors"
)

type cbs struct {
	method string

	// cbs cartridges have 3 banks of 4096 bytes
	banks [][]uint8

	// identifies the currently selected bank
	bank int

	// superchip contains the cartridge RAM. not sure if it was actually called
	// the superchip in CBS cartridges but
	superchip []uint8
}

func newCBS(data []byte) (cartMapper, error) {
	const bankSize = 4096

	cart := &cbs{}
	cart.method = "CBS (FA)"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, "not enough bytes in the cartridge file")
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		copy(cart.banks[k], data[offset:offset+bankSize])
	}

	// 256 bytes of cartidge ram in all instances
	cart.superchip = make([]uint8, 256)

	cart.initialise()

	return cart, nil
}

func (cart cbs) String() string {
	return fmt.Sprintf("%s bank: %d", cart.method, cart.bank)
}

func (cart *cbs) initialise() {
	cart.bank = len(cart.banks) - 1
	for i := range cart.superchip {
		cart.superchip[i] = 0x00
	}
}

func (cart *cbs) read(addr uint16) (uint8, error) {
	if addr >= 0x0100 && addr <= 0x01ff {
		return cart.superchip[addr-0x100], nil
	}

	data := cart.banks[cart.bank][addr]

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else if addr == 0x0ffa {
		cart.bank = 2
	}

	return data, nil
}

func (cart *cbs) write(addr uint16, data uint8) error {
	if addr <= 0x00ff {
		cart.superchip[addr] = data
		return nil
	}

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else if addr == 0x0ffa {
		cart.bank = 2
	} else {
		return errors.New(errors.UnwritableAddress, addr)
	}

	return nil
}

func (cart cbs) getBank(addr uint16) int {
	// cbs cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return cart.bank
}

func (cart *cbs) setBank(addr uint16, bank int) error {
	if bank < 0 || bank > len(cart.banks) {
		return errors.New(errors.CartridgeError, fmt.Sprintf("invalid bank (%d) for cartridge type (%s)", bank, cart.method))
	}
	cart.bank = bank
	return nil
}

func (cart *cbs) saveState() interface{} {
	superchip := make([]uint8, len(cart.superchip))
	copy(superchip, cart.superchip)
	return []interface{}{cart.bank, superchip}
}

func (cart *cbs) restoreState(state interface{}) error {
	cart.bank = state.([]interface{})[0].(int)
	copy(cart.superchip, state.([]interface{})[1].([]uint8))
	return nil
}

func (cart cbs) numBanks() int {
	return 3
}

func (cart cbs) ram() []uint8 {
	return cart.superchip
}

func (cart cbs) listen(addr uint16, data uint8) {
}
