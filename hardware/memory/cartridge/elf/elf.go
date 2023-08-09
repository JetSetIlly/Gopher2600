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

package elf

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Elf implements the mapper.CartMapper interface.
type Elf struct {
	env *environment.Environment
	dev mapper.CartCoProcDeveloper

	version   string
	pathToROM string

	arm *arm.ARM
	mem *elfMemory

	// the hook that handles cartridge yields
	yieldHook mapper.CartYieldHook

	// armState is a copy of the ARM's state at the moment of the most recent
	// Snapshot. it's used only suring a Plumb() operation
	armState *arm.ARMState
}

// elfReaderAt is an implementation of io.ReaderAt and is used with elf.NewFile()
type elfReaderAt struct {
	// data from the file being used as the source of ELF data
	data []byte

	// the offset into the data slice where the ELF file starts
	offset int64
}

func (r *elfReaderAt) ReadAt(p []byte, start int64) (n int, err error) {
	start += r.offset

	end := start + int64(len(p))
	if end > int64(len(r.data)) {
		end = int64(len(r.data))
	}
	copy(p, r.data[start:end])

	n = int(end - start)
	if n < len(p) {
		return n, fmt.Errorf("not enough bytes in the ELF data to fill the buffer")
	}

	return n, nil

}

// NewElf is the preferred method of initialisation for the Elf type.
func NewElf(env *environment.Environment, pathToROM string, inACE bool) (mapper.CartMapper, error) {
	r := &elfReaderAt{}

	// open file in the normal way and read all data. close the file
	// immediately once all data is rad
	o, err := os.Open(pathToROM)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}
	r.data, err = io.ReadAll(o)
	if err != nil {
		o.Close()
		return nil, fmt.Errorf("ELF: %w", err)
	}
	o.Close()

	// if this is an embedded ELF file in an ACE container we need to extract
	// the ELF contents. we do this by finding the offset of the ELF file. the
	// offset is stored at initial offset 0x28
	if inACE {
		if len(r.data) < 0x30 {
			return nil, fmt.Errorf("ELF: this doesn't look like ELF data embedded in an ACE file")
		}
		r.offset = int64(r.data[0x28]) | (int64(r.data[0x29]) << 8)
	}

	// ELF file is read via our elfReaderAt instance
	ef, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}
	defer ef.Close()

	// keeping things simple. only 32bit ELF files supported
	if ef.Class != elf.ELFCLASS32 {
		return nil, fmt.Errorf("ELF: only 32bit ELF files are supported")
	}

	// sanity checks on ELF data
	if ef.FileHeader.Machine != elf.EM_ARM {
		return nil, fmt.Errorf("ELF: is not ARM")
	}
	if ef.FileHeader.Version != elf.EV_CURRENT {
		return nil, fmt.Errorf("ELF: unknown version")
	}
	if ef.FileHeader.Type != elf.ET_REL {
		return nil, fmt.Errorf("ELF: is not relocatable")
	}

	// big endian byte order is probably fine but we've not tested it
	if ef.FileHeader.ByteOrder != binary.LittleEndian {
		return nil, fmt.Errorf("ELF: is not little-endian")
	}

	cart := &Elf{
		env:       env,
		pathToROM: pathToROM,
		yieldHook: mapper.StubCartYieldHook{},
	}

	cart.mem = newElfMemory()
	cart.arm = arm.NewARM(cart.mem.model, cart.env.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	err = cart.mem.decode(ef)
	if err != nil {
		return cart, nil
	}

	cart.arm.SetByteOrder(ef.ByteOrder)

	cart.mem.busStuffingInit()

	// defer reset until the VCS tries to read the cpubus.Reset address

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Elf) MappedBanks() string {
	return fmt.Sprintf("Bank: 0")
}

