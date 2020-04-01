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

// from bankswitch_sizes.txt:
//
// 2K:
//
// -These carts are not bankswitched, however the data repeats twice in the
// 4K address space.  You'll need to manually double-up these images to 4K
// if you want to put these in say, a 4K cart.
//
// 4K:
//
// -These images are not bankswitched.
//
// 8K:
//
// -F8: This is the 'standard' method to implement 8K carts.  There are two
// addresses which select between two unique 4K sections.  They are 1FF8
// and 1FF9.  Any access to either one of these locations switches banks.
// Accessing 1FF8 switches in the first 4K, and accessing 1FF9 switches in
// the last 4K.  Note that you can only access one 4K at a time!
//
// 16K:
//
// -F6: The 'standard' method for implementing 16K of data.  It is identical
// to the F8 method above, except there are 4 4K banks.  You select which
// 4K bank by accessing 1FF6, 1FF7, 1FF8, and 1FF9.
//
// 32K:
//
// -F4: The 'standard' method for implementing 32K.  Only one cart is known
// to use it- Fatal Run.  Like the F6 method, however there are 8 4K
// banks instead of 4.  You use 1FF4 to 1FFB to select the desired bank.
//
//
// Some carts have extra RAM; There are three known formats for this:
//
// Atari's 'Super Chip' is nothing more than a 128-byte RAM chip that maps
// itsself in the first 256 bytes of cart memory.  (1000-10FFh)
// The first 128 bytes is the write port, while the second 128 bytes is the
// read port.  This is needed, because there is no R/W line to the cart.

type atari struct {
	formatID    string
	description string

	bankSize int

	// atari formats apart from 2k and 4k are divided into banks. 2k and 4k
	// ROMs conceptually have one bank
	banks [][]uint8

	// identifies the currently selected bank
	bank int

	// some ROMs support aditional RAM. in these instances the first 128 bytes
	// of each bank is mapped to RAM. this is sometimes referred to as the
	// superchip
	superchip []uint8

	// ram details
	ramInfo []RAMinfo
}

func (cart atari) String() string {
	if len(cart.banks) == 1 {
		return cart.description
	}
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.formatID, cart.bank)
}

func (cart atari) format() string {
	return cart.formatID
}

func (cart *atari) initialise() {
	cart.bank = 0
	for i := range cart.superchip {
		cart.superchip[i] = 0x00
	}
}

func (cart atari) getBank(addr uint16) int {
	// because atari bank switching swaps out the entire memory space, every
	// address points to whatever the current bank is. compare to parker bros.
	// cartridges.
	return cart.bank
}

func (cart *atari) setBank(addr uint16, bank int) error {
	if bank < 0 || bank >= len(cart.banks) {
		return errors.New(errors.CartridgeError, fmt.Sprintf("%s: invalid bank [%d]", cart.formatID, bank))
	}
	cart.bank = bank
	return nil
}

func (cart *atari) saveState() interface{} {
	superchip := make([]uint8, len(cart.superchip))
	copy(superchip, cart.superchip)
	return []interface{}{cart.bank, superchip}
}

func (cart *atari) restoreState(state interface{}) error {
	cart.bank = state.([]interface{})[0].(int)
	copy(cart.superchip, state.([]interface{})[1].([]uint8))
	return nil
}

func (cart *atari) read(addr uint16) (uint8, bool) {
	if cart.superchip != nil {
		if addr > 127 && addr < 256 {
			return cart.superchip[addr-128], true
		}
	}
	return 0, false
}

func (cart *atari) write(addr uint16, data uint8) bool {
	if cart.superchip != nil {
		if addr <= 127 {
			cart.superchip[addr] = data
			return true
		}
	}
	return false
}

func (cart *atari) addSuperchip() bool {
	// check for cartridge memory:
	//  - this method of detection simply checks whether the first 256 of each
	// bank are empty
	//  - I've guessed that this is a good method. if there's another one I
	// don't know about it.
	nullChar := cart.banks[0][0]
	for k := 0; k < len(cart.banks); k++ {
		for a := 0; a < 256; a++ {
			if cart.banks[k][a] != nullChar {
				return false
			}
		}
	}

	// allocate RAM
	cart.superchip = make([]uint8, 128)

	// update method string
	cart.description = fmt.Sprintf("%s (+ superchip RAM)", cart.description)

	cart.ramInfo = make([]RAMinfo, 1)
	cart.ramInfo[0] = RAMinfo{
		Label:       "Superchip",
		Active:      true,
		ReadOrigin:  0x1080,
		ReadMemtop:  0x10ff,
		WriteOrigin: 0x1000,
		WriteMemtop: 0x107f,
	}

	return true
}

func (cart *atari) listen(addr uint16, data uint8) {
}

func (cart *atari) poke(addr uint16, data uint8) error {
	cart.banks[cart.bank][addr] = data
	return nil
}

func (cart *atari) patch(addr uint16, data uint8) error {
	bank := int(addr) / cart.bankSize
	addr = addr % uint16(cart.bankSize)
	cart.banks[bank][addr] = data
	return nil
}

func (cart atari) getRAMinfo() []RAMinfo {
	return cart.ramInfo
}

// atari4k is the original and most straightforward format
//  o Pitfall
//  o River Raid
//  o Barnstormer
//  o etc.
type atari4k struct {
	atari
}

