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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Elf implements the mapper.CartMapper interface.
type Elf struct {
	env *environment.Environment

	version   string
	pathToROM string

	arm *arm.ARM
	mem *elfMemory

	// the hook that handles cartridge yields
	yieldHook coprocessor.CartYieldHook

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
		yieldHook: coprocessor.StubCartYieldHook{},
	}

	cart.mem = newElfMemory(env)
	cart.arm = arm.NewARM(cart.mem.model, cart.env.Prefs.ARM, cart.mem, cart)
	cart.mem.Plumb(cart.arm)
	err = cart.mem.decode(ef)
	if err != nil {
		return cart, nil
	}

	cart.arm.SetByteOrder(ef.ByteOrder)
	cart.mem.busStuffingInit()

	// defer VCS reset until the VCS tries to read the cpubus.Reset address

	// run arm initialisation functions if present. next call to arm.Run() will
	// cause the main function to execute
	err = cart.mem.runInitialisation(cart.arm)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}

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
	cart.yieldHook = &coprocessor.StubCartYieldHook{}
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
	// stream bytes rather than injecting them into the VCS as they arrive
	cart.mem.stream.active = true

	// initialise ROM for the VCS
	if cart.mem.stream.active {
		cart.mem.stream.push(streamEntry{
			addr: 0x1ffc,
			data: 0x00,
		})
		cart.mem.stream.push(streamEntry{
			addr: 0x1ffd,
			data: 0x10,
		})
		cart.mem.strongarm.nextRomAddress = 0x1000
		cart.mem.stream.startDrain()
	} else {
		cart.mem.setStrongArmFunction(vcsEmulationInit)
	}

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
	if cart.mem.stream.active {
		if !cart.mem.stream.drain {
			cart.runARM(addr)
		}
		if addr == cart.mem.stream.peek().addr&memorymap.CartridgeBits {
			e := cart.mem.stream.pull()
			cart.mem.gpio.data[DATA_ODR] = e.data
		}
	}
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

func (cart *Elf) runARM(addr uint16) bool {
	if cart.mem.stream.active {
		// do nothing with the ARM if the byte stream is draining
		if cart.mem.stream.drain {
			return true
		}

		// run preempted snoopDataBus() function if required
		if cart.mem.stream.preemptedSnoopDataBus != nil {
			if addr != cart.mem.strongarm.nextRomAddress {
				return true
			}
			cart.mem.stream.preemptedSnoopDataBus(cart.mem)
		}
	}

	cart.arm.StartProfiling()
	defer cart.arm.ProcessProfiling()

	// call arm once and then check for yield conditions
	cart.mem.yield, _ = cart.arm.Run()

	// keep calling runArm() for as long as program does not need to sync with the VCS...
	for cart.mem.yield.Type != coprocessor.YieldSyncWithVCS {
		// ... or if the yield hook says to return to the VCS immediately
		switch cart.yieldHook.CartYield(cart.mem.yield.Type) {
		case coprocessor.YieldHookEnd:
			return false
		case coprocessor.YieldHookContinue:
			cart.mem.yield, _ = cart.arm.Run()
		}
	}
	return true
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Elf) AccessPassive(addr uint16, data uint8) error {
	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.mem.parallelARM = (addr&memorymap.OriginCart != memorymap.OriginCart)

	// if address is the reset address then trigger the reset procedure
	if (addr&memorymap.CartridgeBits)|memorymap.OriginCart == (cpubus.Reset&memorymap.CartridgeBits)|memorymap.OriginCart {
		// after this call to cart reset, the cartridge will be wanting to run
		// the vcsEmulationInit() strongarm function
		cart.reset()
	}

	// set GPIO data and address information
	cart.mem.gpio.data[DATA_IDR] = data
	cart.mem.gpio.data[ADDR_IDR] = uint8(addr)
	cart.mem.gpio.data[ADDR_IDR+1] = uint8(addr >> 8)

	// handle ARM synchronisation for non-byte-streaming mode. the sequence of
	// calls to runARM() and whatever strongarm function might be active was
	// arrived through experimentation. a more efficient way of doing this
	// hasn't been discovered yet
	if !cart.mem.stream.active {
		runStrongarm := func() bool {
			if cart.mem.strongarm.running.function == nil {
				return false
			}
			cart.mem.strongarm.running.function(cart.mem)
			if cart.mem.strongarm.running.function == nil {
				cart.runARM(addr)
				if cart.mem.strongarm.running.function != nil {
					cart.mem.strongarm.running.function(cart.mem)
				}
			}
			return true
		}

		if runStrongarm() {
			return nil
		}

		cart.runARM(addr)
		if runStrongarm() {
			return nil
		}

		cart.runARM(addr)
		if runStrongarm() {
			return nil
		}

		cart.runARM(addr)

		return nil
	}

	// run ARM and strongarm function again
	cart.runARM(addr)

	return nil
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
		return 0, false
	}

	if cart.mem.stream.active {
		if cart.mem.stream.peek().busstuff {
			e := cart.mem.stream.pull()
			return e.data, true
		}
		return 0, false
	}

	if cart.mem.busStuff {
		cart.mem.busStuff = false
		return cart.mem.busStuffData, true
	}
	return 0, false
}

// ELFSection implements the coprocessor.CartCoProcRelocatable interface.
func (cart *Elf) ELFSection(name string) ([]uint8, uint32, bool) {
	if idx, ok := cart.mem.sectionsByName[name]; ok {
		s := cart.mem.sections[idx]
		return s.data, s.origin, true
	}
	return nil, 0, false
}

// CoProcExecutionState implements the coprocessor.CartCoProcBus interface.
func (cart *Elf) CoProcExecutionState() coprocessor.CoProcExecutionState {
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
func (cart *Elf) GetCoProc() coprocessor.CartCoProc {
	return cart.arm
}

// SetYieldHook implements the coprocessor.CartCoProcBus interface.
func (cart *Elf) SetYieldHook(hook coprocessor.CartYieldHook) {
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
