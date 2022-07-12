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
	SetRegisters([arm.NumRegisters]uint32)
}

type elfMemory struct {
	model   memorymodel.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	gpio gpio

	textSection       []byte
	textSectionOrigin uint32
	textSectionMemtop uint32

	dataSection       []byte
	dataSectionOrigin uint32
	dataSectionMemtop uint32

	rodataSectionPresent bool
	rodataSection        []byte
	rodataSectionOrigin  uint32
	rodataSectionMemtop  uint32

	bssSectionPresent bool
	bssSection        []byte
	bssSectionOrigin  uint32
	bssSectionMemtop  uint32

	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// strongARM support
	strongArmProgram   []byte
	strongArmOrigin    uint32
	strongArmMemtop    uint32
	strongArmFunctions map[uint32]strongArmFunction

	arm       yieldARM
	strongarm strongArmState
}

func newElfMemory(f *elf.File) (*elfMemory, error) {
	mem := &elfMemory{
		gpio: newGPIO(),
	}

	// always using PlusCart model for now
	mem.model = memorymodel.NewMap(memorymodel.PlusCart)

	var err error

	// .text section
	section := f.Section(".text")
	if section == nil {
		return nil, curated.Errorf("ELF: could not fine .text")
	}
	if section.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
		return nil, curated.Errorf("ELF: .text section is not relocatable")
	}
	if section.Flags&elf.SHF_EXECINSTR != elf.SHF_EXECINSTR {
		return nil, curated.Errorf("ELF: .text section is not executable")
	}
	mem.textSection, err = section.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	mem.textSectionOrigin = mem.model.FlashOrigin + 0x08000000
	mem.textSectionMemtop = mem.textSectionOrigin + uint32(len(mem.textSection))

	// remaining sections start at flashorigin and are consecutive
	origin := mem.model.FlashOrigin
	memtop := origin

	// .data section
	section = f.Section(".data")
	if section == nil {
		return nil, curated.Errorf("ELF: could not fine .data")
	}
	if section.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
		return nil, curated.Errorf("ELF: .data section is not relocatable")
	}
	mem.dataSection, err = section.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	mem.dataSectionOrigin = origin
	memtop = origin + uint32(len(mem.dataSection))
	mem.dataSectionMemtop = memtop
	origin = (memtop + 4) & 0xfffffffc

	// .rodata section
	section = f.Section(".rodata")
	if section != nil {
		if section.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
			return nil, curated.Errorf("ELF: .data section is not relocatable")
		}
		mem.rodataSectionPresent = true
		mem.rodataSection, err = section.Data()
		if err != nil {
			return nil, curated.Errorf("ELF: %v", err)
		}
		mem.rodataSectionOrigin = origin
		memtop = origin + uint32(len(mem.rodataSection))
		mem.rodataSectionMemtop = memtop
		origin = (memtop + 4) & 0xfffffffc
	}

	// .bss section
	section = f.Section(".bss")
	if section != nil {
		if section.Flags&elf.SHF_ALLOC != elf.SHF_ALLOC {
			return nil, curated.Errorf("ELF: .bss section is not relocatable")
		}
		mem.bssSectionPresent = true
		mem.bssSection, err = section.Data()
		if err != nil {
			return nil, curated.Errorf("ELF: %v", err)
		}
		mem.bssSectionOrigin = origin
		memtop = origin + uint32(len(mem.bssSection))
		mem.bssSectionMemtop = memtop
		origin = (memtop + 4) & 0xfffffffc
	}

	// strongARM functions are added when the relocation information suggests
	// that it is required
	mem.strongArmFunctions = make(map[uint32]strongArmFunction)
	mem.strongArmOrigin = mem.textSectionMemtop
	mem.strongArmMemtop = mem.strongArmOrigin

	// relocate text section
	section = f.Section(".rel.text")
	if section == nil {
		return nil, curated.Errorf("ELF: could not fine .rel.text")
	}
	if section.Type != elf.SHT_REL {
		return nil, curated.Errorf("ELF: .rel.text is not type SHT_REL")
	}

	relocation, err := section.Data()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	symbols, err := f.Symbols()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}

	for i := 0; i < len(relocation); i += 8 {
		var v uint32

		offset := uint32(relocation[i]) | uint32(relocation[i+1])<<8 | uint32(relocation[i+2])<<16 | uint32(relocation[i+3])<<24
		relType := relocation[i+4]

		symbolIdx := uint32(relocation[i+5]) | uint32(relocation[i+6])<<8 | uint32(relocation[i+7])<<16
		s := symbols[symbolIdx-1]

		switch elf.R_ARM(relType) {
		case elf.R_ARM_ABS32:
			switch s.Name {
			// GPIO pins
			case "ADDR_IDR":
				v = uint32(mem.gpio.lookupOrigin | toArm_address)
			case "DATA_ODR":
				v = uint32(mem.gpio.lookupOrigin | fromArm_Opcode)
			case "DATA_MODER":
				v = uint32(mem.gpio.BOrigin | gpio_mode)

			// strongARM functions
			case "vcsWrite3":
				v = mem.relocateStrongArmFunction(mem.vcsWrite3)
			case "vcsJmp3":
				v = mem.relocateStrongArmFunction(mem.vcsJmp3)
			case "vcsLda2":
				v = mem.relocateStrongArmFunction(mem.vcsLda2)
			case "vcsSta3":
				v = mem.relocateStrongArmFunction(mem.vcsSta3)
			case "SnoopDataBus":
				v = mem.relocateStrongArmFunction(mem.snoopDataBus)
			case "vcsRead4":
				v = mem.relocateStrongArmFunction(mem.vcsRead4)
			case "vcsStartOverblank":
				v = mem.relocateStrongArmFunction(mem.vcsStartOverblank)
			case "vcsEndOverblank":
				v = mem.relocateStrongArmFunction(mem.vcsEndOverblank)
			case "vcsLdaForBusStuff2":
				v = mem.relocateStrongArmFunction(mem.vcsLdaForBusStuff2)
			case "vcsLdxForBusStuff2":
				v = mem.relocateStrongArmFunction(mem.vcsLdxForBusStuff2)
			case "vcsLdyForBusStuff2":
				v = mem.relocateStrongArmFunction(mem.vcsLdyForBusStuff2)
			case "vcsWrite5":
				v = mem.relocateStrongArmFunction(mem.vcsWrite5)
			case "vcsLdx2":
				v = mem.relocateStrongArmFunction(mem.vcsLdx2)
			case "vcsLdy2":
				v = mem.relocateStrongArmFunction(mem.vcsLdy2)
			case "vcsSta4":
				v = mem.relocateStrongArmFunction(mem.vcsSta4)
			case "vcsStx3":
				v = mem.relocateStrongArmFunction(mem.vcsStx3)
			case "vcsStx4":
				v = mem.relocateStrongArmFunction(mem.vcsStx4)
			case "vcsSty3":
				v = mem.relocateStrongArmFunction(mem.vcsSty3)
			case "vcsSty4":
				v = mem.relocateStrongArmFunction(mem.vcsSty4)
			case "vcsTxs2":
				v = mem.relocateStrongArmFunction(mem.vcsTxs2)
			case "vcsJsr6":
				v = mem.relocateStrongArmFunction(mem.vcsJsr6)
			case "vcsNop2":
				v = mem.relocateStrongArmFunction(mem.vcsNop2)
			case "vcsNop2n":
				v = mem.relocateStrongArmFunction(mem.vcsNop2n)
			case "vcsCopyOverblankToRiotRam":
				v = mem.relocateStrongArmFunction(mem.vcsCopyOverblankToRiotRam)
			case "memset":
				v = mem.relocateStrongArmFunction(mem.memset)
			case "memcpy":
				v = mem.relocateStrongArmFunction(mem.memcpy)

			// strongARM tables
			case "ReverseByte":
				v = mem.relocateStrongArmTable(reverseByteTable)
			case "ColorLookup":
				v = mem.relocateStrongArmTable(ntscColorTable)

			default:
				switch f.Sections[s.Section].Name {
				case ".text":
					v = mem.textSectionOrigin
				case ".data":
					v = mem.dataSectionOrigin
				case ".rodata":
					v = mem.rodataSectionOrigin
				case ".bss":
					v = mem.bssSectionOrigin
				default:
					return nil, curated.Errorf("ELF: undefined symbol (%s)", s.Name)
				}

				v += uint32(s.Value)
			}

			// add placeholder value to relocation address
			w := uint32(mem.textSection[offset])
			w |= uint32(mem.textSection[offset+1]) << 8
			w |= uint32(mem.textSection[offset+2]) << 16
			w |= uint32(mem.textSection[offset+3]) << 24
			v += w

			// commit write
			mem.textSection[offset] = uint8(v)
			mem.textSection[offset+1] = uint8(v >> 8)
			mem.textSection[offset+2] = uint8(v >> 16)
			mem.textSection[offset+3] = uint8(v >> 24)

			// what we log depends on the info field. for info of type 0x03
			// (lower nibble) then the relocation is of a entire section.
			// in this case the symbol will not have a name so we use the
			// section name instead
			if s.Info&0x03 == 0x03 {
				logger.Logf("ELF", "relocate %s (%08x) => %08x", f.Sections[s.Section].Name, mem.textSectionOrigin+offset, v)
			} else {
				logger.Logf("ELF", "relocate %s (%08x) => %08x", s.Name, mem.textSectionOrigin+offset, v)
			}

		case elf.R_ARM_THM_PC22:
			// this value is labelled R_ARM_THM_CALL in objdump output
			//
			// "R_ARM_THM_PC22 Bits 0-10 encode the 11 most significant bits of
			// the branch offset, bits 0-10 of the next instruction word the 11
			// least significant bits. The unit is 2-byte Thumb instructions."
			// page 32 of "SWS ESPC 0003 A-08"

			switch f.Sections[s.Section].Name {
			case ".text":
				v = mem.textSectionOrigin
			case ".data":
				v = mem.dataSectionOrigin
			case ".rodata":
				v = mem.rodataSectionOrigin
			case ".bss":
				v = mem.bssSectionOrigin
			default:
				return nil, curated.Errorf("ELF: undefined symbol (%s)", s.Name)
			}

			v += uint32(s.Value)
			v &= 0xfffffffe
			v -= (mem.textSectionOrigin + offset + 4)

			imm11 := (v >> 1) & 0x7ff
			imm10 := (v >> 12) & 0x3ff
			t1 := (v >> 22) & 0x01
			t2 := (v >> 23) & 0x01
			s := (v >> 24) & 0x01
			j1 := uint32(0)
			j2 := uint32(0)
			if t1 == 0x01 {
				j1 = s ^ 0x00
			} else {
				j1 = s ^ 0x01
			}
			if t2 == 0x01 {
				j2 = s ^ 0x00
			} else {
				j2 = s ^ 0x01
			}

			op1 := uint16(0xf000 | (s << 10) | imm10)
			op2 := uint16(0xd000 | (j1 << 13) | (j2 << 11) | imm11)

			mem.textSection[offset] = uint8(op1)
			mem.textSection[offset+1] = uint8(op1 >> 8)
			mem.textSection[offset+2] = uint8(op2)
			mem.textSection[offset+3] = uint8(op2 >> 8)

		default:
			return nil, curated.Errorf("ELF: unhandled ARM relocation type (%v)", relType)
		}
	}

	// find entry point and use it to set the resetPC value. the Entry field in
	// the elf.File structure is no good for our purposes
	for _, s := range symbols {
		if s.Name == "main" || s.Name == "elf_main" {
			mem.resetPC = mem.textSectionOrigin + uint32(s.Value)
			break // for loop
		}
	}

	// make sure resetPC value is aligned correctly
	mem.resetPC &= 0xfffffffe

	// intialise stack pointer and link register. these values have no
	// reasoning behind them but they work in most cases
	//
	// the link register should really link to a program that will indicate the
	// program has ended. if we were emulating the real Uno/PlusCart firmware,
	// the link register would point to the resume address in the firmware
	mem.resetSP = mem.model.SRAMOrigin | 0x0000ffdc
	mem.resetLR = mem.model.FlashOrigin

	// SRAM creation
	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram))

	return mem, nil
}