// ID implements the mapper.CartMapper interface.
func (cart *Elf) ID() string {
	return "ELF"
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Elf) Snapshot() mapper.CartMapper {
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
func (cart *Elf) PlumbFromDifferentEmulation(env *environment.Environment) {
	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.arm = arm.NewARM(cart.mem.model, cart.env.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
	cart.yieldHook = &mapper.StubCartYieldHook{}
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Elf) Plumb(env *environment.Environment) {
	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.mem.Plumb(cart.arm)
	cart.arm.Plumb(cart.armState, cart.mem, cart)
	cart.armState = nil
}

// Reset implements the mapper.CartMapper interface.
func (cart *Elf) Reset() {
}

// reset is distinct from Reset(). this reset function is implied by the
// reading of the cpubus.Reset address.
func (cart *Elf) reset() {
	cart.mem.setStrongArmFunction(vcsEmulationInit)

	// set arguments for initial execution of ARM program
	cart.mem.args[argAddrSystemType-argOrigin] = argSystemType_NTSC
	cart.mem.args[argAddrClockHz-argOrigin] = 0xef
	cart.mem.args[argAddrClockHz-argOrigin+1] = 0xbe
	cart.mem.args[argAddrClockHz-argOrigin+2] = 0xad
	cart.mem.args[argAddrClockHz-argOrigin+3] = 0xde
	cart.arm.SetInitialRegisters(argOrigin)
}

// Access implements the mapper.CartMapper interface.
func (cart *Elf) Access(addr uint16, _ bool) (uint8, uint8, error) {
	cart.mem.busStuffDelay = true
	return cart.mem.gpio.data[DATA_ODR], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *Elf) AccessVolatile(addr uint16, data uint8, _ bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Elf) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Elf) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *Elf) Patch(_ int, _ uint8) error {
	return fmt.Errorf("ELF: patching unsupported")
}

func (cart *Elf) runARM() bool {
	if cart.dev != nil {
		cart.dev.StartProfiling()
		defer cart.dev.ProcessProfiling()
	}

	// call arm once and then check for yield conditions
	cart.mem.yield, _ = cart.arm.Run()

	// keep calling runArm() for as long as program does not need to sync with the VCS...
	for cart.mem.yield.Type != mapper.YieldSyncWithVCS {
		// ... or if the yield hook says to return to the VCS immediately
		switch cart.yieldHook.CartYield(cart.mem.yield.Type) {
		case mapper.YieldHookEnd:
			return false
		case mapper.YieldHookContinue:
			cart.mem.yield, _ = cart.arm.Run()
		}
		cart.mem.yield, _ = cart.arm.Run()
	}
	return true
}

// try to run strongarm function. returns success.
func (cart *Elf) runStrongarm(addr uint16, data uint8) bool {
	if cart.mem.strongarm.running.function != nil {
		cart.mem.gpio.data[DATA_IDR] = data
		cart.mem.gpio.data[ADDR_IDR] = uint8(addr)
		cart.mem.gpio.data[ADDR_IDR+1] = uint8(addr >> 8)
		cart.mem.strongarm.running.function(cart.mem)

		if cart.mem.strongarm.running.function == nil {
			cart.runARM()
			if cart.mem.strongarm.running.function != nil {
				cart.mem.strongarm.running.function(cart.mem)
			}
		}

		// if the most recently run strongarm function has instructed the ARM
		// emulation to resume immediately then we loop until we encounter one
		// which wants to yield to the VCS
		for cart.mem.resumeARMimmediately {
			cart.mem.resumeARMimmediately = false
			cart.runARM()
			if cart.mem.strongarm.running.function != nil {
				cart.mem.strongarm.running.function(cart.mem)
			}
		}

		return true
	}
	return false
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Elf) AccessPassive(addr uint16, data uint8) {
	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.mem.parallelARM = (addr&memorymap.OriginCart != memorymap.OriginCart)

	// if address is the reset address then trigger the reset procedure
	if (addr&memorymap.CartridgeBits)|memorymap.OriginCart == (cpubus.Reset&memorymap.CartridgeBits)|memorymap.OriginCart {
		cart.reset()
	}

	if cart.runStrongarm(addr, data) {
		return
	}

	// set data first and continue once. this seems to be necessary to allow
	// the PlusROM exit rountine to work correctly
	cart.mem.gpio.data[DATA_IDR] = data

	cart.runARM()
	if cart.runStrongarm(addr, data) {
		return
	}

	// set address and continue
	cart.mem.gpio.data[ADDR_IDR] = uint8(addr)
	cart.mem.gpio.data[ADDR_IDR+1] = uint8(addr >> 8)

	cart.runARM()
	if cart.runStrongarm(addr, data) {
		return
	}

	cart.runARM()
	if cart.runStrongarm(addr, data) {
		return
	}

	cart.runARM()
	if cart.runStrongarm(addr, data) {
		return
	}

	cart.runARM()
	if cart.runStrongarm(addr, data) {
		return
	}

	cart.runARM()

	// we must understand that the above synchronisation is almost certainly
	// "wrong" in the general sense. it works for the examples seen so far but
	// that means nothing
}