// this is a regular cartridge of 4096 bytes
//  o Pitfall
//  o Adventure
//  o Yars Revenge
//  o etc.
func newAtari4k(data []byte) (cartMapper, error) {
	cart := &atari4k{}
	cart.bankSize = 4096
	cart.description = "atari 4k"
	cart.formatID = "4k"
	cart.banks = make([][]uint8, 1)

	if len(data) != cart.bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	cart.banks[0] = make([]uint8, cart.bankSize)
	copy(cart.banks[0], data)

	cart.initialise()

	return cart, nil
}

func (cart atari4k) numBanks() int {
	return 1
}

func (cart *atari4k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}
	return cart.banks[0][addr], nil
}

func (cart *atari4k) write(addr uint16, data uint8) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	return errors.New(errors.BusError, addr)
}

// atari2k is the half-size cartridge of 2048 bytes
//	o Combat
//  o Dragster
//  o Outlaw
//	o Surround
//  o early cartridges
type atari2k struct {
	atari
}

func newAtari2k(data []byte) (cartMapper, error) {
	cart := &atari2k{}
	cart.bankSize = 2048
	cart.description = "atari 2k"
	cart.formatID = "2k"
	cart.banks = make([][]uint8, 1)

	if len(data) != cart.bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	cart.banks[0] = make([]uint8, cart.bankSize)
	copy(cart.banks[0], data)

	cart.initialise()

	return cart, nil
}

func (cart atari2k) numBanks() int {
	return 1
}

func (cart *atari2k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}
	return cart.banks[0][addr&0x07ff], nil
}

func (cart *atari2k) write(addr uint16, data uint8) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	return errors.New(errors.BusError, addr)
}

// atari8k (F8)
//	o ET
//  o Krull
//  o etc.
type atari8k struct {
	atari
}

func newAtari8k(data []uint8) (cartMapper, error) {
	cart := &atari8k{}
	cart.bankSize = 4096
	cart.description = "atari 8k"
	cart.formatID = "F8"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != cart.bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	cart.initialise()

	return cart, nil
}

func (cart atari8k) numBanks() int {
	return 2
}

func (cart *atari8k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.banks[cart.bank][addr]

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	}

	return data, nil
}

func (cart *atari8k) write(addr uint16, data uint8) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else {
		return errors.New(errors.BusError, addr)
	}

	return nil
}

// atari16k (F6)
//	o Crystal Castle
//	o RS Boxing
//  o Midnite Magic
//  o etc.
type atari16k struct {
	atari
}

func newAtari16k(data []byte) (cartMapper, error) {
	cart := &atari16k{}
	cart.bankSize = 4096
	cart.description = "atari 16k"
	cart.formatID = "F6"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != cart.bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	cart.initialise()

	return cart, nil
}

func (cart atari16k) numBanks() int {
	return 4
}

func (cart *atari16k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.banks[cart.bank][addr]

	if addr == 0x0ff6 {
		cart.bank = 0
	} else if addr == 0x0ff7 {
		cart.bank = 1
	} else if addr == 0x0ff8 {
		cart.bank = 2
	} else if addr == 0x0ff9 {
		cart.bank = 3
	}

	return data, nil
}

func (cart *atari16k) write(addr uint16, data uint8) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if addr == 0x0ff6 {
		cart.bank = 0
	} else if addr == 0x0ff7 {
		cart.bank = 1
	} else if addr == 0x0ff8 {
		cart.bank = 2
	} else if addr == 0x0ff9 {
		cart.bank = 3
	} else {
		return errors.New(errors.BusError, addr)
	}

	return nil
}

// atari32k (F8)
// o Fatal Run
// o Super Mario Bros.
// o Donkey Kong (homebrew)
// o etc.
type atari32k struct {
	atari
}

func newAtari32k(data []byte) (cartMapper, error) {
	cart := &atari32k{}
	cart.bankSize = 4096
	cart.description = "atari 32k"
	cart.formatID = "F4"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) != cart.bankSize*cart.numBanks() {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	cart.initialise()

	return cart, nil
}

func (cart atari32k) numBanks() int {
	return 8
}

func (cart *atari32k) read(addr uint16) (uint8, error) {
	if data, ok := cart.atari.read(addr); ok {
		return data, nil
	}

	data := cart.banks[cart.bank][addr]

	if addr == 0x0ff4 {
		cart.bank = 0
	} else if addr == 0x0ff5 {
		cart.bank = 1
	} else if addr == 0x0ff6 {
		cart.bank = 2
	} else if addr == 0x0ff7 {
		cart.bank = 3
	} else if addr == 0x0ff8 {
		cart.bank = 4
	} else if addr == 0x0ff9 {
		cart.bank = 5
	} else if addr == 0x0ffa {
		cart.bank = 6
	} else if addr == 0x0ffb {
		cart.bank = 7
	}

	return data, nil
}

func (cart *atari32k) write(addr uint16, data uint8) error {
	if ok := cart.atari.write(addr, data); ok {
		return nil
	}

	if addr == 0x0ff4 {
		cart.bank = 0
	} else if addr == 0x0ff5 {
		cart.bank = 1
	} else if addr == 0x0ff6 {
		cart.bank = 2
	} else if addr == 0x0ff7 {
		cart.bank = 3
	} else if addr == 0x0ff8 {
		cart.bank = 4
	} else if addr == 0x0ff9 {
		cart.bank = 5
	} else if addr == 0x0ffa {
		cart.bank = 6
	} else if addr == 0x0ffb {
		cart.bank = 7
	} else {
		return errors.New(errors.BusError, addr)
	}

	return nil
}
