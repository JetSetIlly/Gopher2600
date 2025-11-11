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
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/notifications"
)

// Elf implements the mapper.CartMapper interface.
type Elf struct {
	env     *environment.Environment
	version string

	arm *arm.ARM
	mem *elfMemory

	// the hook that handles cartridge yields
	yieldHook coprocessor.CartYieldHook

	// armState is a copy of the ARM's state at the moment of the most recent
	// Snapshot. it's used only suring a Plumb() operation
	armState *arm.ARMState

	// commandline extensions
	commands *commandline.Commands

	// the initial state created immediately after creation [at the end of NewELF()]
	// we use these to return the mapper to the initial state when Reset() is called
	resetStateARM *arm.ARMState
	resetStateMem *elfMemory

	// the reset doesn't work correctly when the cartridge has just been recreated/reset. this
	// inhibit counter isn't a great solution but it works by inhibiting the reset procedure from
	// firing before the count reaches zero
	//
	// set on Elf create and also after a successful reset
	resetInhibit int
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
func NewElf(env *environment.Environment, inACE bool) (mapper.CartMapper, error) {
	r := &elfReaderAt{}

	// open file in the normal way and read all data. close the file
	// immediately once all data is rad
	o, err := os.Open(env.Loader.Filename)
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

	// if ef.FileHeader.Type != elf.ET_REL {
	// 	return nil, fmt.Errorf("ELF: is not relocatable")
	// }

	// big endian byte order is probably fine but we've not tested it
	if ef.FileHeader.ByteOrder != binary.LittleEndian {
		return nil, fmt.Errorf("ELF: is not little-endian")
	}

	cart := &Elf{
		env:       env,
		yieldHook: coprocessor.StubCartYieldHook{},
	}

	cart.commands, err = newCommands()
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}

	cart.mem = newElfMemory(cart.env)
	cart.arm = arm.NewARM(cart.env, cart.mem.model, cart.mem, cart)
	cart.arm.CycleDuringImmediateMode(true)
	cart.mem.Plumb(cart.env, cart.arm)
	err = cart.mem.decode(ef)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}

	cart.arm.SetByteOrder(ef.ByteOrder)
	cart.mem.vcsInitBusStuffing()

	// defer VCS reset until the VCS tries to read the reset address

	// run arm initialisation functions if present. next call to arm.Run() will
	// cause the main function to execute
	err = cart.mem.runInitialisation(cart.arm)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}

	// send notification that some symbols in the ELF remain unresolved
	if cart.mem.unresolvedSymbols {
		if cart.env.Prefs.ARM.UndefinedSymbolWarning.Get().(bool) {
			cart.env.Notifications.Notify(notifications.NotifyElfUndefinedSymbols)
		}
	}

	// create snapshot of this initial state
	cart.resetStateARM = cart.arm.Snapshot()
	cart.resetStateMem = cart.mem.Snapshot()
	cart.resetInhibit = 4

	return cart, nil
}

// Reset implements the mapper.CartMapper interface.
func (cart *Elf) Reset() error {
	if cart.resetInhibit > 0 {
		cart.resetInhibit--
		return nil
	}
	cart.mem.inhibitStrongarmAccess = true
	defer func() { cart.mem.inhibitStrongarmAccess = false }()
	cart.mem = cart.resetStateMem.Snapshot()
	cart.mem.Plumb(cart.env, cart.arm)
	cart.arm.Plumb(cart.env, cart.resetStateARM, cart.mem, cart)
	cart.yieldHook = &coprocessor.StubCartYieldHook{}
	cart.resetInhibit = 1
	return nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Elf) MappedBanks() string {
	return ""
}

// List of IDs that can be returned by the ID() function.
const (
	IdElf        = "ELF"
	IdElfWithPXE = "ELF (with PXE)"
)

