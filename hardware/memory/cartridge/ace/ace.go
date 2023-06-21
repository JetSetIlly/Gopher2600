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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Ace implements the mapper.CartMapper interface.
type Ace struct {
	env *environment.Environment
	dev mapper.CartCoProcDeveloper

	arm *arm.ARM
	mem *aceMemory

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
func NewAce(env *environment.Environment, data []byte) (mapper.CartMapper, error) {
	cart := &Ace{
		env:       env,
		yieldHook: mapper.StubCartYieldHook{},
	}

	var err error
	cart.mem, err = newAceMemory(data)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(cart.mem.model, cart.env.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)

	logger.Logf("ACE", "ccm: %08x to %08x", cart.mem.sramOrigin, cart.mem.sramMemtop)
	logger.Logf("ACE", "flash: %08x to %08x", cart.mem.flashOrigin, cart.mem.flashMemtop)
	logger.Logf("ACE", "vcs program: %08x to %08x", cart.mem.flashVCSOrigin, cart.mem.flashVCSMemtop)
	logger.Logf("ACE", "arm program: %08x to %08x", cart.mem.flashARMOrigin, cart.mem.flashARMMemtop)

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Ace) MappedBanks() string {
	return fmt.Sprintf("Bank: 0")
}

// ID implements the mapper.CartMapper interface.
func (cart *Ace) ID() string {
	return fmt.Sprintf("ACE (%s)", cart.mem.header.version)
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
func (cart *Ace) PlumbFromDifferentEmulation(env *environment.Environment) {
	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ACE instance because the ARM state is nil")
	}
	cart.arm = arm.NewARM(cart.mem.model, cart.env.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Ace) Plumb(env *environment.Environment) {
	cart.env = env
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

// Access implements the mapper.CartMapper interface.
func (cart *Ace) Access(addr uint16, _ bool) (uint8, uint8, error) {
	if cart.mem.isDataModeOut() {
		return cart.mem.gpio[DATA_ODR_idx], mapper.CartDrivenPins, nil
	}
	return 0, 0, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *Ace) AccessVolatile(addr uint16, data uint8, _ bool) error {
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
	return fmt.Errorf("ACE: patching unsupported")
}

func (cart *Ace) runARM() {
	// call arm once and then check for yield conditions
	yld, _ := cart.arm.Run()

	// keep calling runArm() for as long as program does not need to sync with the VCS...
	for yld != mapper.YieldSyncWithVCS {
		// ... or if the yield hook says to return to the VCS immedtiately
		if cart.yieldHook.CartYield(yld) {
			return
		}
		yld, _ = cart.arm.Run()
	}
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Ace) AccessPassive(addr uint16, data uint8) {
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
	if !cart.mem.isDataModeOut() {
		cart.mem.gpio[DATA_IDR_idx] = data
	}

	cart.runARM()

	// set address for ARM program
	cart.mem.gpio[ADDR_IDR_idx] = uint8(addr)
	cart.mem.gpio[ADDR_IDR_idx+1] = uint8(addr >> 8)

	// continue and wait for the fourth YieldSyncWithVCS...
	for i := 0; i < 4; i++ {
		cart.runARM()
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *Ace) Step(clock float32) {
	cart.arm.Step(clock)
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Ace) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, 1)
	c[0] = mapper.BankContent{Number: 0,
		Data:    cart.mem.flashVCS,
		Origins: []uint16{memorymap.OriginCart},
	}
	return c
}

// implements arm.CartridgeHook interface.
func (cart *Ace) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

// BusStuff implements the mapper.CartBusStuff interface.
func (cart *Ace) BusStuff() (uint8, bool) {
	if cart.mem.isDataModeOut() {
		cart.mem.gpio[DATA_MODER_idx] = 0x00
		cart.mem.gpio[DATA_MODER_idx+1] = 0x00
		return cart.mem.gpio[DATA_ODR_idx], true
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
	return 0
}

// CoProcState implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcState() mapper.CoProcState {
	if cart.parallelARM {
		return mapper.CoProcParallel
	}
	return mapper.CoProcStrongARMFeed
}

// CoProcRegister implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcRegister(n int) (uint32, bool) {
	if n > 15 {
		return 0, false
	}
	return cart.arm.Registers()[n], true
}

// CoProcRegister implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcRegisterSet(n int, value uint32) bool {
	return cart.arm.SetRegister(n, value)
}

// CoProcStackFrame implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcStackFrame() uint32 {
	return cart.arm.StackFrame()
}

// CoProcRead8bit implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcRead8bit(addr uint32) (uint8, bool) {
	return cart.mem.Read8bit(addr)
}

// CoProcRead16bit implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcRead16bit(addr uint32) (uint16, bool) {
	return cart.mem.Read16bit(addr)
}

// CoProcRead32bit implements the mapper.CartCoProc interface.
func (cart *Ace) CoProcRead32bit(addr uint32) (uint32, bool) {
	return cart.mem.Read32bit(addr)
}

// BreakpointsEnable implements the mapper.CartCoProc interface.
func (cart *Ace) BreakpointsEnable(enable bool) {
	cart.arm.BreakpointsEnable(enable)
}

// SetYieldHook implements the mapper.CartCoProc interface.
func (cart *Ace) SetYieldHook(hook mapper.CartYieldHook) {
	cart.yieldHook = hook
}