func (mem *elfMemory) relocateStrongArmTable(table strongarmTable) uint32 {
	// address of table in memory
	addr := mem.strongArmMemtop

	// add null function to end of strongArmProgram array
	mem.strongArmProgram = append(mem.strongArmProgram, table...)

	// update memtop of strongArm program
	mem.strongArmMemtop += uint32(len(table))

	return addr
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
	if addr >= mem.gpio.AOrigin && addr <= mem.gpio.AMemtop {
		if !write && addr == mem.gpio.AOrigin|toArm_address {
			mem.arm.Yield()
		}
		return &mem.gpio.A, addr - mem.gpio.AOrigin
	}
	if addr >= mem.gpio.BOrigin && addr <= mem.gpio.BMemtop {
		return &mem.gpio.B, addr - mem.gpio.BOrigin
	}
	if addr >= mem.gpio.lookupOrigin && addr <= mem.gpio.lookupMemtop {
		return &mem.gpio.lookup, addr - mem.gpio.lookupOrigin
	}
	if addr >= mem.textSectionOrigin && addr <= mem.textSectionMemtop {
		return &mem.textSection, addr - mem.textSectionOrigin
	}
	if addr >= mem.dataSectionOrigin && addr <= mem.dataSectionMemtop {
		return &mem.dataSection, addr - mem.dataSectionOrigin
	}
	if mem.rodataSectionPresent && addr >= mem.rodataSectionOrigin && addr <= mem.rodataSectionMemtop {
		return &mem.rodataSection, addr - mem.rodataSectionOrigin
	}
	if mem.bssSectionPresent && addr >= mem.bssSectionOrigin && addr <= mem.bssSectionMemtop {
		return &mem.bssSection, addr - mem.bssSectionOrigin
	}
	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, addr - mem.sramOrigin
	}
	if addr >= mem.strongArmOrigin && addr <= mem.strongArmMemtop {
		if f, ok := mem.strongArmFunctions[addr+1]; ok {
			mem.setStrongArmFunction(f)
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