// ID implements the mapper.CartMapper interface.
func (cart *Elf) ID() string {
	if cart.mem.pxe.enabled {
		return IdElfWithPXE
	}
	return IdElf
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
	cart.arm = arm.NewARM(cart.env, cart.mem.model, cart.mem, cart)
	cart.mem.Plumb(cart.env, cart.arm)
	cart.arm.Plumb(cart.env, cart.armState, cart.mem, cart)
	cart.armState = nil
	cart.yieldHook = &coprocessor.StubCartYieldHook{}
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Elf) Plumb(env *environment.Environment) {
	// very important we inhibit strongarm access for the duration of the
	// plumbing process. see comment in the inhibitStrongAccess declaration for
	// explanation
	cart.mem.inhibitStrongarmAccess = true
	defer func() { cart.mem.inhibitStrongarmAccess = false }()

	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.mem.Plumb(cart.env, cart.arm)
	cart.arm.Plumb(cart.env, cart.armState, cart.mem, cart)
	cart.armState = nil
}

// reset is distinct from Reset(). this reset function is implied by the
// reading of the reset address.
func (cart *Elf) reset() {
	// stream bytes rather than injecting them into the VCS as they arrive
	cart.mem.stream.active = !cart.mem.stream.disabled

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
		cart.mem.setStrongArmFunction(vcsLibInit)
	}

	// set arguments for initial execution of ARM program
	systemType := argSystemType_NTSC
	switch cart.env.TV.GetFrameInfo().Spec.ID {
	case "NTSC":
		systemType = argSystemType_NTSC
	case "PAL":
		systemType = argSystemType_PAL
	case "PAL60":
		systemType = argSystemType_PAL60
	default:
		systemType = argSystemType_NTSC
	}

	flags := argFlags_NoExit

	binary.LittleEndian.PutUint32(cart.mem.args[argAddrSystemType-argOrigin:], uint32(systemType))
	binary.LittleEndian.PutUint32(cart.mem.args[argAddrClockHz-argOrigin:], uint32(cart.arm.Clk))
	binary.LittleEndian.PutUint32(cart.mem.args[argAddrFlags-argOrigin:], uint32(flags))

	cart.arm.SetInitialRegisters(argOrigin)
}

