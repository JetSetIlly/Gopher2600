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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

type yieldARM interface {
	Yield()
}

type aceMemory struct {
	model   memorymodel.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	armProgram []byte
	armOrigin  uint32
	armMemtop  uint32

	vcsProgram []byte // including virtual arguments
	vcsOrigin  uint32
	vcsMemtop  uint32

	gpioA       []byte
	gpioAOrigin uint32
	gpioAMemtop uint32

	gpioB       []byte
	gpioBOrigin uint32
	gpioBMemtop uint32

	flash       []byte
	flashOrigin uint32
	flashMemtop uint32

	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	arm yieldARM

	// whether bus stuff is active at the current moment and the data to stuff
	busStuff     bool
	busStuffData uint8
}

const (
	aceHeaderMagic         = 0
	aceHeaderDriverName    = 9
	aceHeaderDriverVersion = 24
	aceHeaderROMSize       = 28
	aceHeaderROMChecksum   = 32
	aceHeaderEntryPoint    = 36
	aceStartOfVCSProgram   = 40
)

const (
	gpio_mode      = 0x00 // gpioB
	toArm_address  = 0x10 // gpioA
	toArm_data     = 0x10 // gpioB
	fromArm_Opcode = 0x14 // gpioB
	gpio_memtop    = 0x18
)

func newAceMemory(version string, data []byte) (*aceMemory, error) {
	mem := &aceMemory{}

	switch version {
	case "ACE-2600":
		return nil, curated.Errorf("ACE: unocart not yet supported")
	case "ACE-PC00":
		mem.model = memorymodel.NewMap(memorymodel.PlusCart)
	default:
		return nil, curated.Errorf("ACE: unrecognised version (%s)", version)
	}

	romSize := (uint32(data[aceHeaderROMSize])) |
		(uint32(data[aceHeaderROMSize+1]) << 8) |
		(uint32(data[aceHeaderROMSize+2]) << 16) |
		(uint32(data[aceHeaderROMSize+3]) << 24)

	// ignoring checksum

	entryPoint := (uint32(data[aceHeaderEntryPoint])) |
		(uint32(data[aceHeaderEntryPoint+1]) << 8) |
		(uint32(data[aceHeaderEntryPoint+2]) << 16) |
		(uint32(data[aceHeaderEntryPoint+3]) << 24)

	mem.resetSP = mem.model.SRAMOrigin | 0x0000ffdc
	mem.resetLR = mem.model.FlashOrigin
	mem.resetPC = mem.model.FlashOrigin + entryPoint

	// align reset PC value. this needs to be done now in order for the
	// dataOffset calculation to be correct
	mem.resetPC &= 0xfffffffe

	// offset into the data array for start of ARM program. not entirely sure
	// of the significance of the jumpVector value or what it refers to
	const jumpVector = 0x08020200
	dataOffset := mem.resetPC - jumpVector - mem.model.FlashOrigin

	// copy arm program
	mem.armProgram = make([]byte, romSize)
	copy(mem.armProgram, data[dataOffset:])
	mem.armOrigin = mem.resetPC
	mem.armMemtop = mem.armOrigin + uint32(len(mem.armProgram)) - 1

	// define the Thumb-2 bytecode for a function whose only purpose is to jump
	// back to where it came from bytecode is for instruction "BX LR" with a
	// "true" value in R0
	nullFunction := []byte{
		0x00,       // for alignment
		0x01, 0x20, // MOV R1, #1 (the function returns true)
		0x70, 0x47, // BX LR
	}

	nullFunctionAddress := mem.resetPC + uint32(len(mem.armProgram)) + 2

	// append function to end of flash
	mem.armProgram = append(mem.armProgram, nullFunction...)
	mem.armMemtop += uint32(len(nullFunction))

	// flash creation
	flashSize := 0x18000 // 96k
	mem.flash = make([]byte, flashSize)
	mem.flashOrigin = mem.model.FlashOrigin
	mem.flashMemtop = mem.flashOrigin + uint32(len(mem.flash)) - 1

	// copy vcs program
	mem.vcsProgram = make([]byte, len(data))
	copy(mem.vcsProgram, data)
	mem.vcsOrigin = mem.flashMemtop + 1
	mem.vcsMemtop = mem.vcsOrigin + uint32(len(mem.vcsProgram)) - 1

	// SRAM creation
	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin+1)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram)) - 1

	// set virtual argument. detailed information in the PlusCart firmware
	// source:
	//
	// atari-2600-pluscart-master/source/STM32firmware/PlusCart/Src/cartridge_emulation_ACE.c

	// cart_rom argument
	mem.flash[0] = uint8(mem.vcsOrigin)
	mem.flash[1] = uint8(mem.vcsOrigin >> 8)
	mem.flash[2] = uint8(mem.vcsOrigin >> 16)
	mem.flash[3] = uint8(mem.vcsOrigin >> 24)

	// SRAM memory argument
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

	// not setting system clock or version arguments

	// system clock argument
	copy(mem.vcsProgram[16:20], []byte{0x00, 0x00, 0x00, 0x01})

	// ACE version number
	copy(mem.vcsProgram[20:24], []byte{0x00, 0x00, 0x00, 0x02})

	// pluscart revision number
	copy(mem.vcsProgram[24:28], []byte{0x00, 0x00, 0x00, 0x03})

	// end of argument indicator
	copy(mem.vcsProgram[28:32], []byte{0x00, 0x26, 0xe4, 0xac})

	// GPIO pins
	mem.gpioA = make([]byte, gpio_memtop)
	mem.gpioAOrigin = 0x40020c00
	mem.gpioAMemtop = mem.gpioAOrigin | gpio_memtop

	mem.gpioB = make([]byte, gpio_memtop)
	mem.gpioBOrigin = 0x40020800
	mem.gpioBMemtop = mem.gpioBOrigin | gpio_memtop

	// default NOP instruction for opcode
	mem.gpioB[fromArm_Opcode] = 0xea

	return mem, nil
}

