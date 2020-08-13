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
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Cartridge defines the information and operations for a VCS cartridge
type Cartridge struct {
	bus.DebugBus
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

// Peek is an implementation of memory.DebugBus. Address must be normalised.
func (cart *Cartridge) Peek(addr uint16) (uint8, error) {
	return cart.mapper.Read(addr&memorymap.CartridgeBits, false)
}

// Poke is an implementation of memory.DebugBus. Address must be normalised.
func (cart *Cartridge) Poke(addr uint16, data uint8) error {
	return cart.mapper.Write(addr&memorymap.CartridgeBits, data, false, true)
}

// Patch writes to cartridge memory. Offset is measured from the start of
// cartridge memory. It differs from Poke in that respect
func (cart *Cartridge) Patch(offset int, data uint8) error {
	return cart.mapper.Patch(offset, data)
}

// Read is an implementation of memory.CPUBus. Address should not be
// normalised.
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
	if _, ok := cart.mapper.(*supercharger.Supercharger); ok {
		return cart.mapper.Read(addr, cart.Passive)
	}
	return cart.mapper.Read(addr&memorymap.CartridgeBits, cart.Passive)
}

// Write is an implementation of memory.CPUBus. Address should not be
// normalised.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	if _, ok := cart.mapper.(*supercharger.Supercharger); ok {
		return cart.mapper.Write(addr, data, cart.Passive, false)
	}
	return cart.mapper.Write(addr&memorymap.CartridgeBits, data, cart.Passive, false)
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
//
// How cartridges are mapped into the VCS's 4k space can differs dramatically.
// Much of the implementation details have been cribbed from Kevin Horton's
// "Cart Information" document [sizes.txt]. Other sources of information noted
// as appropriate.
func (cart *Cartridge) Attach(cartload cartridgeloader.Loader) error {
	err := cartload.Load()
	if err != nil {
		return err
	}

	// note name of cartridge
	cart.Filename = cartload.Filename
	cart.Hash = cartload.Hash
	cart.mapper = newEjected()

	cartload.Mapping = strings.ToUpper(cartload.Mapping)

	if cartload.Mapping == "" || cartload.Mapping == "AUTO" {
		return cart.fingerprint(cartload)
	}

	addSuperchip := false

	switch cartload.Mapping {
	case "2k":
		cart.mapper, err = newAtari2k(cartload.Data)
	case "4k":
		cart.mapper, err = newAtari4k(cartload.Data)
	case "F8":
		cart.mapper, err = newAtari8k(cartload.Data)
	case "F6":
		cart.mapper, err = newAtari16k(cartload.Data)
	case "F4":
		cart.mapper, err = newAtari32k(cartload.Data)
	case "2k+":
		cart.mapper, err = newAtari2k(cartload.Data)
		addSuperchip = true
	case "4k+":
		cart.mapper, err = newAtari4k(cartload.Data)
		addSuperchip = true
	case "F8+":
		cart.mapper, err = newAtari8k(cartload.Data)
		addSuperchip = true
	case "F6+":
		cart.mapper, err = newAtari16k(cartload.Data)
		addSuperchip = true
	case "F4+":
		cart.mapper, err = newAtari32k(cartload.Data)
		addSuperchip = true
	case "FA":
		cart.mapper, err = newCBS(cartload.Data)
	case "FE":
		// !!TODO: FE cartridge mapping
	case "E0":
		cart.mapper, err = newParkerBros(cartload.Data)
	case "E7":
		cart.mapper, err = newMnetwork(cartload.Data)
	case "3F":
		cart.mapper, err = newTigervision(cartload.Data)
	case "AR":
		cart.mapper, err = supercharger.NewSupercharger(cartload)
	case "DPC":
		cart.mapper, err = newDPC(cartload.Data)
	case "DPC+":
		cart.mapper, err = harmony.NewDPCplus(cartload.Data)
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
func (cart Cartridge) GetBank(addr uint16) banks.Details {
	if addr&memorymap.OriginCart != memorymap.OriginCart {
		return banks.Details{NonCart: true}
	}

	return cart.mapper.GetBank(addr & memorymap.CartridgeBits)
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

// GetRegistersBus returns interface to the registers of the cartridge or nil
// if cartridge has no registers
func (cart Cartridge) GetRegistersBus() bus.CartRegistersBus {
	if bus, ok := cart.mapper.(bus.CartRegistersBus); ok {
		return bus
	}
	return nil
}

// GetStaticBus returns interface to the static area of the cartridge or nil if
// cartridge has no static area
func (cart Cartridge) GetStaticBus() bus.CartStaticBus {
	if bus, ok := cart.mapper.(bus.CartStaticBus); ok {
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

// GetTapeBus returns interface to a tape bus or nil if catridge has no tape
func (cart Cartridge) GetTapeBus() bus.CartTapeBus {
	if bus, ok := cart.mapper.(bus.CartTapeBus); ok {
		return bus
	}
	return nil
}

// IterateBanks returns the sequence of banks in a cartridge. To return the
// next bank in the sequence, call the function with the instance of
// banks.Content returned from the previous call. The end of the sequence is
// indicated by the nil value. Start a new iteration with the nil argument.
func (cart Cartridge) IterateBanks(prev *banks.Content) (*banks.Content, error) {
	// to keep the mappers as neat as possible, handle the nil special condition here
	if prev == nil {
		prev = &banks.Content{Number: -1}
	}

	return cart.mapper.IterateBanks(prev), nil
}
