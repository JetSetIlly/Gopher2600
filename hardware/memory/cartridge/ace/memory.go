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
	"encoding/binary"
	"fmt"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/logger"
)

type interruptARM interface {
	Interrupt()
}

type aceHeader struct {
	version       string
	driverName    string
	driverVersion uint32
	romSize       uint32
	checksum      uint32
	entry         uint32
}

const (
	aceHeaderMagic         = 0
	aceHeaderDriverName    = 8
	aceHeaderDriverVersion = 24
	aceHeaderROMSize       = 28
	aceHeaderROMChecksum   = 32
	aceHeaderEntryPoint    = 36
	aceStartOfVCSProgram   = 40
)

type aceMemory struct {
	env *environment.Environment

	header aceHeader

	model   architecture.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	gpio       []byte
	gpioOrigin uint32
	gpioMemtop uint32

	// CCM is a faster RAM than SRAM
	ccm       []byte
	ccmOrigin uint32
	ccmMemtop uint32

	// flash memory is divided into three segments
	flash       []byte
	flashOrigin uint32
	flashMemtop uint32

	// the sram and ARM segments must be consecutive
	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// minimal interface to the ARM
	arm            interruptARM
	armInterruptCt int

	// parallelARM is true whenever the address bus is not a cartridge address (ie.
	// a TIA or RIOT address). this means that the arm is running unhindered
	// and will not have yielded for that colour clock
	parallelARM bool

	// most recent yield from the coprocessor
	yield coprocessor.CoProcYield

	// count of cycles accumulated
	cycles float32
}

const maxArmInterrupCt = 6

const (
	DATA_MODER = 0x40020800
	ADDR_IDR   = 0x40020c10
	DATA_IDR   = 0x40020810
	DATA_ODR   = 0x40020814

	GPIO_ORIGIN = 0x40020800
	GPIO_MEMTOP = 0x40020cff
)

func (mem *aceMemory) isDataModeOut() bool {
	return mem.gpio[DATA_MODER-mem.gpioOrigin] == 0x55 && mem.gpio[DATA_MODER-mem.gpioOrigin+1] == 0x55
}

func (mem *aceMemory) setDataMode(out bool) {
	if out {
		mem.gpio[DATA_MODER-mem.gpioOrigin] = 0x55
		mem.gpio[DATA_MODER-mem.gpioOrigin+1] = 0x55
	} else {
		mem.gpio[DATA_MODER-mem.gpioOrigin] = 0x00
		mem.gpio[DATA_MODER-mem.gpioOrigin+1] = 0x00
	}
}

