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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
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
	header aceHeader

	model   architecture.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	gpio []byte

	// SRAM is called CCM in STM32 architectures
	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// flash memory is divided into three segments
	//
	// (1) the first chunk is flash memory used during the execution of the
	// program. (2) the VCS segment contains the data intended for execution on
	// and directly used by the 6507. (3) the ARM segment meanwhile contains
	// data for use by the ARM
	//
	// the VCS and ARM segments must be consecutive
	flash       []byte
	flashOrigin uint32
	flashMemtop uint32

	flashVCS       []byte
	flashVCSOrigin uint32
	flashVCSMemtop uint32

	flashARM       []byte
	flashARMOrigin uint32
	flashARMMemtop uint32

	arm interruptARM
}

const (
	DATA_MODER = 0x40020800
	ADDR_IDR   = 0x40020c10
	DATA_IDR   = 0x40020810
	DATA_ODR   = 0x40020814

	DATA_MODER_idx = 0
	ADDR_IDR_idx   = 4
	DATA_IDR_idx   = 8
	DATA_ODR_idx   = 12
	GPIO_SIZE      = 16
)

func (mem *aceMemory) isDataModeOut() bool {
	return mem.gpio[DATA_MODER_idx] == 0x55 && mem.gpio[DATA_MODER_idx+1] == 0x55
}

func (mem *aceMemory) setDataMode(out bool) {
	if out {
		mem.gpio[DATA_MODER_idx] = 0x55
		mem.gpio[DATA_MODER_idx+1] = 0x55
	} else {
		mem.gpio[DATA_MODER_idx] = 0x00
		mem.gpio[DATA_MODER_idx+1] = 0x00
	}
}