// Access implements the mapper.CartMapper interface.
func (cart *Elf) Access(addr uint16, _ bool) (uint8, uint8, error) {
	if cart.mem.stream.active {
		if !cart.mem.stream.drain {
			_ = cart.runARM(addr)
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
	return mapper.BankInfo{Sequential: true, Number: 0, IsRAM: false}
}

func (cart *Elf) runARM(addr uint16) bool {
	if cart.mem.stream.active {
		// do nothing with the ARM if the byte stream is draining
		if cart.mem.stream.drain {
			return true
		}

		// run preempted snoopDataBus() function if required
		if cart.mem.stream.snoopDataBus {
			snoopDataBus_streaming(cart.mem, addr)
			return true
		}
	}

	cart.arm.StartProfiling()
	defer cart.arm.ProcessProfiling()

	// call arm once and then check for yield conditions
	cart.mem.yield, _ = cart.arm.Run()

	// keep calling runArm() for as long as program does not need to sync with the VCS
	for cart.mem.yield.Type != coprocessor.YieldSyncWithVCS {
		// the ARM should never return YieldProgramEnded when executing code
		// from the ELF type. if it does then it is an error and we should yield
		// with YieldExecutionError
		if cart.mem.yield.Type == coprocessor.YieldProgramEnded {
			cart.mem.yield.Type = coprocessor.YieldExecutionError
			cart.mem.yield.Error = fmt.Errorf("ELF does not support ProgramEnded yield")
		}

		// treat infinite loops like a YieldSyncWithVCS
		if cart.mem.yield.Type == coprocessor.YieldInfiniteLoop {
			return true
		}

		switch cart.yieldHook.CartYield(cart.mem.yield) {
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
	if cart.mem.stream.active && cart.mem.stream.drain {
		return nil
	}

	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.mem.parallelARM = (addr&memorymap.OriginCart != memorymap.OriginCart)

	// reset address with any mirror origin
	const resetAddrAnyMirror = (cpu.Reset & memorymap.CartridgeBits) | memorymap.OriginCart

	// if address is the reset address then trigger the reset procedure
	if (addr&memorymap.CartridgeBits)|memorymap.OriginCart == resetAddrAnyMirror {
		// after this call to cart reset, the cartridge will be wanting to run
		// the vcsEmulationInit() strongarm function
		cart.reset()
	}

	// set GPIO data and address information
	cart.mem.gpio.data[DATA_IDR] = data
	cart.mem.gpio.data[ADDR_IDR] = uint8(addr)
	cart.mem.gpio.data[ADDR_IDR+1] = uint8(addr >> 8)

	// if byte-streaming is active then the access is relatively simple
	if cart.mem.stream.active {
		_ = cart.runARM(addr)
		return nil
	}

	// handle ARM synchronisation for non-byte-streaming mode. the sequence of
	// calls to runARM() and whatever strongarm function might be active was
	// arrived through experimentation. a more efficient way of doing this
	// hasn't been discovered yet

	runStrongarm := func() bool {
		if cart.mem.strongarm.running.function == nil {
			return false
		}
		cart.mem.strongarm.running.function(cart.mem)
		if cart.mem.strongarm.running.function == nil {
			if cart.runARM(addr) {
				if cart.mem.strongarm.running.function != nil {
					cart.mem.strongarm.running.function(cart.mem)
				}
			}
		}
		return true
	}

	if runStrongarm() {
		return nil
	}

	if cart.runARM(addr) {
		if runStrongarm() {
			return nil
		}

		if cart.runARM(addr) {
			if runStrongarm() {
				return nil
			}

			_ = cart.runARM(addr)
		}
	}

	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *Elf) Step(clock float32) {
	cart.arm.Step(clock)
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *Elf) CopyBanks() []mapper.BankContent {
	return nil
}

// implements arm.CartridgeHook interface.
func (cart *Elf) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

// BusStuff implements the mapper.CartBusStuff interface.
func (cart *Elf) BusStuff() (uint8, bool) {
	if !cart.mem.usesBusStuffing {
		return 0, false
	}

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

// ReferenceStatic implements the mapper.CartStaticBus interface.
func (cart *Elf) ReferenceStatic() mapper.CartStatic {
	return cart.mem
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

// CoProcSourceDebugging implements the source coprocessor.CartCoProcSourceDebugging interface
func (cart *Elf) CoProcSourceDebugging() {
	// streaming can interfere with breakpoint recovery
	cart.mem.stream.disabled = true
}

// Section implements the coprocessor.CartCoProcELF interface.
func (cart *Elf) Section(name string) ([]uint8, uint32) {
	if sec, ok := cart.mem.sectionsByName[name]; ok {
		return sec.data, sec.origin
	}
	return nil, 0
}

func (cart *Elf) ExecutableSections() []string {
	var x []string
	for _, sec := range cart.mem.sections {
		if sec.executable() {
			x = append(x, sec.name)
		}
	}
	return x
}

// DWARF implements the coprocessor.CartCoProcELF interface.
func (cart *Elf) DWARF() (*dwarf.Data, error) {
	get := func(name string) []byte {
		s, ok := cart.mem.sectionsByName[name]
		if ok {
			return s.data
		}
		return nil
	}

	d, err := dwarf.New(
		get(".debug_abbrev"),
		get(".debug_aranges"),
		get(".debug_frame"),
		get(".debug_info"),
		get(".debug_line"),
		get(".debug_pubnames"),
		get(".debug_ranges"),
		get(".debug_str"),
	)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// ByteOrder implements the coprocessor.CartCoProcELF interface.
func (cart *Elf) ByteOrder() binary.ByteOrder {
	return cart.mem.byteOrder
}

// Symbols implements the coprocessor.CartCoProcELF interface.
func (cart *Elf) Symbols() []elf.Symbol {
	return cart.mem.symbols
}
