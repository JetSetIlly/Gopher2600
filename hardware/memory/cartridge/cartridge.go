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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
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

	// when cartridge is in passive mode, cartridge hotspots do not work. We
	// send the passive value to the Read() and Write() functions of the mapper
	// and we also use it to prevent the Listen() function from triggering.
	// Useful for disassembly when we don't want the cartridge to react.
	Passive bool
}

// NewCartridge is the preferred method of initialisation for the cartridge
// type
func NewCartridge() *Cartridge {
	cart := &Cartridge{}
	cart.Eject()
	return cart
}

func (cart Cartridge) String() string {
	return cart.Filename
}

// MappingSummary returns a current string summary of the mapper
func (cart Cartridge) MappingSummary() string {
	return fmt.Sprintf("%s", cart.mapper)
}

// ID returns the cartridge mapping ID
func (cart Cartridge) ID() string {
	return cart.mapper.ID()
}

// Peek is an implementation of memory.DebuggerBus. Address must be normalised.
func (cart *Cartridge) Peek(addr uint16) (uint8, error) {
	return cart.mapper.Read(addr^memorymap.OriginCart, false)
}

// Poke is an implementation of memory.DebuggerBus. Address must be normalised.
func (cart *Cartridge) Poke(addr uint16, data uint8) error {
	return cart.mapper.Write(addr^memorymap.OriginCart, data, false, true)
}

// Patch writes to cartridge memory. Offset is measured from the start of
// cartridge memory. It differs from Poke in that respect
func (cart *Cartridge) Patch(offset int, data uint8) error {
	return cart.mapper.Patch(offset, data)
}

// Read is an implementation of memory.CPUBus. Address must be normalised.
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
	return cart.mapper.Read(addr^memorymap.OriginCart, cart.Passive)
}

// Write is an implementation of memory.CPUBus. Address must be normalised.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	return cart.mapper.Write(addr^memorymap.OriginCart, data, cart.Passive, false)
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger
func (cart *Cartridge) Eject() {
	cart.Filename = "ejected"
	cart.Hash = ""
	cart.mapper = newEjected()
}

// IsEjected returns true if no cartridge is attached
func (cart *Cartridge) IsEjected() bool {
	_, ok := cart.mapper.(*ejected)
	return ok
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

	cartload.Mapping = strings.ToUpper(cartload.Mapping)

	if cartload.Mapping == "" || cartload.Mapping == "AUTO" {
		return cart.fingerprint(data)
	}

	addSuperchip := false

	switch cartload.Mapping {
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
		// !!TODO: FE cartridge mapping
	case "E0":
		cart.mapper, err = newParkerBros(data)
	case "E7":
		cart.mapper, err = newMnetwork(data)
	case "3F":
		cart.mapper, err = newTigervision(data)
	case "AR":
		// !!TODO: AR cartridge mapping

	case "DPC":
		cart.mapper, err = newDPC(data)
	case "DPC+":
		cart.mapper, err = harmony.NewDPCplus(data)
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
	cart.mapper.Initialise()
}

// NumBanks returns the number of banks in the catridge
func (cart Cartridge) NumBanks() int {
	return cart.mapper.NumBanks()
}

// GetBank returns the current bank information for the specified address. See
// documentation for memorymap.Bank for more information.
func (cart Cartridge) GetBank(addr uint16) memorymap.BankDetails {
	if addr&memorymap.OriginCart != memorymap.OriginCart {
		return memorymap.BankDetails{NonCart: true}
	}

	return cart.mapper.GetBank(addr & memorymap.CartridgeBits)
}

// SetBank maps in a bank to the segment at the stated address. For many
// cartridge mappers memory is not segmented and the address is ignored (except
// to check for whether it is a valid cartridge address).
//
// SetBank() will return a CartridgeSetBank error if for some reason the bank
// cannot be mapped to a specified address.
func (cart *Cartridge) SetBank(addr uint16, bank int) error {
	if addr&memorymap.OriginCart != memorymap.OriginCart {
		return errors.New(errors.CartridgeError, "address not in cartridge area")
	}
	if bank < 0 && bank >= cart.mapper.NumBanks() {
		return errors.New(errors.CartridgeError, "bank invalid")
	}
	return cart.mapper.SetBank(addr&memorymap.CartridgeBits, bank)
}

// BankSize returns the number of bytes in each bank. The number of segments
// can be obtained by dividing 4096 by the returned value.
func (cart Cartridge) BankSize() uint16 {
	switch cart.mapper.(type) {
	case *mapper3ePlus:
		return 1024
	case *mnetwork:
		return 2048
	case *parkerBros:
		return 1024
	case *tigervision:
		return 2048
	case *supercharger.Supercharger:
		return 2048
	default:
		return 4096
	}
}

// Listen for data at the specified address.
//
// The VCS cartridge port is wired up to all 13 address lines of the 6507.
// Under normal operation, the chip-select line is used by the cartridge to
// know when to put data on the data bus. If it's not "on" then the cartridge
// does nothing.
//
// However, the option is there to "listen" on the address bus. Notably the
// tigervision (3F) mapping listens for address 0x003f, which is in the TIA
// address space. When this address is triggered, the tigervision cartridge
// will use whatever is on the data bus to switch banks.
func (cart Cartridge) Listen(addr uint16, data uint8) {
	if !cart.Passive {
		cart.mapper.Listen(addr, data)
	}
}

// Step should be called every CPU cycle. The attached cartridge may or may not
// change its state as a result. In fact, very few cartridges care about this.
func (cart Cartridge) Step() {
	cart.mapper.Step()
}

// GetDebugBus returns interface to the debugging bus to the cartridge.
func (cart Cartridge) GetDebugBus() bus.CartDebugBus {
	if bus, ok := cart.mapper.(bus.CartDebugBus); ok {
		return bus
	}
	return nil
}

// GetRAMbus returns an array of bus.CartRAM or nil if catridge contains no RAM
func (cart Cartridge) GetRAMbus() bus.CartRAMbus {
	if bus, ok := cart.mapper.(bus.CartRAMbus); ok {
		return bus
	}
	return nil
}
