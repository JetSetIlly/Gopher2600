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
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/ace"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/dpcplus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/elf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/moviecart"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

// Cartridge defines the information and operations for a VCS cartridge.
type Cartridge struct {
	env *environment.Environment

	// filename/hash taken from cartridgeloader. choosing not to keep a
	// reference to the cartridge loader itself.
	Filename  string
	ShortName string
	Hash      string

	// the specific cartridge data, mapped appropriately to the memory
	// interfaces
	mapper mapper.CartMapper

	// the CartBusStuff and CartCoProc interface are accessed a lot if
	// available. rather than performing type assertions too frequently we do
	// it in the Attach() function and the Plumb() function
	hasBusStuff  bool
	busStuff     mapper.CartBusStuff
	hasCoProcBus bool
	coprocBus    coprocessor.CartCoProcBus
}

// sentinal error returned if operation is on the ejected cartridge type.
var Ejected = errors.New("cartridge ejected")

// NewCartridge is the preferred method of initialisation for the cartridge
// type.
func NewCartridge(env *environment.Environment) *Cartridge {
	cart := &Cartridge{env: env}
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
// different VCS emulation than the one being plumbed into.
//
// See mapper.PlumbFromDifferentEmulation for how this affects mapper
// implementations.
func (cart *Cartridge) Plumb(env *environment.Environment, fromDifferentEmulation bool) {
	cart.env = env
	cart.busStuff, cart.hasBusStuff = cart.mapper.(mapper.CartBusStuff)
	cart.coprocBus, cart.hasCoProcBus = cart.mapper.(coprocessor.CartCoProcBus)

	if fromDifferentEmulation {
		if m, ok := cart.mapper.(mapper.PlumbFromDifferentEmulation); ok {
			m.PlumbFromDifferentEmulation(cart.env)
			return
		}
	}

	cart.mapper.Plumb(cart.env)
}

// Reset volative contents of Cartridge.
func (cart *Cartridge) Reset() error {
	return cart.mapper.Reset()
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

// MappedBanks returns a string summary of the mapping. ie. what banks are mapped in. If bank
// switching is not applicable to the cartridge then an empty string is returned.
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
	v, _, err := cart.mapper.Access(addr&memorymap.CartridgeBits, true)
	return v, err
}

// Poke is an implementation of memory.DebugBus. Address must be normalised.
func (cart *Cartridge) Poke(addr uint16, data uint8) error {
	return cart.mapper.AccessVolatile(addr&memorymap.CartridgeBits, data, true)
}

// Read is an implementation of memory.CPUBus.
func (cart *Cartridge) Read(addr uint16) (uint8, uint8, error) {
	return cart.mapper.Access(addr&memorymap.CartridgeBits, false)
}

// Write is an implementation of memory.CPUBus.
func (cart *Cartridge) Write(addr uint16, data uint8) error {
	return cart.mapper.AccessVolatile(addr&memorymap.CartridgeBits, data, false)
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
	var err error

	cart.Filename = cartload.Filename
	cart.ShortName = cartload.Name
	cart.Hash = cartload.HashSHA1
	cart.mapper = newEjected()

	// reset loader stream before we go any further
	err = cartload.Reset()
	if err != nil {
		return fmt.Errorf("cartridge: %w", err)
	}

	// log result of Attach() on function return
	defer func() {
		// we might have arrived here as a result of an error so we should
		// check cart.mapper before trying to access it
		if cart.mapper == nil {
			return
		}

		// get busstuff and coproc interfaces
		cart.busStuff, cart.hasBusStuff = cart.mapper.(mapper.CartBusStuff)
		cart.coprocBus, cart.hasCoProcBus = cart.mapper.(coprocessor.CartCoProcBus)

		if _, ok := cart.mapper.(*ejected); !ok {
			logger.Logf(cart.env, "cartridge", "inserted %s", cart.mapper.ID())
		}
	}()

	// whether the mapping was determined automatically. we keep track of this
	// so that we don't add a superchip if it wasn't expressly asked for
	var auto bool

	// specifying a mapper will sometimes imply the adding of a superchip
	var forceSuperchip bool

	mapping := strings.ToUpper(cartload.Mapping)

	// automatic fingerprinting of cartridge
	if mapping == "" || mapping == "AUTO" {
		auto = true
		mapping, err = cart.fingerprint(cartload)
		if err != nil {
			return fmt.Errorf("cartridge: %w", err)
		}

		// reset loader stream after fingerprinting
		err = cartload.Reset()
		if err != nil {
			return fmt.Errorf("cartridge: %w", err)
		}
	}

	// if the mapping has explicitely set as ACE we still want to unwrap it according to the current
	// preference value. this requires a call to fingerprintACE to get the correct unwrapp mapping.
	// if the fingerprint fails we just continue with original ACE mapping and let the actual ACE
	// mapper deal with the error (as it would if unwrapACE preference is not enabled)
	if mapping == "ACE" {
		if cart.env.Prefs.UnwrapACE.Get().(bool) {
			var ok bool
			ok, mapping = fingerprintAce(cartload, true)
			if ok {
				logger.Logf(cart.env, "cartridge", "ACE wrapping suggested but %s preferred", mapping)
				cartload.Mapping = mapping
			}
		}
	}

	switch mapping {
	case "2K":
		cart.mapper, err = newAtari2k(cart.env)
	case "4K":
		cart.mapper, err = newAtari4k(cart.env)
	case "F8":
		cart.mapper, err = newAtari8k(cart.env)
	case "WF8":
		cart.mapper, err = newWF8(cart.env)
	case "F6":
		cart.mapper, err = newAtari16k(cart.env)
	case "F4":
		cart.mapper, err = newAtari32k(cart.env)
	case "2KSC":
		cart.mapper, err = newAtari2k(cart.env)
		forceSuperchip = true
	case "4KSC":
		cart.mapper, err = newAtari4k(cart.env)
		forceSuperchip = true
	case "F8SC":
		cart.mapper, err = newAtari8k(cart.env)
		forceSuperchip = true
	case "F6SC":
		cart.mapper, err = newAtari16k(cart.env)
		forceSuperchip = true
	case "F4SC":
		cart.mapper, err = newAtari32k(cart.env)
		forceSuperchip = true
	case "CV":
		cart.mapper, err = newCommaVid(cart.env)
	case "FA":
		cart.mapper, err = newCBS(cart.env)
	case "FA2":
		cart.mapper, err = newFA2(cart.env)
	case "FE":
		cart.mapper, err = newSCABS(cart.env)
	case "E0":
		cart.mapper, err = newParkerBros(cart.env)
	case "E7":
		cart.mapper, err = newMnetwork(cart.env)
	case "JANE":
		cart.mapper, err = newJANE(cart.env)
	case "3F":
		cart.mapper, err = newTigervision(cart.env)
	case "UA":
		cart.mapper, err = newUA(cart.env)
	case "AR":
		cart.mapper, err = supercharger.NewSupercharger(cart.env)
	case "DF":
		cart.mapper, err = newDF(cart.env)
	case "3E":
		cart.mapper, err = new3e(cart.env)
	case "E3P":
		// synonym for 3E+
		fallthrough
	case "E3+":
		// synonym for 3E+
		fallthrough
	case "3E+":
		cart.mapper, err = new3ePlus(cart.env)
	case "EF":
		cart.mapper, err = newEF(cart.env)
	case "EFSC":
		cart.mapper, err = newEF(cart.env)
		forceSuperchip = true
	case "BF":
		cart.mapper, err = newBF(cart.env)
	case "BFSC":
		cart.mapper, err = newBF(cart.env)
		forceSuperchip = true
	case "SB":
		cart.mapper, err = newSuperbank(cart.env)
	case "WD":
		cart.mapper, err = newWicksteadDesign(cart.env)
	case "DPC":
		cart.mapper, err = newDPC(cart.env, cartload)
	case "DPC+":
		cart.mapper, err = dpcplus.NewDPCplus(cart.env, "DPC+")
	case "DPCP":
		cart.mapper, err = dpcplus.NewDPCplus(cart.env, "DPCP")

	case "CDF":
		cart.mapper, err = cdf.NewCDF(cart.env, "CDFJ")
	case "CDF0":
		cart.mapper, err = cdf.NewCDF(cart.env, "CDF0")
	case "CDF1":
		cart.mapper, err = cdf.NewCDF(cart.env, "CDF1")
	case "CDFJ":
		cart.mapper, err = cdf.NewCDF(cart.env, "CDFJ")
	case "CDFJ+":
		cart.mapper, err = cdf.NewCDF(cart.env, "CDFJ+")

	case "MVC":
		cart.mapper, err = moviecart.NewMoviecart(cart.env)
	case "ACE":
		cart.mapper, err = ace.NewAce(cart.env)
	case "ELF":
		cart.mapper, err = elf.NewElf(cart.env, false)

	case "ELF_in_ACE":
		cart.mapper, err = elf.NewElf(cart.env, true)

	case unrecognisedMapper:
		return fmt.Errorf("cartridge: unrecognised mapper")

	default:
		return fmt.Errorf("cartridge: unrecognised mapper (%s)", mapping)
	}
	if err != nil {
		return fmt.Errorf("cartridge: %w", err)
	}

	// if the forceSuperchip flag has been raised or if cartridge mapper
	// implements the optionalSuperChip interface then try to add the additional
	// RAM
	if forceSuperchip {
		if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
			superchip.AddSuperchip(true)
		} else {
			logger.Logf(cart.env, "cartridge", "cannot add superchip to %s mapper", cart.ID())
		}
	} else if auto {
		if superchip, ok := cart.mapper.(mapper.OptionalSuperchip); ok {
			superchip.AddSuperchip(false)
		}
	}

	// if this is a moviecart cartridge then return now without checking for plusrom
	if _, ok := cart.mapper.(*moviecart.Moviecart); ok {
		return nil
	}

	// in addition to the regular fingerprint we also check to see if this
	// is PlusROM cartridge (which can be combined with a regular cartridge
	// format)
	if cart.fingerprintPlusROM(cartload) {
		plus, err := plusrom.NewPlusROM(cart.env, cart.mapper, cartload)

		if err != nil {
			if errors.Is(err, plusrom.NotAPlusROM) {
				logger.Log(cart.env, "cartridge", err)
				return nil
			}
			if errors.Is(err, plusrom.CannotAdoptROM) {
				logger.Log(cart.env, "cartridge", err)
				return nil
			}
			return fmt.Errorf("cartridge: %w", err)
		}

		logger.Logf(cart.env, "cartridge", "%s cartridge contained in PlusROM", cart.ID())

		// we've wrapped the main cartridge mapper inside the PlusROM
		// mapper and we need to point the mapper field to the the new
		// PlusROM instance
		cart.mapper = plus
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

// SetBank sets the current bank of the cartridge
func (cart *Cartridge) SetBank(bank string) error {
	if set, ok := cart.mapper.(mapper.SelectableBank); ok {
		return set.SetBank(bank)
	}
	return fmt.Errorf("cartridge: %s does not support setting of bank", cart.mapper.ID())
}

// CommandExtension returns an instance of commandline.Commands suitable for commandline
// tab-completion. Implements commandline.Extension interface
func (cart *Cartridge) CommandExtension(extension string) *commandline.Commands {
	if com, ok := cart.mapper.(commandline.Extension); ok {
		return com.CommandExtension(extension)
	}
	return nil
}

// ParseCommand forwards a terminal command to the mapper
func (cart *Cartridge) ParseCommand(w io.Writer, command string) error {
	if com, ok := cart.mapper.(mapper.TerminalCommand); ok {
		return com.ParseCommand(w, command)
	}
	return fmt.Errorf("cartridge: %s does not support any terminal commands", cart.mapper.ID())
}

// AccessPassive is called so that the cartridge can respond to changes to the
// address and data bus even when the data bus is not addressed to the cartridge.
//
// The VCS cartridge port is wired up to all 13 address lines of the 6507.
// Under normal operation, the chip-select line is used by the cartridge to
// know when to put data on the data bus. If it's not "on" then the cartridge
// does nothing.
//
// However, regardless of the chip-select line, the address and data buses can
// be monitored for activity.
//
// Notably the tigervision (3F) mapper monitors and waits for address 0x003f,
// which is in the TIA address space. When this address is triggered, the
// tigervision cartridge will use whatever is on the data bus to switch banks.
//
// Similarly, the CBS (FA) mapper will switch banks on cartridge addresses 1ff8
// to 1ffa (and mirrors) but only if the data bus has the low bit set to one.
func (cart *Cartridge) AccessPassive(addr uint16, data uint8) error {
	return cart.mapper.AccessPassive(addr, data)
}

// Step should be called every CPU cycle. The attached cartridge may or may not
// change its state as a result. In fact, very few cartridges care about this.
func (cart *Cartridge) Step(clock float32) {
	cart.mapper.Step(clock)
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
		if bus.GetRAM() == nil {
			return nil
		}
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

// GetCoProcBus returns interface to the coprocessor interface or nil if no
// coprocessor is available on the cartridge.
func (cart *Cartridge) GetCoProcBus() coprocessor.CartCoProcBus {
	if cart.hasCoProcBus {
		return cart.coprocBus
	}
	return nil
}

// GetCoProc returns interface to the coprocessor interface or nil if no
// coprocessor is available on the cartridge.
func (cart *Cartridge) GetCoProc() coprocessor.CartCoProc {
	if cart.hasCoProcBus {
		return cart.coprocBus.GetCoProc()
	}
	return nil
}

func (cart *Cartridge) GetSuperchargerFastLoad() mapper.CartSuperChargerFastLoad {
	if c, ok := cart.mapper.(mapper.CartSuperChargerFastLoad); ok {
		return c
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
	return "", fmt.Errorf("cartridge: %s does not support ROM dumping", cart.mapper.ID())
}

// SetYieldHook implements the coprocessor.CartCoProcBus interface.
func (cart *Cartridge) SetYieldHook(hook coprocessor.CartYieldHook) {
	if cart.hasCoProcBus {
		cart.coprocBus.SetYieldHook(hook)
	}
}

// CoProcExecutionState implements the coprocessor.CartCoProcBus interface
//
// If cartridge does not have a coprocessor then an empty instance of
// mapper.CoProcExecutionState is returned
func (cart *Cartridge) CoProcExecutionState() coprocessor.CoProcExecutionState {
	if cart.hasCoProcBus {
		return cart.coprocBus.CoProcExecutionState()
	}
	return coprocessor.CoProcExecutionState{}
}

// BusStuff implements the mapper.CartBusStuff interface
func (cart *Cartridge) BusStuff() (uint8, bool) {
	if cart.hasBusStuff {
		return cart.busStuff.BusStuff()
	}
	return 0, false
}

// Patch implements the mapper.CartPatchable interface
func (cart *Cartridge) Patch(offset int, data uint8) error {
	if cart, ok := cart.mapper.(mapper.CartPatchable); ok {
		return cart.Patch(offset, data)
	}
	return fmt.Errorf("cartridge is not patchable")
}
