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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/dpcplus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/moviecart"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

// Cartridge defines the information and operations for a VCS cartridge.
type Cartridge struct {
	instance *instance.Instance

	// filename/hash taken from cartridgeloader. choosing not to keep a
	// reference to the cartridge loader itself.
	Filename  string
	ShortName string
	Hash      string

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
func NewCartridge(instance *instance.Instance) *Cartridge {
	cart := &Cartridge{instance: instance}
	cart.Eject()
	return cart
}

// Snapshot creates a copy of the current cartridge.
func (cart *Cartridge) Snapshot() *Cartridge {
	n := *cart
	n.mapper = cart.mapper.Snapshot()
	return &n
}

// Plumb makes sure everything is ship-shape after a rewind event.
//
// The fromDifferentEmulation indicates that the State has been created by a
// different VCS instance than the one being plumbed into.
//
// See mapper.PlumbFromDifferentEmulation for how this affects mapper
// implementations.
func (cart *Cartridge) Plumb(fromDifferentEmulation bool) {
	if fromDifferentEmulation {
		if m, ok := cart.mapper.(mapper.PlumbFromDifferentEmulation); ok {
			m.PlumbFromDifferentEmulation()
			return
		}
	}
	cart.mapper.Plumb()
}

// Reset volative contents of Cartridge.
func (cart *Cartridge) Reset() {
	cart.mapper.Reset()
}

// String returns a summary of the cartridge, it's mapper and any containers.
//
// For just the filename use the Filename field or the ShortName field.
// Filename includes the path.
//
// For just the mapping use the ID() function and for just the container ID use
// the ContainerID() function.
func (cart *Cartridge) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s (%s)", cart.ShortName, cart.ID()))
	if cc := cart.GetContainer(); cc != nil {
		s.WriteString(fmt.Sprintf(" [%s]", cc.ContainerID()))
	}
	return s.String()
}

// MappedBanks returns a string summary of the mapping. ie. what banks are mapped in.
func (cart *Cartridge) MappedBanks() string {
	return cart.mapper.MappedBanks()
}

// ID returns the cartridge mapping ID.
func (cart *Cartridge) ID() string {
	return cart.mapper.ID()
}

// Container returns the cartridge continer ID. If the cartridge is not in a
// container the empty string is returned.
func (cart *Cartridge) ContainerID() string {
	if cc := cart.GetContainer(); cc != nil {
		return cc.ContainerID()
	}
	return ""
}

// Peek is an implementation of memory.DebugBus. Address must be normalised.
func (cart *Cartridge) Peek(addr uint16) (uint8, error) {
	return cart.mapper.Read(addr&memorymap.CartridgeBits, true)
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

// Read is an implementation of memory.CPUBus.
func (cart *Cartridge) Read(addr uint16) (uint8, error) {
	return cart.mapper.Read(addr&memorymap.CartridgeBits, false)
}

// Write is an implementation of memory.CPUBus.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	return cart.mapper.Write(addr&memorymap.CartridgeBits, data, false, false)
}

