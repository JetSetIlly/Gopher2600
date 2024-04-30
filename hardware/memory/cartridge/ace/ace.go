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
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Ace implements the mapper.CartMapper interface.
type Ace struct {
	env *environment.Environment

	arm *arm.ARM
	mem *aceMemory

	// the hook that handles cartridge yields
	yieldHook coprocessor.CartYieldHook

	// armState is a copy of the ARM's state at the moment of the most recent
	// Snapshot. it's used only during a Plumb() operation
	armState *arm.ARMState
}

// NewAce is the preferred method of initialisation for the Ace type.
func NewAce(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("ACE: %w", err)
	}

	cart := &Ace{
		env:       env,
		yieldHook: coprocessor.StubCartYieldHook{},
	}

	cart.mem, err = newAceMemory(env, data, cart.env.Prefs.ARM)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(cart.env, cart.mem.model, cart.mem, cart)
	cart.mem.Plumb(cart.arm)

	logger.Logf(env, "ACE", "ccm: %08x to %08x", cart.mem.ccmOrigin, cart.mem.ccmMemtop)
	logger.Logf(env, "ACE", "flash: %08x to %08x", cart.mem.downloadOrigin, cart.mem.downloadMemtop)
	logger.Logf(env, "ACE", "buffer: %08x to %08x", cart.mem.bufferOrigin, cart.mem.bufferMemtop)
	logger.Logf(env, "ACE", "gpio: %08x to %08x", cart.mem.gpioOrigin, cart.mem.gpioMemtop)

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
	cart.arm = arm.NewARM(cart.env, cart.mem.model, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
	cart.yieldHook = coprocessor.StubCartYieldHook{}
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
		return cart.mem.gpio[DATA_ODR-cart.mem.gpioOrigin], mapper.CartDrivenPins, nil
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

func (cart *Ace) runARM() bool {
	// call arm once and then check for yield conditions
	var cycles float32
	cart.mem.yield, cycles = cart.arm.Run()
	cart.mem.cycles += cycles

	// keep calling runArm() for as long as program does not need to sync with the VCS
	for cart.mem.yield.Type != coprocessor.YieldSyncWithVCS {
		switch cart.yieldHook.CartYield(cart.mem.yield.Type) {
		case coprocessor.YieldHookEnd:
			return false
		case coprocessor.YieldHookContinue:
			cart.mem.yield, cycles = cart.arm.Run()
			cart.mem.cycles += cycles
		}
	}
	return true
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Ace) AccessPassive(addr uint16, data uint8) error {
	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.mem.parallelARM = (addr&memorymap.OriginCart != memorymap.OriginCart)

	// start profiling before the run sequence
	cart.arm.StartProfiling()
	defer cart.arm.ProcessProfiling()

	// set data first and continue once. this seems to be necessary to allow
	// the PlusROM exit routine to work correctly
	cart.mem.gpio[DATA_IDR-cart.mem.gpioOrigin] = data
	_ = cart.runARM()

	// set address for ARM program
	cart.mem.gpio[ADDR_IDR-cart.mem.gpioOrigin] = uint8(addr)
	cart.mem.gpio[ADDR_IDR-cart.mem.gpioOrigin+1] = uint8(addr >> 8)

	// continue and wait for the sixth YieldSyncWithVCS...
	for cart.mem.armInterruptCt < 6 {
		cart.runARM()
	}
	cart.mem.armInterruptCt = 0

	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *Ace) Step(clock float32) {
	if cart.mem.cycles > 0 {
		cart.mem.cycles -= float32(cart.env.Prefs.ARM.Clock.Get().(float64)) / clock
	} else {
		cart.arm.Step(clock)
	}
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Ace) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, 1)
	c[0] = mapper.BankContent{Number: 0,
		Data:    cart.mem.buffer,
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
		cart.mem.gpio[DATA_MODER-cart.mem.gpioOrigin] = 0x00
		cart.mem.gpio[DATA_MODER-cart.mem.gpioOrigin+1] = 0x00
		return cart.mem.gpio[DATA_ODR-cart.mem.gpioOrigin], true
	}
	return 0, false
}

// ExecutableOrigin implements the coprocessor.CartCoProcRelocatable interface.
func (cart *Ace) ExecutableOrigin() uint32 {
	return cart.mem.resetPC
}

// CoProcExecutionState implements the coprocessor.CartCoProcBus interface.
func (cart *Ace) CoProcExecutionState() coprocessor.CoProcExecutionState {
	if cart.mem.parallelARM {
		return coprocessor.CoProcExecutionState{
			Sync:  coprocessor.CoProcParallel,
			Yield: cart.mem.yield,
		}
	}
	return coprocessor.CoProcExecutionState{
		Sync:  coprocessor.CoProcStrongARMFeed,
		Yield: cart.mem.yield,
	}
}

// CoProcRegister implements the coprocessor.CartCoProcBus interface.
func (cart *Ace) GetCoProc() coprocessor.CartCoProc {
	return cart.arm
}

// SetYieldHook implements the coprocessor.CartCoProcBus interface.
func (cart *Ace) SetYieldHook(hook coprocessor.CartYieldHook) {
	cart.yieldHook = hook
}
