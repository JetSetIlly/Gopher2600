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

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/logger"
)

// Cartridge defines the information and operations for a VCS cartridge.
type Cartridge struct {
	bus.DebugBus
	bus.CPUBus

	prefs *preferences.Preferences

	Filename string
	Hash     string

	// the specific cartridge data, mapped appropriately to the memory
	// interfaces
	mapper mapper.CartMapper
}

// Sentinal error returned if operation is on the ejected cartridge type.
const (
	Ejected = "cartridge ejected"
)

// NewCartridge is the preferred method of initialisation for the cartridge
// type.
func NewCartridge(prefs *preferences.Preferences) *Cartridge {
	cart := &Cartridge{prefs: prefs}
	cart.Eject()
	return cart
}

func (cart *Cartridge) Snapshot() mapper.CartSnapshot {
	return cart.mapper.Snapshot()
}

func (cart *Cartridge) Plumb(s mapper.CartSnapshot) {
	cart.mapper.Plumb(s)
}

// Reset volative contents of Cartridge.
func (cart *Cartridge) Reset() {
	if cart.prefs != nil && cart.prefs.RandomState.Get().(bool) {
		cart.mapper.Reset(cart.prefs.RandSrc)
	} else {
		cart.mapper.Reset(nil)
	}
}

func (cart Cartridge) String() string {
	return cart.Filename
}

// MappingSummary returns a current string summary of the mapper.
func (cart Cartridge) MappingSummary() string {
	return cart.mapper.String()
}

// ID returns the cartridge mapping ID.
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
// cartridge memory. It differs from Poke in that respect.
func (cart *Cartridge) Patch(offset int, data uint8) error {
	return cart.mapper.Patch(offset, data)
}

// Read is an implementation of memory.CPUBus. Address should not be
// normalised.
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
	if _, ok := cart.mapper.(*supercharger.Supercharger); ok {
		return cart.mapper.Read(addr, false)
	}
	return cart.mapper.Read(addr&memorymap.CartridgeBits, false)
}

// Write is an implementation of memory.CPUBus. Address should not be
// normalised.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	if _, ok := cart.mapper.(*supercharger.Supercharger); ok {
		return cart.mapper.Write(addr, data, false, false)
	}
	return cart.mapper.Write(addr&memorymap.CartridgeBits, data, false, false)
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger.
func (cart *Cartridge) Eject() {
	cart.Filename = "ejected"
	cart.Hash = ""
	cart.mapper = newEjected()
}

// IsEjected returns true if no cartridge is attached.
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

	cart.Filename = cartload.Filename
	cart.Hash = cartload.Hash
	cart.mapper = newEjected()

	// fingerprint cartridgeloader.Loader
	if cartload.Mapping == "" || cartload.Mapping == "AUTO" {
		err := cart.fingerprint(cartload)
		if err != nil {
			return curated.Errorf("cartridge: %v", err)
		}

		// in addition to the regular fingerprint we also check to see if this
		// is PlusROM cartridge (which can be combined with a regular cartridge
		// format)
		if cart.fingerprintPlusROM(cartload) {
			// try creating a NewPlusROM instance
			pr, err := plusrom.NewPlusROM(cart.mapper, cartload.OnLoaded)

			if err != nil {
				// if the error is a NotAPlusROM error then log the false
				// positive and return a success, keeping the main cartridge
				// mapper intact
				if curated.Is(err, plusrom.NotAPlusROM) {
					logger.Log("cartridge", err.Error())
					return nil
				}

				return curated.Errorf("cartridge: %v", err)
			}

			// we've wrapped the main cartridge mapper inside the PlusROM
			// mapper and we need to point the mapper field to the the new
			// PlusROM instance
			cart.mapper = pr

			// log that this is PlusROM cartridge
			logger.Log("cartridge", fmt.Sprintf("%s cartridge contained in PlusROM", cart.ID()))
		}

		return nil
	}

	// a specific cartridge mapper was specified

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
	case "DF":
		cart.mapper, err = newDF(cartload.Data)
	case "3E":
		cart.mapper, err = new3e(cartload.Data)
	case "3E+":
		cart.mapper, err = new3ePlus(cartload.Data)
	case "DPC":
		cart.mapper, err = newDPC(cartload.Data)
	case "DPC+":
		cart.mapper, err = harmony.NewDPCplus(cartload.Data)
	}

	if err != nil {
		return curated.Errorf("cartridge: %v", err)
	}

	if addSuperchip {
		if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
			superchip.AddSuperchip()
		}
	}

	return nil
}

// NumBanks returns the number of banks in the catridge.
func (cart Cartridge) NumBanks() int {
	return cart.mapper.NumBanks()
}

// GetBank returns the current bank information for the specified address. See
// documentation for memorymap.Bank for more information.
func (cart Cartridge) GetBank(addr uint16) mapper.BankInfo {
	if addr&memorymap.OriginCart != memorymap.OriginCart {
		return mapper.BankInfo{NonCart: true}
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
	cart.mapper.Listen(addr, data)
}

// Step should be called every CPU cycle. The attached cartridge may or may not
// change its state as a result. In fact, very few cartridges care about this.
func (cart Cartridge) Step() {
	cart.mapper.Step()
}

// GetRegistersBus returns interface to the registers of the cartridge or nil
// if cartridge has no registers.
func (cart Cartridge) GetRegistersBus() mapper.CartRegistersBus {
	if bus, ok := cart.mapper.(mapper.CartRegistersBus); ok {
		return bus
	}
	return nil
}

// GetStaticBus returns interface to the static area of the cartridge or nil if
// cartridge has no static area.
func (cart Cartridge) GetStaticBus() mapper.CartStaticBus {
	if bus, ok := cart.mapper.(mapper.CartStaticBus); ok {
		return bus
	}
	return nil
}

// GetRAMbus returns interface to ram busor  nil if catridge contains no RAM.
func (cart Cartridge) GetRAMbus() mapper.CartRAMbus {
	if bus, ok := cart.mapper.(mapper.CartRAMbus); ok {
		return bus
	}
	return nil
}

// GetTapeBus returns interface to a tape bus or nil if catridge has no tape.
func (cart Cartridge) GetTapeBus() mapper.CartTapeBus {
	if bus, ok := cart.mapper.(mapper.CartTapeBus); ok {
		return bus
	}
	return nil
}

// GetContainer returns interface to cartridge container or nil if cartridge is
// not in a container.
func (cart Cartridge) GetContainer() mapper.CartContainer {
	if cc, ok := cart.mapper.(mapper.CartContainer); ok {
		return cc
	}
	return nil
}

// GetCartHotspots returns interface to hotspots bus or nil if cartridge has no
// hotspots it wants to report.
func (cart Cartridge) GetCartHotspots() mapper.CartHotspotsBus {
	if cc, ok := cart.mapper.(mapper.CartHotspotsBus); ok {
		return cc
	}
	return nil
}

// CopyBanks returns the sequence of banks in a cartridge. To return the
// next bank in the sequence, call the function with the instance of
// mapper.BankContent returned from the previous call. The end of the sequence is
// indicated by the nil value. Start a new iteration with the nil argument.
func (cart Cartridge) CopyBanks() ([]mapper.BankContent, error) {
	return cart.mapper.CopyBanks(), nil
}
