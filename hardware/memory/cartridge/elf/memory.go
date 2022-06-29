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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/logger"
)

type yieldARM interface {
	Yield()
	Registers() [arm.NumRegisters]uint32
}

type elfMemory struct {
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

	gpioLookup       []byte
	gpioLookupOrigin uint32
	gpioLookupMemtop uint32

	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// strongARM support
	strongArmProgram   []byte
	strongArmOrigin    uint32
	strongArmMemtop    uint32
	strongArmFunctions map[uint32]strongArmFunction

	arm       yieldARM
	strongarm strongarm
}

const (
	gpio_mode      = 0x00 // gpioB
	toArm_address  = 0x10 // gpioA
	toArm_data     = 0x10 // gpioB
	fromArm_Opcode = 0x14 // gpioB
	gpio_memtop    = 0x18
)

func newElfMemory(f *elf.File) (*elfMemory, error) {
	mem := &elfMemory{}

	// always using PlusCart model for now
	mem.model = memorymodel.NewMap(memorymodel.PlusCart)

	mem.resetSP = mem.model.SRAMOrigin | 0x0000ffdc
	mem.resetLR = mem.model.FlashOrigin
	const jumpVector = 0x08020000
	mem.resetPC = mem.model.FlashOrigin + jumpVector

	// align reset PC value (maybe this should be done in the ARM package once
	// the value has been received - on second thoughts, no we shouldn't do
	// this because we need to know the true value of resetPC in MapAddress()
	// below)
	mem.resetPC &= 0xfffffffe

	// load elf sections
	textSection := f.Section(".text")
	if textSection == nil {
		return nil, curated.Errorf("ELF: could not fine .text")
	}
	if textSection.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
		return nil, curated.Errorf("ELF: .text section is not relocatable")
	}
	if textSection.Flags&elf.SHF_EXECINSTR != elf.SHF_EXECINSTR {
		return nil, curated.Errorf("ELF: .text section is not executable")
	}

	dataSection := f.Section(".data")
	if dataSection == nil {
		return nil, curated.Errorf("ELF: could not fine .data")
	}
	if dataSection.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
		return nil, curated.Errorf("ELF: .data section is not relocatable")
	}

	reltextSection := f.Section(".rel.text")
	if reltextSection == nil {
		return nil, curated.Errorf("ELF: could not fine .rel.text")
	}
	if reltextSection.Type != elf.SHT_REL {
		return nil, curated.Errorf("ELF: .rel.text is not type SHT_REL")
	}

	// copy vcs program to our emulated memory
	vcsProg, err := dataSection.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	mem.vcsProgram = make([]byte, len(vcsProg))
	copy(mem.vcsProgram, vcsProg)
	mem.vcsOrigin = mem.model.FlashOrigin
	mem.vcsMemtop = mem.vcsOrigin + uint32(len(mem.vcsProgram))

	// copy arm program to our emulated memory
	armProg, err := textSection.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	mem.armProgram = make([]byte, len(armProg))
	copy(mem.armProgram, armProg)
	mem.armOrigin = mem.resetPC
	mem.armMemtop = mem.armOrigin + uint32(len(mem.armProgram))

	// GPIO pins
	mem.gpioA = make([]byte, gpio_memtop)
	mem.gpioAOrigin = 0x40020c00
	mem.gpioAMemtop = mem.gpioAOrigin | gpio_memtop

	mem.gpioB = make([]byte, gpio_memtop)
	mem.gpioBOrigin = 0x40020800
	mem.gpioBMemtop = mem.gpioBOrigin | gpio_memtop

	mem.gpioLookup = make([]byte, gpio_memtop)
	mem.gpioLookupOrigin = 0x40020400
	mem.gpioLookupMemtop = mem.gpioLookupOrigin | gpio_memtop
	offset := toArm_address
	val := mem.gpioAOrigin | toArm_address
	mem.gpioLookup[offset] = uint8(val)
	mem.gpioLookup[offset+1] = uint8(val >> 8)
	mem.gpioLookup[offset+2] = uint8(val >> 16)
	mem.gpioLookup[offset+3] = uint8(val >> 24)
	offset = fromArm_Opcode
	val = mem.gpioBOrigin | fromArm_Opcode
	mem.gpioLookup[offset] = uint8(val)
	mem.gpioLookup[offset+1] = uint8(val >> 8)
	mem.gpioLookup[offset+2] = uint8(val >> 16)
	mem.gpioLookup[offset+3] = uint8(val >> 24)

	// default NOP instruction for opcode
	mem.gpioB[fromArm_Opcode] = 0xea

	// SRAM creation
	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram))

	// strongARM functions are added when the relocation information suggests
	// that it is required
	mem.strongArmFunctions = make(map[uint32]strongArmFunction)
	mem.strongArmOrigin = mem.armMemtop
	mem.strongArmMemtop = mem.strongArmOrigin

	// relocate arm program (for emulated memory)
	relocation, err := reltextSection.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	symbols, err := f.Symbols()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}

	write := func(offset uint32, val uint32) {
		mem.armProgram[offset] = uint8(val)
		mem.armProgram[offset+1] = uint8(val >> 8)
		mem.armProgram[offset+2] = uint8(val >> 16)
		mem.armProgram[offset+3] = uint8(val >> 24)
	}

	for i := 0; i < len(relocation); i += 8 {
		offset := uint32(relocation[i]) | uint32(relocation[i+1])<<8 | uint32(relocation[i+2])<<16 | uint32(relocation[i+3])<<24
		relType := relocation[i+4]
		symbolIdx := uint32(relocation[i+5]) | uint32(relocation[i+6])<<8 | uint32(relocation[i+7])<<16
		symbolIdx--

		switch elf.R_ARM(relType) {
		case elf.R_ARM_ABS32:
			n := symbols[symbolIdx].Name
			switch n {
			case "_binary_4k_rom_bin_start":
				v := mem.vcsOrigin
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "_binary_4k_rom_bin_size":
				v := uint32(len(mem.vcsProgram))
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "_binary_4k_rom_bin_end":
				v := uint32(mem.vcsOrigin + uint32(len(mem.vcsProgram)))
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "ADDR_IDR":
				v := uint32(mem.gpioLookupOrigin | toArm_address)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "DATA_ODR":
				v := uint32(mem.gpioLookupOrigin | fromArm_Opcode)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "DATA_MODER":
				v := uint32(mem.gpioBOrigin | gpio_mode)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)

			// strongArm functions
			case "vcsJsr6":
				write(offset, mem.relocateStrongArmFunction(mem.vcsJsr6))

			default:
				return nil, curated.Errorf("ELF: unrelocated symbol (%s)", n)
			}
		default:
			return nil, curated.Errorf("ELF: unhandled ARM relocation type")
		}
	}

	return mem, nil
}

