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
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	bus.DebuggerBus
	bus.CPUBus

	Filename string
	Hash     string

	// the specific cartridge data, mapped appropriately to the memory
	// interfaces
	mapper cartMapper
}

// NewCartridge is the preferred method of initialisation for the cartridge
// type
func NewCartridge() *Cartridge {
	cart := &Cartridge{}
	cart.Eject()
	return cart
}

func (cart Cartridge) String() string {
	return cart.Summary()
}

// Summary returns brief information about the cartridge. Two lines: first line
// is the path to the cartridge and the second line is information about the
// mapper, including bank information
func (cart Cartridge) Summary() string {
	return fmt.Sprintf("%s\n%s", cart.Filename, cart.mapper)
}

// Format returns the cartridge format ID
func (cart Cartridge) Format() string {
	return cart.mapper.format()
}

// Peek is an implementation of memory.DebuggerBus. Address must be normalised.
func (cart *Cartridge) Peek(addr uint16) (uint8, error) {
	return cart.Read(addr)
}

// Poke is an implementation of memory.DebuggerBus. This poke pokes the current
// cartridge bank. See Patch for a different method. Address must be
// normalised.
func (cart *Cartridge) Poke(addr uint16, data uint8) error {
	return cart.mapper.poke(addr^memorymap.OriginCart, data)
}

// Patch writes to cartridge memory. Offset is measured from the start of
// cartridge memory. It differs from Poke in that respect
func (cart *Cartridge) Patch(offset uint16, data uint8) error {
	return cart.mapper.patch(offset, data)
}

// Read is an implementation of memory.CPUBus. Address must be normalised.
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
	return cart.mapper.read(addr ^ memorymap.OriginCart)
}

// Write is an implementation of memory.CPUBus. Address must be normalised.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	return cart.mapper.write(addr^memorymap.OriginCart, data)
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.Filename = ejectedName
	cart.Hash = ejectedHash
	cart.mapper = newEjected()
}

// IsEjected returns true if no cartridge is attached
func (cart *Cartridge) IsEjected() bool {
	return cart.Hash == ejectedHash
}

// Attach the cartridge loader to the VCS and make available the data to the CPU
// bus
func (cart *Cartridge) Attach(cartload cartridgeloader.Loader) error {
	data, err := cartload.Load()
	if err != nil {
		return err
	}

	// note name of cartridge
	cart.Filename = cartload.Filename
	cart.mapper = newEjected()

	// generate hash
	cart.Hash = fmt.Sprintf("%x", sha1.Sum(data))

	// check that the hash matches the expected value
	if cartload.Hash != "" && cartload.Hash != cart.Hash {
		return errors.New(errors.CartridgeError, "unexpected hash value")
	}

	// how cartridges are mapped into the 4k space can differs dramatically.
	// the following implementation details have been cribbed from Kevin
	// Horton's "Cart Information" document [sizes.txt]

	cartload.Format = strings.ToUpper(cartload.Format)

	if cartload.Format == "" || cartload.Format == "AUTO" {
		return cart.fingerprint(data)
	}

	addSuperchip := false

	switch cartload.Format {
	case "2k":
		cart.mapper, err = newAtari2k(data)
	case "4k":
		cart.mapper, err = newAtari4k(data)
	case "F8":
		cart.mapper, err = newAtari8k(data)
	case "F6":
		cart.mapper, err = newAtari16k(data)
	case "F4":
		cart.mapper, err = newAtari32k(data)

	case "2k+SC":
		cart.mapper, err = newAtari2k(data)
		addSuperchip = true
	case "4k+SC":
		cart.mapper, err = newAtari4k(data)
		addSuperchip = true
	case "F8+SC":
		cart.mapper, err = newAtari8k(data)
		addSuperchip = true
	case "F6+SC":
		cart.mapper, err = newAtari16k(data)
		addSuperchip = true
	case "F4+SC":
		cart.mapper, err = newAtari32k(data)
		addSuperchip = true

	case "FA":
		cart.mapper, err = newCBS(data)
	case "FE":
		// !!TODO: FE cartridge format
	case "E0":
		cart.mapper, err = newparkerBros(data)
	case "E7":
		cart.mapper, err = newMnetwork(data)
	case "3F":
		cart.mapper, err = newTigervision(data)
	case "AR":
		// !!TODO: AR cartridge format

	case "DPC":
		cart.mapper, err = newDPC(data)
	}

	if addSuperchip {
		if superchip, ok := cart.mapper.(optionalSuperchip); ok {
			if !superchip.addSuperchip() {
				err = errors.New(errors.CartridgeError, "error adding superchip")
			}
		} else {
			err = errors.New(errors.CartridgeError, "error adding superchip")
		}
	}

	return err
}

// Initialise the cartridge
func (cart *Cartridge) Initialise() {
	cart.mapper.initialise()
}

// NumBanks returns the number of banks in the catridge
func (cart Cartridge) NumBanks() int {
	return cart.mapper.numBanks()
}

// GetBank returns the current bank number for the specified address
//
// WARNING: For some cartridge types this is the same as asking for the current
// address
//
// Address must be a normlised cartridge address.
func (cart Cartridge) GetBank(addr uint16) int {
	return cart.mapper.getBank(addr & memorymap.AddressMaskCart)
}

// SetBank maps the specified address such that it references the specified
// bank. For many cart mappers this just means switching banks for the entire
// cartridge. Address must be normalised.
//
// NOTE: For some cartridge types, the specific address is not important
func (cart *Cartridge) SetBank(addr uint16, bank int) error {
	return cart.mapper.setBank(addr&memorymap.AddressMaskCart, bank)
}

// SaveState notes and returns the current state of the cartridge (RAM
// contents, selected bank)
func (cart *Cartridge) SaveState() interface{} {
	return cart.mapper.saveState()
}

// RestoreState retuns the state of the cartridge to a previously known state
func (cart *Cartridge) RestoreState(state interface{}) error {
	return cart.mapper.restoreState(state)
}

// Listen for data at the specified address.
//
// The VCS cartridge port is wired up to all 13 address lines of the 6507.
// Under normal operation, the chip-select line is used by the cartridge to
// know when to put data on the data bus. If it's not "on" then the cartridge
// does nothing.
//
// However, the option is there to "listen" on the address bus. Notably the
// tigervision (3F) format listens for address 0x003f, which is in the TIA
// address space. When this address is triggered, the tigervision cartridge
// will use whatever is on the data bus to switch banks.
func (cart Cartridge) Listen(addr uint16, data uint8) {
	cart.mapper.listen(addr, data)
}

// GetRAMinfo returns an instance of RAMinfo or nil if catridge contains no RAM
func (cart Cartridge) GetRAMinfo() []RAMinfo {
	return cart.mapper.getRAMinfo()
}

// Step should be called every CPU cycle. The attached cartridge may or may not
// change its state as a result. In fact, very few cartridges care about this.
func (cart Cartridge) Step() {
	cart.mapper.step()
}