func newAceMemory(data []byte) (*aceMemory, error) {
	mem := &aceMemory{}

	// read header
	mem.header.version = string(data[:aceHeaderDriverName])
	logger.Logf("ACE", "header: version name: %s", mem.header.version)

	mem.header.driverName = string(data[aceHeaderDriverName:aceHeaderDriverVersion])
	logger.Logf("ACE", "header: driver name: %s", mem.header.driverName)

	mem.header.driverVersion = (uint32(data[aceHeaderDriverVersion])) |
		(uint32(data[aceHeaderDriverVersion+1]) << 8) |
		(uint32(data[aceHeaderDriverVersion+2]) << 16) |
		(uint32(data[aceHeaderDriverVersion+3]) << 24)
	logger.Logf("ACE", "header: driver version: %08x", mem.header.driverVersion)

	mem.header.romSize = (uint32(data[aceHeaderROMSize])) |
		(uint32(data[aceHeaderROMSize+1]) << 8) |
		(uint32(data[aceHeaderROMSize+2]) << 16) |
		(uint32(data[aceHeaderROMSize+3]) << 24)
	logger.Logf("ACE", "header: romsize: %08x", mem.header.romSize)

	mem.header.checksum = (uint32(data[aceHeaderROMChecksum])) |
		(uint32(data[aceHeaderROMChecksum+1]) << 8) |
		(uint32(data[aceHeaderROMChecksum+2]) << 16) |
		(uint32(data[aceHeaderROMChecksum+3]) << 24)
	logger.Logf("ACE", "header: checksum: %08x", mem.header.checksum)

	mem.header.entry = (uint32(data[aceHeaderEntryPoint])) |
		(uint32(data[aceHeaderEntryPoint+1]) << 8) |
		(uint32(data[aceHeaderEntryPoint+2]) << 16) |
		(uint32(data[aceHeaderEntryPoint+3]) << 24)
	logger.Logf("ACE", "header: entrypoint: %08x", mem.header.entry)

	var entryAdjust uint32

	switch mem.header.version {
	case "ACE-PC00":
		entryAdjust = 0x08020200
		mem.model = architecture.NewMap(architecture.PlusCart)
	case "ACE-UF00":
		entryAdjust = 0x08020200
		mem.header.entry += entryAdjust
		logger.Logf("ACE", "header: entrypoint adjusted to: %08x", mem.header.entry)
		mem.model = architecture.NewMap(architecture.PlusCart)
	case "ACE-2600":
		fallthrough
	default:
		return nil, fmt.Errorf("ACE: version: %s not supported", mem.header.version)
	}

	// SRAM creation
	sramSize := 0x0000fa00 // 64k
	mem.sram = make([]byte, sramSize)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram)) - 1

	// the placement of data in flash memory revolves around the ARM entry point
	mem.resetPC = (mem.model.FlashOrigin + mem.header.entry) & 0xfffffffe
	mem.resetLR = mem.resetPC
	mem.resetSP = mem.sramMemtop - 3

	// copy vcs program
	mem.flashVCS = data[:mem.header.entry-entryAdjust-1]
	mem.flashVCSOrigin = mem.resetPC - uint32(len(mem.flashVCS))
	mem.flashVCSMemtop = mem.flashVCSOrigin + uint32(len(mem.flashVCS)-1)

	// copy arm program
	mem.flashARM = data[mem.header.entry-entryAdjust-1:]
	mem.flashARMOrigin = mem.resetPC
	mem.flashARMMemtop = mem.flashARMOrigin + uint32(len(mem.flashARM)-1)

	// define the Thumb-2 bytecode for a function whose only purpose is to jump
	// back to where it came from bytecode is for instruction "BX LR" with a
	// "true" value in R0
	nullFunction := []byte{
		0x01, 0x20, // MOV R1, #1 (the function returns true)
		0x70, 0x47, // BX LR
	}

	// placing nullFunction at end of ARM program
	nullFunctionAddress := mem.flashARMMemtop + 2
	mem.flashARM = append(mem.flashARM, nullFunction...)
	mem.flashARMMemtop += uint32(len(nullFunction))

	// choose size for the remainder of the flash memory and place at the flash
	// origin value for architecture
	const flashOverhead = 64000
	var flashSize uint32

	if len(data) < 128000-flashOverhead {
		flashSize = 128000
	} else if len(data) < 256000-flashOverhead {
		flashSize = 256000
	} else if len(data) < 512000-flashOverhead {
		flashSize = 512000
	} else {
		flashSize = flashOverhead
	}

	mem.flash = make([]byte, flashSize)
	mem.flashOrigin = mem.model.FlashOrigin
	mem.flashMemtop = mem.flashOrigin + uint32(len(mem.flash)-1)

	// set virtual argument. detailed information in the PlusCart firmware
	// source:
	//
	// atari-2600-pluscart-master/source/STM32firmware/PlusCart/Src/cartridge_emulation_ACE.c

	// cart_rom argument
	mem.flash[0] = uint8(mem.flashVCSOrigin)
	mem.flash[1] = uint8(mem.flashVCSOrigin >> 8)
	mem.flash[2] = uint8(mem.flashVCSOrigin >> 16)
	mem.flash[3] = uint8(mem.flashVCSOrigin >> 24)

	// CCM memory argument
	mem.flash[4] = uint8(mem.model.SRAMOrigin)
	mem.flash[5] = uint8(mem.model.SRAMOrigin >> 8)
	mem.flash[6] = uint8(mem.model.SRAMOrigin >> 16)
	mem.flash[7] = uint8(mem.model.SRAMOrigin >> 24)

	// addresses of func_reboot_into_cartridge() and emulate_firmware_cartridge()
	// for our purposes, the function needs only to jump back to the link address

	// reboot_into_cartridge argument
	mem.flash[8] = uint8(nullFunctionAddress)
	mem.flash[9] = uint8(nullFunctionAddress >> 8)
	mem.flash[10] = uint8(nullFunctionAddress >> 16)
	mem.flash[11] = uint8(nullFunctionAddress >> 24)

	// emulate_firmware_cartridge argument
	mem.flash[12] = uint8(nullFunctionAddress)
	mem.flash[13] = uint8(nullFunctionAddress >> 8)
	mem.flash[14] = uint8(nullFunctionAddress >> 16)
	mem.flash[15] = uint8(nullFunctionAddress >> 24)

	// system clock argument
	// copy(mem.flash[16:20], []byte{0x00, 0x60, 0xfe, 0xcd})
	copy(mem.flash[16:20], []byte{0x80, 0x1d, 0x2c, 0x04})

	// ACE version number
	copy(mem.flash[20:24], []byte{0x00, 0x00, 0x00, 0x02})

	// pluscart revision number
	copy(mem.flash[24:28], []byte{0x00, 0x00, 0x00, 0x03})

	// end of argument indicator
	copy(mem.flash[28:32], []byte{0x00, 0x26, 0xe4, 0xac})

	// GPIO pins
	mem.gpio = make([]byte, GPIO_SIZE)

	// default NOP instruction for opcode
	mem.gpio[DATA_ODR_idx] = 0xea

	return mem, nil
}