func newAceMemory(env *environment.Environment, data []byte, armPrefs *preferences.ARMPreferences) (*aceMemory, error) {
	mem := &aceMemory{
		env:   env,
		model: architecture.NewMap(architecture.PlusCart),
	}

	// CCM creation
	mem.ccm = make([]byte, 0x00010000)
	mem.ccmOrigin = mem.model.Regions["CCM"].Origin
	mem.ccmMemtop = mem.ccmOrigin + uint32(len(mem.ccm)) - 1

	// read header
	mem.header.version = string(data[:aceHeaderDriverName])
	logger.Logf(mem.env, "ACE", "header: version name: %s", mem.header.version)

	mem.header.driverName = string(data[aceHeaderDriverName:aceHeaderDriverVersion])
	logger.Logf(mem.env, "ACE", "header: driver name: %s", mem.header.driverName)

	mem.header.driverVersion = (uint32(data[aceHeaderDriverVersion])) |
		(uint32(data[aceHeaderDriverVersion+1]) << 8) |
		(uint32(data[aceHeaderDriverVersion+2]) << 16) |
		(uint32(data[aceHeaderDriverVersion+3]) << 24)
	logger.Logf(mem.env, "ACE", "header: driver version: %08x", mem.header.driverVersion)

	mem.header.romSize = (uint32(data[aceHeaderROMSize])) |
		(uint32(data[aceHeaderROMSize+1]) << 8) |
		(uint32(data[aceHeaderROMSize+2]) << 16) |
		(uint32(data[aceHeaderROMSize+3]) << 24)
	logger.Logf(mem.env, "ACE", "header: romsize: %08x", mem.header.romSize)

	mem.header.checksum = (uint32(data[aceHeaderROMChecksum])) |
		(uint32(data[aceHeaderROMChecksum+1]) << 8) |
		(uint32(data[aceHeaderROMChecksum+2]) << 16) |
		(uint32(data[aceHeaderROMChecksum+3]) << 24)
	logger.Logf(mem.env, "ACE", "header: checksum: %08x", mem.header.checksum)

	mem.header.entry = (uint32(data[aceHeaderEntryPoint])) |
		(uint32(data[aceHeaderEntryPoint+1]) << 8) |
		(uint32(data[aceHeaderEntryPoint+2]) << 16) |
		(uint32(data[aceHeaderEntryPoint+3]) << 24)
	logger.Logf(mem.env, "ACE", "header: entrypoint: %08x", mem.header.entry)

	mem.flash = data[:]
	switch mem.header.version {
	case "ACE-PC00":
		mem.flashOrigin = mem.model.Regions["Flash"].Origin
		mem.header.entry = 0x1028
		logger.Logf(mem.env, "ACE", "header: entrypoint adjusted to: %08x", mem.header.entry)
	case "ACE-UF00":
		mem.flashOrigin = mem.model.Regions["Flash"].Origin
	case "ACE-2600":
		fallthrough
	default:
		return nil, fmt.Errorf("ACE: version: %s not supported", mem.header.version)
	}
	mem.flashMemtop = mem.flashOrigin + uint32(len(mem.flash))

	// the placement of data in memory revolves around the ARM entry point
	mem.resetPC = arm.AlignTo16bits(mem.flashOrigin + mem.header.entry)
	mem.resetLR = mem.resetPC
	mem.resetSP = mem.ccmMemtop - 3

	// note the real entry point
	logger.Logf(mem.env, "ACE", "actual entrypoint: %08x", mem.resetPC)

	// define the Thumb-2 bytecode for a function whose only purpose is to jump
	// back to where it came from bytecode is for instruction "BX LR" with a
	// "true" value in R0
	nullFunction := []byte{
		0x01, 0x20, // MOV R1, #1 (the function returns true)
		0x70, 0x47, // BX LR
	}

	// placing nullFunction at end of ARM program
	nullFunctionAddress := mem.flashMemtop + 1

	// the code location of the null function must not be on a 16bit boundary
	if arm.IsAlignedTo16bits(nullFunctionAddress) {
		logger.Logf(mem.env, "ACE", "correcting alignment at end of ARM program")
		mem.flash = append(mem.flash, 0x00)
		mem.flashMemtop++
		nullFunctionAddress++
	}

	mem.flash = append(mem.flash, nullFunction...)
	mem.flashMemtop += uint32(len(nullFunction))

	// although the code location of the null function is on a 16bit boundary
	// (see above), the code is reached by interwork branching. we're using the
	// Thumb-2 instruction set so this means that the zero bit of the address
	// must be set to one
	//
	// interwork branching uses the BLX instruction. BLX ignores bit zero of the
	// address. this means that the correct (aligned) address will be used when
	// setting the program counter
	nullFunctionAddress |= 0x01

	logger.Logf(mem.env, "ACE", "null function place at %08x", nullFunctionAddress)

	// generous amount for SRAM to accomodate DPCP
	// mem.sram = make([]byte, 0x20000)
	// mem.sramOrigin = mem.model.Regions["SRAM"].Origin
	// mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram)-1)

	// choose size for the remainder of the flash memory and place at the flash
	// origin value for architecture
	const sramOverhead = 64000
	var sramSize uint32

	if len(data) < 128000-sramOverhead {
		sramSize = 128000
	} else if len(data) < 256000-sramOverhead {
		sramSize = 256000
	} else if len(data) < 512000-sramOverhead {
		sramSize = 512000
	} else {
		sramSize = sramOverhead
	}

	mem.sram = make([]byte, sramSize)
	mem.sramOrigin = mem.model.Regions["SRAM"].Origin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram)-1)

	// set virtual argument. detailed information in the PlusCart firmware
	// source:
	//
	// atari-2600-pluscart-master/source/STM32firmware/PlusCart/Src/cartridge_emulation_ACE.c

	// ROM file argument
	binary.LittleEndian.PutUint32(mem.sram, mem.flashOrigin)

	// CCM memory argument
	binary.LittleEndian.PutUint32(mem.sram[4:], mem.ccmOrigin)

	// addresses of func_reboot_into_cartridge() and emulate_firmware_cartridge()
	// for our purposes, the function needs only to jump back to the link address
	binary.LittleEndian.PutUint32(mem.sram[8:], nullFunctionAddress)
	binary.LittleEndian.PutUint32(mem.sram[12:], nullFunctionAddress)

	// system clock argument
	clk := int(armPrefs.Clock.Get().(float64) * 1000000)
	binary.LittleEndian.PutUint32(mem.sram[16:], uint32(clk))

	// ACE version number
	aceVersion := 2
	binary.LittleEndian.PutUint32(mem.sram[20:], uint32(aceVersion))

	// pluscart revision number
	plusCartRevision := 3
	binary.LittleEndian.PutUint32(mem.sram[24:], uint32(plusCartRevision))

	// GPIO addresses
	binary.LittleEndian.PutUint32(mem.sram[28:], ADDR_IDR)
	binary.LittleEndian.PutUint32(mem.sram[32:], DATA_IDR)
	binary.LittleEndian.PutUint32(mem.sram[36:], DATA_ODR)
	binary.LittleEndian.PutUint32(mem.sram[40:], DATA_MODER)

	// end of argument indicator
	copy(mem.sram[44:48], []byte{0x00, 0x26, 0xe4, 0xac})

	// GPIO pins
	mem.gpio = make([]byte, GPIO_MEMTOP-GPIO_ORIGIN+1)
	mem.gpioOrigin = GPIO_ORIGIN
	mem.gpioMemtop = GPIO_MEMTOP

	// default NOP instruction for opcode
	mem.gpio[DATA_ODR-mem.gpioOrigin] = 0xea

	return mem, nil
}

