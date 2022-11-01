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

package ace

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Ace implements the mapper.CartMapper interface.
type Ace struct {
	instance *instance.Instance
	dev      mapper.CartCoProcDeveloper

	version string
	arm     *arm.ARM
	mem     *aceMemory

	// the hook that handles cartridge yields
	yieldHook mapper.CartYieldHook

	// parallelARM is true whenever the address bus is not a cartridge address (ie.
	// a TIA or RIOT address). this means that the arm is running unhindered
	// and will not have yielded for that colour clock
	parallelARM bool

	// armState is a copy of the ARM's state at the moment of the most recent
	// Snapshot. it's used only suring a Plumb() operation
	armState *arm.ARMState
}

// NewAce is the preferred method of initialisation for the Ace type.
func NewAce(instance *instance.Instance, version string, data []byte) (mapper.CartMapper, error) {
	cart := &Ace{
		instance: instance,
		version:  version,
	}

	var err error
	cart.mem, err = newAceMemory(version, data)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(cart.mem.model, cart.instance.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)

	logger.Logf("ACE", "vcs program: %08x to %08x", cart.mem.vcsOrigin, cart.mem.vcsMemtop)
	logger.Logf("ACE", "arm program: %08x to %08x", cart.mem.armOrigin, cart.mem.armMemtop)
	logger.Logf("ACE", "sram: %08x to %08x (%dbytes)", cart.mem.sramOrigin, cart.mem.sramMemtop, len(cart.mem.sram))
	logger.Logf("ACE", "GPIO IN: %08x to %08x", cart.mem.gpioAOrigin, cart.mem.gpioAMemtop)
	logger.Logf("ACE", "GPIO OUT: %08x to %08x", cart.mem.gpioBOrigin, cart.mem.gpioBMemtop)

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Ace) MappedBanks() string {
	return fmt.Sprintf("Bank: 0")
}

// ID implements the mapper.CartMapper interface.
func (cart *Ace) ID() string {
	return fmt.Sprintf("ACE (%s)", cart.version)
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Ace) Snapshot() mapper.CartMapper {
	n := *cart

	// taking a snapshot of ARM state via the ARM itself can cause havoc if
	// this instance of the cart is not current (because the ARM pointer itself
	// may be stale or pointing to another emulation)
	if cart.armState == nil {
		n.armState = cart.arm.Snapshot()
	} else {
		n.armState = cart.armState.Snapshot()
	}

	n.mem = cart.mem.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Ace) PlumbFromDifferentEmulation() {
	if cart.armState == nil {
		panic("cannot plumb this ACE instance because the ARM state is nil")
	}
	cart.arm = arm.NewARM(cart.mem.model, cart.instance.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Ace) Plumb() {
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
}

// Reset implements the mapper.CartMapper interface.
func (cart *Ace) Reset() {
}

// Read implements the mapper.CartMapper interface.
func (cart *Ace) Read(addr uint16, passive bool) (uint8, error) {
	if passive {
		cart.Listen(addr|memorymap.OriginCart, 0x00)
	}
	return cart.mem.gpioB[fromArm_data], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *Ace) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if passive || poke {
		return nil
	}

	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Ace) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Ace) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *Ace) Patch(_ int, _ uint8) error {
	return curated.Errorf("ACE: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *Ace) Listen(addr uint16, data uint8) {
	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.parallelARM = (addr&memorymap.OriginCart != memorymap.OriginCart)

	// start profiling before the run sequence
	if cart.dev != nil {
		cart.dev.StartProfiling()
		defer cart.dev.ProcessProfiling()
	}

	// set data first and continue once. this seems to be necessary to allow
	// the PlusROM exit routine to work correctly
	cart.mem.gpioB[toArm_data] = data

	yld, _, _ := cart.arm.Run()
	for yld != mapper.YieldForVCS {
		cart.yieldHook.CartYield(yld)
		yld, _, _ = cart.arm.Run()
	}

	// set address and continue x4
	cart.mem.gpioA[toArm_address] = uint8(addr)
	cart.mem.gpioA[toArm_address+1] = uint8(addr >> 8)

	yld, _, _ = cart.arm.Run()
	for yld != mapper.YieldForVCS {
		cart.yieldHook.CartYield(yld)
		yld, _, _ = cart.arm.Run()
	}

	yld, _, _ = cart.arm.Run()
	for yld != mapper.YieldForVCS {
		cart.yieldHook.CartYield(yld)
		yld, _, _ = cart.arm.Run()
	}

	yld, _, _ = cart.arm.Run()
	for yld != mapper.YieldForVCS {
		cart.yieldHook.CartYield(yld)
		yld, _, _ = cart.arm.Run()
	}

	yld, _, _ = cart.arm.Run()
	for yld != mapper.YieldForVCS {
		cart.yieldHook.CartYield(yld)
		yld, _, _ = cart.arm.Run()
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *Ace) Step(clock float32) {
	cart.arm.Step(clock)
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Ace) CopyBanks() []mapper.BankContent {
	return nil
}

// implements arm.CartridgeHook interface.
func (cart *Ace) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

// BusStuff implements the mapper.CartBusStuff interface.
func (cart *Ace) BusStuff() (uint8, bool) {
	if cart.mem.gpioB[gpio_mode] == 0x55 && cart.mem.gpioB[gpio_mode+1] == 0x55 {
		cart.mem.gpioB[gpio_mode] = 0x00
		cart.mem.gpioB[gpio_mode+1] = 0x00
		return cart.mem.gpioB[fromArm_data], true
	}
	return 0, false
}

// CoProcID implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcID() string {
	return cart.arm.CoProcID()
}

// SetDisassembler implements the mapper.CartCoProc interface.
func (cart *Ace) SetDisassembler(disasm mapper.CartCoProcDisassembler) {
	cart.arm.SetDisassembler(disasm)
}

// SetDeveloper implements the mapper.CartCoProc interface.
func (cart *Ace) SetDeveloper(dev mapper.CartCoProcDeveloper) {
	cart.dev = dev
	cart.arm.SetDeveloper(dev)
}

// ExecutableOrigin implements the mapper.CartCoProcNonRelocatable interface.
func (cart *Ace) ExecutableOrigin() uint32 {
	return cart.mem.resetPC
}

// CoProcState implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcState() mapper.CoProcState {
	if cart.parallelARM {
		return mapper.CoProcParallel
	}
	return mapper.CoProcStrongARMFeed
}

// BreakpointsDisable implements the mapper.CartCoProc interface.
func (cart *Ace) BreakpointsDisable(disable bool) {
	cart.arm.BreakpointsDisable(disable)
}

// SetYieldHook implements the mapper.CartCoProc interface.
func (cart *Ace) SetYieldHook(hook mapper.CartYieldHook) {
	cart.yieldHook = hook
}