func (mem *elfMemory) relocateStrongArmFunction(f strongArmFunction) uint32 {
	// address of new function in memory
	addr := mem.strongArmMemtop + 3

	// function ID of this strongArm function (for this ROM)
	mem.strongArmFunctions[addr] = f

	// add null function to end of strongArmProgram array
	mem.strongArmProgram = append(mem.strongArmProgram, strongArmStub...)

	// update memtop of strongArm program
	mem.strongArmMemtop += uint32(len(strongArmStub))

	return addr
}

func (mem *elfMemory) Snapshot() *elfMemory {
	m := *mem
	return &m
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *elfMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr >= mem.gpioAOrigin && addr <= mem.gpioAMemtop {
		if !write && addr == mem.gpioAOrigin|toArm_address {
			mem.arm.Yield()
		}
		return &mem.gpioA, addr - mem.gpioAOrigin
	}
	if addr >= mem.gpioBOrigin && addr <= mem.gpioBMemtop {
		return &mem.gpioB, addr - mem.gpioBOrigin
	}
	if addr >= mem.gpioLookupOrigin && addr <= mem.gpioLookupMemtop {
		return &mem.gpioLookup, addr - mem.gpioLookupOrigin
	}
	if addr >= mem.armOrigin && addr <= mem.armMemtop {
		return &mem.armProgram, addr - mem.resetPC
	}
	if addr >= mem.vcsOrigin && addr <= mem.vcsMemtop {
		return &mem.vcsProgram, addr - mem.vcsOrigin
	}
	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, addr - mem.sramOrigin
	}
	if addr >= mem.strongArmOrigin && addr <= mem.strongArmMemtop {
		if f, ok := mem.strongArmFunctions[addr+1]; ok {
			mem.strongarm.function = f
			mem.strongarm.state = 0
			mem.strongarm.registers = mem.arm.Registers()
			mem.arm.Yield()
		}
		return &mem.strongArmProgram, addr - mem.strongArmOrigin
	}

	return nil, addr
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *elfMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}