// Eject removes memory from cartridge space and unlike the real hardware,
// attaches a bank of empty memory - for convenience of the debugger.
func (cart *Cartridge) Eject() {
	cart.Filename = "ejected"
	cart.ShortName = "ejected"
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
	cart.ShortName = cartload.ShortName()
	cart.Hash = cartload.Hash
	cart.mapper = newEjected()

	// log result of Attach() on function return
	defer func() {
		// we might have arrived here as a result of an error so we should
		// check cart.mapper before trying to access it
		if cart.mapper == nil {
			return
		}

		if _, ok := cart.mapper.(*ejected); !ok {
			logger.Logf("cartridge", "inserted %s", cart.mapper.ID())
		}
	}()

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
			pr, err := plusrom.NewPlusROM(cart.instance, cart.mapper, cartload.VCSHook)

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
			logger.Logf("cartridge", "%s cartridge contained in PlusROM", cart.ID())
		}

		return nil
	}

	// a specific cartridge mapper was specified

	trySuperchip := false

	switch strings.ToUpper(cartload.Mapping) {
	case "2K":
		cart.mapper, err = newAtari2k(cart.instance, *cartload.Data)
	case "4K":
		cart.mapper, err = newAtari4k(cart.instance, *cartload.Data)
	case "F8":
		cart.mapper, err = newAtari8k(cart.instance, *cartload.Data)
	case "F6":
		cart.mapper, err = newAtari16k(cart.instance, *cartload.Data)
	case "F4":
		cart.mapper, err = newAtari32k(cart.instance, *cartload.Data)
	case "2KSC":
		cart.mapper, err = newAtari2k(cart.instance, *cartload.Data)
		trySuperchip = true
	case "4KSC":
		cart.mapper, err = newAtari4k(cart.instance, *cartload.Data)
		trySuperchip = true
	case "F8SC":
		cart.mapper, err = newAtari8k(cart.instance, *cartload.Data)
		trySuperchip = true
	case "F6SC":
		cart.mapper, err = newAtari16k(cart.instance, *cartload.Data)
		trySuperchip = true
	case "F4SC":
		cart.mapper, err = newAtari32k(cart.instance, *cartload.Data)
		trySuperchip = true
	case "FA":
		cart.mapper, err = newCBS(cart.instance, *cartload.Data)
	case "FE":
		// !!TODO: FE cartridge mapping
	case "E0":
		cart.mapper, err = newParkerBros(cart.instance, *cartload.Data)
	case "E7":
		cart.mapper, err = newMnetwork(cart.instance, *cartload.Data)
	case "3F":
		cart.mapper, err = newTigervision(cart.instance, *cartload.Data)
	case "AR":
		cart.mapper, err = supercharger.NewSupercharger(cart.instance, cartload)
	case "DF":
		cart.mapper, err = newDF(cart.instance, *cartload.Data)
	case "3E":
		cart.mapper, err = new3e(cart.instance, *cartload.Data)
	case "E3P":
		// synonym for 3E+
		fallthrough
	case "E3+":
		// synonym for 3E+
		fallthrough
	case "3E+":
		cart.mapper, err = new3ePlus(cart.instance, *cartload.Data)
	case "EF":
		cart.mapper, err = newEF(cart.instance, *cartload.Data)
	case "EFSC":
		cart.mapper, err = newEF(cart.instance, *cartload.Data)
		trySuperchip = true
	case "SB":
		cart.mapper, err = newSuperbank(cart.instance, *cartload.Data)
	case "DPC":
		cart.mapper, err = newDPC(cart.instance, *cartload.Data)
	case "DPC+":
		cart.mapper, err = dpcplus.NewDPCplus(cart.instance, cart.Filename, *cartload.Data)
	case "CDF":
		// CDF mapper defaults to version CDFJ
		cart.mapper, err = cdf.NewCDF(cart.instance, cart.Filename, "CDFJ", *cartload.Data)
	case "MVC":
		cart.mapper, err = moviecart.NewMoviecart(cart.instance, cartload)
	}

	if err != nil {
		return curated.Errorf("cartridge: %v", err)
	}

	if trySuperchip {
		if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
			superchip.AddSuperchip()
		} else {
			logger.Logf("cartridge", "cannot add superchip to %s mapper", cart.ID())
		}
	}

	return nil
}

// NumBanks returns the number of banks in the catridge.
func (cart *Cartridge) NumBanks() int {
	return cart.mapper.NumBanks()
}