func (mem *aceMemory) Snapshot() *aceMemory {
	m := *mem
	m.armProgram = make([]byte, len(mem.armProgram))
	copy(m.armProgram, mem.armProgram)
	m.vcsProgram = make([]byte, len(mem.vcsProgram))
	copy(m.vcsProgram, mem.vcsProgram)
	m.gpioA = make([]byte, len(mem.gpioA))
	copy(m.gpioA, mem.gpioA)
	m.gpioB = make([]byte, len(mem.gpioB))
	copy(m.gpioB, mem.gpioB)
	m.flash = make([]byte, len(mem.flash))
	copy(m.flash, mem.flash)
	m.sram = make([]byte, len(mem.sram))
	copy(m.sram, mem.sram)
	return &m
}

// Plumb implements the mapper.CartMapper interface.
func (mem *aceMemory) Plumb(arm yieldARM) {
	mem.arm = arm
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *aceMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr >= mem.gpioAOrigin && addr <= mem.gpioAMemtop {
		if !write && addr == mem.gpioAOrigin|toArm_address {
			mem.arm.Yield()
		}
		return &mem.gpioA, addr - mem.gpioAOrigin
	}
	if addr >= mem.gpioBOrigin && addr <= mem.gpioBMemtop {
		return &mem.gpioB, addr - mem.gpioBOrigin
	}
	if addr >= mem.armOrigin && addr <= mem.armMemtop {
		return &mem.armProgram, addr - mem.armOrigin
	}
	if addr >= mem.vcsOrigin && addr <= mem.vcsMemtop {
		return &mem.vcsProgram, addr - mem.vcsOrigin
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
		mapper.CartStaticSegment{
			Name:   "Flash",
			Origin: a.flashOrigin,
			Memtop: a.flashMemtop,
		},
		mapper.CartStaticSegment{
			Name:   "SRAM",
			Origin: a.sramOrigin,
			Memtop: a.sramMemtop,
		},
		mapper.CartStaticSegment{
			Name:   "ARM Program",
			Origin: a.armOrigin,
			Memtop: a.armMemtop,
		},
		mapper.CartStaticSegment{
			Name:   "VCS Program",
			Origin: a.vcsOrigin,
			Memtop: a.vcsMemtop,
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
	case "ARM Program":
		return a.armProgram, true
	case "VCS Program":
		return a.vcsProgram, true
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
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

func (a *aceMemory) Read32bit(addr uint32) (uint32, bool) {
	mem, addr := a.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// GetStatic implements the bus.CartStaticBus interface.
func (cart *Ace) GetStatic() mapper.CartStatic {
	return cart.mem.Snapshot()
}

// StaticWrite implements the bus.CartStaticBus interface.
func (cart *Ace) PutStatic(segment string, idx uint16, data uint8) bool {
	mem, ok := cart.mem.Reference(segment)
	if !ok {
		return false
	}

	if int(idx) >= len(mem) {
		return false
	}
	mem[idx] = data

	return true
}