func (mem *aceMemory) Snapshot() *aceMemory {
	m := *mem
	m.gpio = make([]byte, len(mem.gpio))
	copy(m.gpio, mem.gpio)
	m.ccm = make([]byte, len(mem.ccm))
	copy(m.ccm, mem.ccm)
	m.flash = make([]byte, len(mem.flash))
	copy(m.flash, mem.flash)
	m.sram = make([]byte, len(mem.sram))
	copy(m.sram, mem.sram)
	return &m
}

// Plumb implements the mapper.CartMapper interface.
func (mem *aceMemory) Plumb(arm interruptARM) {
	mem.arm = arm
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *aceMemory) MapAddress(addr uint32, write bool, executing bool) (*[]byte, uint32) {
	switch addr {
	case DATA_MODER:
		return &mem.gpio, mem.gpioOrigin
	case ADDR_IDR:
		if !write {
			mem.armInterruptCt++
			mem.arm.Interrupt()
		}
		return &mem.gpio, mem.gpioOrigin
	case DATA_IDR:
		return &mem.gpio, mem.gpioOrigin
	case DATA_ODR:
		return &mem.gpio, mem.gpioOrigin
	}

	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, mem.sramOrigin
	}
	if addr >= mem.flashOrigin && addr <= mem.flashMemtop {
		return &mem.flash, mem.flashOrigin
	}
	if addr >= mem.ccmOrigin && addr <= mem.ccmMemtop {
		return &mem.ccm, mem.ccmOrigin
	}

	return nil, addr
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *aceMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}

// IsExecutable implements the arm.SharedMemory interface.
func (mem *aceMemory) IsExecutable(addr uint32) bool {
	return true
}

// returns a list of memory areas in the cartridge's static memory
func (a *aceMemory) Segments() []mapper.CartStaticSegment {
	return []mapper.CartStaticSegment{
		{
			Name:   "Flash",
			Origin: a.flashOrigin,
			Memtop: a.flashMemtop,
		},
		{
			Name:   "SRAM",
			Origin: a.sramOrigin,
			Memtop: a.sramMemtop,
		},
		{
			Name:   "CCM",
			Origin: a.ccmOrigin,
			Memtop: a.ccmMemtop,
		},
	}
}

// returns a copy of the data in the named segment. the segment name should
// be taken from the Name field of one of the CartStaticSegment instances
// returned by the Segments() function
func (a *aceMemory) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case "Flash":
		return a.flash, true
	case "SRAM":
		return a.sram, true
	case "CCM":
		return a.ccm, true
	}
	return []uint8{}, false
}

// read 8, 16 or 32 bit values from the address. the address should be in
// the range given in one of the CartStaticSegment returned by the
// Segments() function.
func (a *aceMemory) Read8bit(addr uint32) (uint8, bool) {
	mem, origin := a.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

func (a *aceMemory) Read16bit(addr uint32) (uint16, bool) {
	mem, origin := a.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

func (a *aceMemory) Read32bit(addr uint32) (uint32, bool) {
	mem, origin := a.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// Read8bit implements the mapper.CartStatic interface
func (a *aceMemory) Write8bit(addr uint32, data uint8) bool {
	mem, origin := a.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return false
	}
	(*mem)[addr] = data
	return true
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *Ace) GetStatic() mapper.CartStatic {
	return cart.mem.Snapshot()
}

// ReferenceStatic implements the mapper.CartStaticBus interface.
func (cart *Ace) ReferenceStatic() mapper.CartStatic {
	return cart.mem
}

// StaticWrite implements the mapper.CartStaticBus interface.
func (cart *Ace) PutStatic(segment string, idx int, data uint8) bool {
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