// GetBank returns the current bank information for the specified address. See
// documentation for memorymap.Bank for more information.
func (cart *Cartridge) GetBank(addr uint16) mapper.BankInfo {
	bank := cart.mapper.GetBank(addr & memorymap.CartridgeBits)
	bank.NonCart = addr&memorymap.OriginCart != memorymap.OriginCart
	return bank
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
func (cart *Cartridge) Listen(addr uint16, data uint8) {
	cart.mapper.Listen(addr, data)
}

// Step should be called every CPU cycle. The attached cartridge may or may not
// change its state as a result. In fact, very few cartridges care about this.
func (cart *Cartridge) Step(clock float32) {
	cart.mapper.Step(clock)
}

// Hotload cartridge ROM into emulation. Not changing any other state of the
// emulation.
func (cart *Cartridge) HotLoad(cartload cartridgeloader.Loader) error {
	if hl, ok := cart.mapper.(mapper.CartHotLoader); ok {
		err := cartload.Load()
		if err != nil {
			return err
		}

		cart.Hash = cartload.Hash

		err = hl.HotLoad(*cartload.Data)
		if err != nil {
			return err
		}

		return nil
	}

	return curated.Errorf("cartridge: %s does not support hotloading", cart.mapper.ID())
}

// GetRegistersBus returns interface to the registers of the cartridge or nil
// if cartridge has no registers.
func (cart *Cartridge) GetRegistersBus() mapper.CartRegistersBus {
	if bus, ok := cart.mapper.(mapper.CartRegistersBus); ok {
		return bus
	}
	return nil
}

// GetStaticBus returns interface to the static area of the cartridge or nil if
// cartridge has no static area.
func (cart *Cartridge) GetStaticBus() mapper.CartStaticBus {
	if bus, ok := cart.mapper.(mapper.CartStaticBus); ok {
		return bus
	}
	return nil
}

// GetRAMbus returns interface to ram busor  nil if catridge contains no RAM.
func (cart *Cartridge) GetRAMbus() mapper.CartRAMbus {
	if bus, ok := cart.mapper.(mapper.CartRAMbus); ok {
		return bus
	}
	return nil
}

// GetTapeBus returns interface to a tape bus or nil if catridge has no tape.
func (cart *Cartridge) GetTapeBus() mapper.CartTapeBus {
	if bus, ok := cart.mapper.(mapper.CartTapeBus); ok {
		return bus
	}
	return nil
}

// GetContainer returns interface to cartridge container or nil if cartridge is
// not in a container.
func (cart *Cartridge) GetContainer() mapper.CartContainer {
	if cc, ok := cart.mapper.(mapper.CartContainer); ok {
		return cc
	}
	return nil
}

// GetCartHotspots returns interface to hotspots bus or nil if cartridge has no
// hotspots it wants to report.
func (cart *Cartridge) GetCartLabelsBus() mapper.CartLabelsBus {
	if cc, ok := cart.mapper.(mapper.CartLabelsBus); ok {
		return cc
	}
	return nil
}

// GetCartHotspotsBus returns interface to hotspots bus or nil if cartridge has no
// hotspots it wants to report.
func (cart *Cartridge) GetCartHotspotsBus() mapper.CartHotspotsBus {
	if cc, ok := cart.mapper.(mapper.CartHotspotsBus); ok {
		return cc
	}
	return nil
}

// GetCoProcBus returns interface to coprocessor bus.
func (cart *Cartridge) GetCoProcBus() mapper.CartCoProcBus {
	if cpd, ok := cart.mapper.(mapper.CartCoProcBus); ok {
		return cpd
	}
	return nil
}

// CopyBanks returns the sequence of banks in a cartridge. To return the
// next bank in the sequence, call the function with the instance of
// mapper.BankContent returned from the previous call. The end of the sequence is
// indicated by the nil value. Start a new iteration with the nil argument.
func (cart *Cartridge) CopyBanks() ([]mapper.BankContent, error) {
	return cart.mapper.CopyBanks(), nil
}

// RewindBoundary returns true if the cartridge indicates that something has
// happened that should not be part of the rewind history. Returns false if
// cartridge mapper does not care about the rewind sub-system.
func (cart *Cartridge) RewindBoundary() bool {
	if rb, ok := cart.mapper.(mapper.CartRewindBoundary); ok {
		return rb.RewindBoundary()
	}
	return false
}

// ROMDump implements the mapper.CartROMDump interface.
func (cart *Cartridge) ROMDump() (string, error) {
	romdump := fmt.Sprintf("%s.bin", unique.Filename("", cart.ShortName))
	if rb, ok := cart.mapper.(mapper.CartROMDump); ok {
		return romdump, rb.ROMDump(romdump)
	}
	return "", curated.Errorf("cartridge: %s does not support ROM dumping", cart.mapper.ID())
}

// NewFrame implements the television.NewFrame interface.
//
// Note that there is no GetFrameTriggerInterface() or any similar function
// here. This simplifies the code for attaching and detaching cartridges and
// without any perfomance penalties. In other words, this function is always
// called by the television instance on the event of a new frame.
func (cart *Cartridge) NewFrame(fi television.FrameInfo) error {
	if t, ok := cart.mapper.(television.FrameTrigger); ok {
		return t.NewFrame(fi)
	}
	return nil
}

// BreakpointHasTriggered implements the mapper.CartBreakpoints interface.
func (cart *Cartridge) BreakpointHasTriggered() bool {
	if bp, ok := cart.mapper.(mapper.CartBreakpoints); ok {
		return bp.BreakpointHasTriggered()
	}
	return false
}

// ResumeAfterBreakpoint implements the mapper.CartBreakpoints interface.
func (cart *Cartridge) ResumeAfterBreakpoint() error {
	if bp, ok := cart.mapper.(mapper.CartBreakpoints); ok {
		return bp.ResumeAfterBreakpoint()
	}
	return nil
}

// BreakpointsDisable implements the mapper.CartBreakpoints interface.
func (cart *Cartridge) BreakpointsDisable(disable bool) {
	if bp, ok := cart.mapper.(mapper.CartBreakpoints); ok {
		bp.BreakpointsDisable(disable)
	}
}