func (mem *aceMemory) Snapshot() *aceMemory {
	m := *mem
	m.gpio = make([]byte, len(mem.gpio))
	copy(m.gpio, mem.gpio)
	m.sram = make([]byte, len(mem.sram))
	copy(m.sram, mem.sram)
	m.flash = make([]byte, len(mem.flash))
	copy(m.flash, mem.flash)
	m.flashVCS = make([]byte, len(mem.flashVCS))
	copy(m.flashVCS, mem.flashVCS)
	m.flashARM = make([]byte, len(mem.flashARM))
	copy(m.flashARM, mem.flashARM)
	return &m
}

// Plumb implements the mapper.CartMapper interface.
func (mem *aceMemory) Plumb(arm interruptARM) {
	mem.arm = arm
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *aceMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	switch addr {
	case DATA_MODER:
		return &mem.gpio, DATA_MODER_idx
	case ADDR_IDR:
		if !write {
			mem.arm.Interrupt()
		}
		return &mem.gpio, ADDR_IDR_idx
	case DATA_IDR:
		return &mem.gpio, DATA_IDR_idx
	case DATA_ODR:
		return &mem.gpio, DATA_ODR_idx
	}

	if addr >= mem.flashARMOrigin && addr <= mem.flashARMMemtop {
		return &mem.flashARM, addr - mem.flashARMOrigin
	}
	if addr >= mem.flashVCSOrigin && addr <= mem.flashVCSMemtop {
		return &mem.flashVCS, addr - mem.flashVCSOrigin
	}
	if addr >= mem.flashOrigin && addr <= mem.flashMemtop {
		return &mem.flash, addr - mem.flashOrigin
	}
	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, addr - mem.sramOrigin
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
			Name:   "VCS",
			Origin: a.flashVCSOrigin,
			Memtop: a.flashVCSMemtop,
		},
		{
			Name:   "ARM",
			Origin: a.flashARMOrigin,
			Memtop: a.flashARMMemtop,
		},
		{
			Name:   "CCM",
			Origin: a.sramOrigin,
			Memtop: a.sramMemtop,
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
	case "VCS":
		return a.flashVCS, true
	case "ARM":
		return a.flashARM, true
	case "CCM":
		return a.sram, true
	}
	return []uint8{}, false
}

// read 8, 16 or 32 bit values from the address. the address should be in
// the range given in one of the CartStaticSegment returned by the
// Segments() function.
func (a *aceMemory) Read8bit(addr uint32) (uint8, bool) {
	mem, addr := a.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

func (a *aceMemory) Read16bit(addr uint32) (uint16, bool) {
	mem, addr := a.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

func (a *aceMemory) Read32bit(addr uint32) (uint32, bool) {
	mem, addr := a.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *Ace) GetStatic() mapper.CartStatic {
	return cart.mem.Snapshot()
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
