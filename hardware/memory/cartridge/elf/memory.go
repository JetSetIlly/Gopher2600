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
	"fmt"

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

type elfSection struct {
	name   string
	data   []byte
	origin uint32
	memtop uint32
}

type elfMemory struct {
	model   memorymodel.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	// input/output pins
	gpio gpio

	// the different sections of the loaded ELF binary
	sections map[string]elfSection

	// RAM memory for the ARM
	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// strongARM support
	strongArmProgram   []byte
	strongArmOrigin    uint32
	strongArmMemtop    uint32
	strongArmFunctions map[uint32]strongArmFunction

	// will be set to true if the vcsWrite3() function is used
	usesBusStuffing bool

	// whether bus stuff is active at the current moment and the data to stuff
	busStuff     bool
	busStuffData uint8

	// strongarm data and a small interface to the ARM
	arm       yieldARM
	strongarm strongArmState

	// args is a special memory area that is used for the arguments passed to
	// the main function on startup
	args []byte
}

func newElfMemory(f *elf.File) (*elfMemory, error) {
	mem := &elfMemory{
		gpio:     newGPIO(),
		sections: make(map[string]elfSection),
		args:     make([]byte, argMemtop-argOrigin),
	}

	// always using PlusCart model for now
	mem.model = memorymodel.NewMap(memorymodel.PlusCart)

	// load sections
	origin := mem.model.FlashOrigin
	for _, sec := range f.Sections {
		// ignore relocation sections for now
		switch sec.Type {
		case elf.SHT_REL:
			continue

		case elf.SHT_INIT_ARRAY:
			fallthrough
		case elf.SHT_NOBITS:
			fallthrough
		case elf.SHT_PROGBITS:
			section := elfSection{
				name: sec.Name,
			}

			var err error

			section.data, err = sec.Data()
			if err != nil {
				return nil, curated.Errorf("ELF: %v", err)
			}

			// ignore empty sections
			if len(section.data) == 0 {
				continue
			}

			section.origin = origin
			section.memtop = section.origin + uint32(len(section.data))
			origin = (section.memtop + 4) & 0xfffffffc

			// extend data section so that it is continuous with the following section
			gap := origin - section.memtop - 1
			if gap > 0 {
				extend := make([]byte, gap)
				section.data = append(section.data, extend...)
				section.memtop += gap
			}

			mem.sections[section.name] = section

			logger.Logf("ELF", "%s: %08x to %08x (%d)", section.name, section.origin, section.memtop, len(section.data))

		default:
			logger.Logf("ELF", "ignoring section %s (%s)", sec.Name, sec.Type)
		}
	}

	// strongArm functions are added during relocation
	mem.strongArmFunctions = make(map[uint32]strongArmFunction)
	mem.strongArmOrigin = origin
	mem.strongArmMemtop = origin

	// symbols used during relocation
	symbols, err := f.Symbols()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}

	// relocate all sections
	for _, relsec := range f.Sections {
		// ignore non-relocation sections for now
		if relsec.Type != elf.SHT_REL {
			continue
		}

		// section being relocated. we should really be using the link field of the elf.Section for this
		var secBeingRelocated elfSection
		if s, ok := mem.sections[relsec.Name[4:]]; !ok {
			return nil, curated.Errorf("ELF: could not find section corresponding to %s", relsec.Name)
		} else {
			secBeingRelocated = s
		}

		// reloation data. we walk this manually and extract the relocation
		// entry "by hand". there is no explicit entry type in the Go library
		// (for some reason)
		relsecData, err := relsec.Data()
		if err != nil {
			return nil, curated.Errorf("ELF: %v", err)
		}

		// every relocation entry
		for i := 0; i < len(relsecData); i += 8 {
			var v uint32

			// the relocation entry fields
			offset := uint32(relsecData[i]) | uint32(relsecData[i+1])<<8 | uint32(relsecData[i+2])<<16 | uint32(relsecData[i+3])<<24
			info := uint32(relsecData[i+4]) | uint32(relsecData[i+5])<<8 | uint32(relsecData[i+6])<<16 | uint32(relsecData[i+7])<<24

			// symbol is encoded in the info value
			symbolIdx := info >> 8
			sym := symbols[symbolIdx-1]

			// reltype is encoded in the info value
			relType := info & 0xff

			switch elf.R_ARM(relType) {
			case elf.R_ARM_TARGET1:
				fallthrough
			case elf.R_ARM_ABS32:
				switch sym.Name {
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
					mem.usesBusStuffing = true
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

				// C library functions that are often not linked but required
				case "memset":
					v = mem.relocateStrongArmFunction(mem.memset)
				case "memcpy":
					v = mem.relocateStrongArmFunction(mem.memcpy)
				case "__aeabi_idiv":
					// sometimes linked when building for ARMv6-M target
					v = mem.relocateStrongArmFunction(mem.__aeabi_idiv)

				// strongARM tables
				case "ReverseByte":
					v = mem.relocateStrongArmTable(reverseByteTable)
				case "ColorLookup":
					v = mem.relocateStrongArmTable(ntscColorTable)

				default:
					if sym.Section == elf.SHN_UNDEF {
						return nil, curated.Errorf("ELF: %s is undefined", sym.Name)
					}

					n := f.Sections[sym.Section].Name
					if p, ok := mem.sections[n]; !ok {
						return nil, curated.Errorf("ELF: can not find section (%s) while relocation %s", p.name, sym.Name)
					} else {
						v = p.origin
					}
					v += uint32(sym.Value)
				}

				// add placeholder value to relocation address
				addend := uint32(secBeingRelocated.data[offset])
				addend |= uint32(secBeingRelocated.data[offset+1]) << 8
				addend |= uint32(secBeingRelocated.data[offset+2]) << 16
				addend |= uint32(secBeingRelocated.data[offset+3]) << 24
				v += addend

				// check address is recognised
				mappedData, mappedOffset := mem.MapAddress(v, false)
				if mappedData == nil {
					return nil, curated.Errorf("ELF: illegal relocation address (%08x) for %s", v, sym.Name)
				}

				// peep hole data (for logging output)
				const peepHoleLen = 10
				var mappedDataPeepHole string
				if len(*mappedData) > int(mappedOffset+peepHoleLen) {
					mappedDataPeepHole = fmt.Sprintf("[% 02x...]", (*mappedData)[mappedOffset:mappedOffset+peepHoleLen])
				}

				// commit write
				secBeingRelocated.data[offset] = uint8(v)
				secBeingRelocated.data[offset+1] = uint8(v >> 8)
				secBeingRelocated.data[offset+2] = uint8(v >> 16)
				secBeingRelocated.data[offset+3] = uint8(v >> 24)

				// what we log depends on the info field. for info of type 0x03
				// (lower nibble) then the relocation is of a entire section.
				// in this case the symbol will not have a name so we use the
				// section name instead
				if relsec.Info&0x03 == 0x03 {
					logger.Logf("ELF", "relocate %s (%08x) => %08x %s", f.Sections[sym.Section].Name, secBeingRelocated.origin+offset, v, mappedDataPeepHole)
				} else {
					n := sym.Name
					if n == "" {
						n = "(unnamed)"
					}
					logger.Logf("ELF", "relocate %s (%08x) => %08x %s", n, secBeingRelocated.origin+offset, v, mappedDataPeepHole)
				}

			case elf.R_ARM_THM_PC22:
				// this value is labelled R_ARM_THM_CALL in objdump output
				//
				// "R_ARM_THM_PC22 Bits 0-10 encode the 11 most significant bits of
				// the branch offset, bits 0-10 of the next instruction word the 11
				// least significant bits. The unit is 2-byte Thumb instructions."
				// page 32 of "SWS ESPC 0003 A-08"

				if sym.Section == elf.SHN_UNDEF {
					return nil, curated.Errorf("ELF: %s is undefined", sym.Name)
				}

				n := f.Sections[sym.Section].Name
				if p, ok := mem.sections[n]; !ok {
					return nil, curated.Errorf("ELF: can not find section (%s)", p.name)
				} else {
					v = p.origin
				}
				v += uint32(sym.Value)
				v &= 0xfffffffe
				v -= (secBeingRelocated.origin + offset + 4)

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

				lo := uint16(0xf000 | (s << 10) | imm10)
				hi := uint16(0xd000 | (j1 << 13) | (j2 << 11) | imm11)
				opcode := uint32(lo) | (uint32(hi) << 16)

				secBeingRelocated.data[offset] = uint8(opcode)
				secBeingRelocated.data[offset+1] = uint8(opcode >> 8)
				secBeingRelocated.data[offset+2] = uint8(opcode >> 16)
				secBeingRelocated.data[offset+3] = uint8(opcode >> 24)

				logger.Logf("ELF", "relocate %s (%08x) => %08x", n, secBeingRelocated.origin+offset, opcode)

			default:
				return nil, curated.Errorf("ELF: unhandled ARM relocation type (%v)", relType)
			}
		}
	}

	// find entry point and use it to set the resetPC value. the Entry field in
	// the elf.File structure is no good for our purposes
	for _, s := range symbols {
		if s.Name == "main" || s.Name == "elf_main" {
			mem.resetPC = mem.sections[".text"].origin + uint32(s.Value)
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

	for _, s := range mem.sections {
		if addr >= s.origin && addr <= s.memtop {
			return &s.data, addr - s.origin
		}
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

	// check argument memory last
	if addr >= argOrigin && addr <= argMemtop {
		return &mem.args, addr - argOrigin
	}

	return nil, addr
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *elfMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}