// Step implements the mapper.CartMapper interface.
func (cart *Elf) Step(clock float32) {
	cart.arm.Step(clock)
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Elf) CopyBanks() []mapper.BankContent {
	return nil
}

// implements arm.CartridgeHook interface.
func (cart *Elf) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

// BusStuff implements the mapper.CartBusStuff interface.
func (cart *Elf) BusStuff() (uint8, bool) {
	if cart.mem.busStuffDelay {
		cart.mem.busStuffDelay = false
		return cart.mem.busStuffData, false
	}
	return cart.mem.busStuffData, cart.mem.busStuff
}

// CoProcID implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcID() string {
	return cart.arm.CoProcID()
}

// SetDisassembler implements the mapper.CartCoProc interface.
func (cart *Elf) SetDisassembler(disasm mapper.CartCoProcDisassembler) {
	cart.arm.SetDisassembler(disasm)
}

// SetDeveloper implements the mapper.CartCoProc interface.
func (cart *Elf) SetDeveloper(dev mapper.CartCoProcDeveloper) {
	cart.dev = dev
	cart.arm.SetDeveloper(dev)
}

// ELFSection implements the mapper.CartCoProcELF interface.
func (cart *Elf) ELFSection(name string) ([]uint8, uint32, bool) {
	if idx, ok := cart.mem.sectionsByName[name]; ok {
		s := cart.mem.sections[idx]
		return s.data, s.origin, true
	}
	return nil, 0, false
}

// CoProcExecutionState implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcExecutionState() mapper.CoProcExecutionState {
	if cart.mem.parallelARM {
		return mapper.CoProcExecutionState{
			Sync:  mapper.CoProcParallel,
			Yield: cart.mem.yield,
		}
	}
	return mapper.CoProcExecutionState{
		Sync:  mapper.CoProcStrongARMFeed,
		Yield: cart.mem.yield,
	}
}

// CoProcRegister implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcRegister(n int) (uint32, bool) {
	return cart.arm.Register(n)
}

// CoProcRegister implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcRegisterSet(n int, value uint32) bool {
	return cart.arm.SetRegister(n, value)
}

// CoProcStackFrame implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcStackFrame() uint32 {
	return cart.arm.StackFrame()
}

// CoProcPeek implements the mapper.CartCoProc interface.
func (cart *Elf) CoProcPeek(addr uint32) (uint32, bool) {
	return cart.mem.Read32bit(addr)
}

// BreakpointsEnable implements the mapper.CartCoProc interface.
func (cart *Elf) BreakpointsEnable(enable bool) {
	cart.arm.BreakpointsEnable(enable)
}

// SetYieldHook implements the mapper.CartCoProc interface.
func (cart *Elf) SetYieldHook(hook mapper.CartYieldHook) {
	cart.yieldHook = hook
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *Elf) GetStatic() mapper.CartStatic {
	return cart.mem.Snapshot()
}

// StaticWrite implements the mapper.CartStaticBus interface.
func (cart *Elf) PutStatic(segment string, idx int, data uint8) bool {
	mem, ok := cart.mem.Reference(segment)
	if !ok {
		return false
	}

	if idx >= len(mem) {
		return false
	}
	mem[idx] = data

	return true
}
