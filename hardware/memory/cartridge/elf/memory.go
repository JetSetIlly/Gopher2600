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
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

type interruptARM interface {
	Interrupt()
	Registers() [arm.NumRegisters]uint32
	SetRegisters([arm.NumRegisters]uint32)
}

type elfSection struct {
	name       string
	data       []byte
	origin     uint32
	memtop     uint32
	readOnly   bool
	executable bool
}

// Snapshot implements the mapper.CartMapper interface.
func (s *elfSection) Snapshot() *elfSection {
	n := *s
	n.data = make([]byte, len(s.data))
	copy(n.data, s.data)
	return &n
}

type elfMemory struct {
	model   architecture.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	// input/output pins
	gpio *gpio

	// the different sections of the loaded ELF binary
	sections map[string]*elfSection

	// RAM memory for the ARM
	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	// strongARM support
	strongArmProgram   []byte
	strongArmOrigin    uint32
	strongArmMemtop    uint32
	strongArmFunctions map[uint32]strongArmFunction

	// whether the ARM should resume immediately after a yield, without
	// returning control to the VCS
	//
	// the map correlates the ARM address of the function (same as the
	// strongArmFunctions map) to whether or not the above is true.
	strongArmResumeImmediately map[uint32]bool

	// whether the current strong arm function expects the ARM to resume
	// immediately after execution. valid only for use in the
	// elf.runStrongarm() function
	resumeARMimmediately bool

	// will be set to true if the vcsWrite3(), vcsPlp4Ex(), or vcsPla4Ex() function is used
	usesBusStuffing bool

	// whether bus stuff is active at the current moment and the data to stuff
	busStuff     bool
	busStuffData uint8

	// solution to a timing problem with regards to bus stuff. when the
	// busStuff field is true busStuffDelay is set to true until after the next
	// call to BuStuff()
	//
	// to recap: my understanding is that when bus stuff is true the cartridge
	// is actively driving the data bus. this will affect the next read as well
	// as the next write. however, the bus stuffing instruction vcsWrite3()
	// only wants to affect the next write cycle so we delay the stuffing by
	// one cycle
	busStuffDelay bool

	// strongarm data and a small interface to the ARM
	arm       interruptARM
	strongarm strongArmState

	// args is a special memory area that is used for the arguments passed to
	// the main function on startup
	args []byte
}

func newElfMemory(ef *elf.File) (*elfMemory, error) {
	mem := &elfMemory{
		gpio:     newGPIO(),
		sections: make(map[string]*elfSection),
		args:     make([]byte, argMemtop-argOrigin),
	}

	// always using PlusCart model for now
	mem.model = architecture.NewMap(architecture.PlusCart)

	// load sections
	origin := mem.model.FlashOrigin
	for _, sec := range ef.Sections {
		// ignore debug sections
		if strings.Contains(sec.Name, ".debug") {
			continue
		}

		switch sec.Type {
		case elf.SHT_REL:
			// ignore relocation sections for now
			continue

		case elf.SHT_INIT_ARRAY:
			fallthrough
		case elf.SHT_NOBITS:
			fallthrough
		case elf.SHT_PROGBITS:
			section := &elfSection{
				name:       sec.Name,
				readOnly:   sec.Flags&elf.SHF_WRITE != elf.SHF_WRITE,
				executable: sec.Flags&elf.SHF_EXECINSTR == elf.SHF_EXECINSTR,
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

			// extend memtop so that it is continuous with the following section
			gap := origin - section.memtop - 1
			if gap > 0 {
				extend := make([]byte, gap)
				section.data = append(section.data, extend...)
				section.memtop += gap
			}

			mem.sections[section.name] = section

			logger.Logf("ELF", "%s: %08x to %08x (%d)", section.name, section.origin, section.memtop, len(section.data))
			if section.readOnly {
				logger.Logf("ELF", "%s: is readonly", section.name)
			}
			if section.executable {
				logger.Logf("ELF", "%s: is executable", section.name)
			}
		default:
			logger.Logf("ELF", "ignoring section %s (%s)", sec.Name, sec.Type)
		}
	}

	// strongArm functions are added during relocation
	mem.strongArmFunctions = make(map[uint32]strongArmFunction)
	mem.strongArmOrigin = origin
	mem.strongArmMemtop = origin

	// equivalent map to strongArmFunctions which records whether the function
	// yields to the VCS or expects the ARM to resume immediately on function
	// end
	mem.strongArmResumeImmediately = make(map[uint32]bool)

	// symbols used during relocation
	symbols, err := ef.Symbols()
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}

	// relocate all sections
	for _, relsec := range ef.Sections {
		// ignore non-relocation sections for now
		if relsec.Type != elf.SHT_REL {
			continue
		}

		// ignore debug sections
		if strings.Contains(relsec.Name, ".debug") {
			continue
		}

		// section being relocated
		var secBeingRelocated *elfSection
		if s, ok := mem.sections[relsec.Name[4:]]; !ok {
			return nil, curated.Errorf("ELF: could not find section corresponding to %s", relsec.Name)
		} else {
			secBeingRelocated = s
		}

		// I'm not sure how to handle .debug_macro. it seems to be very
		// different to other sections. problems I've seen so far (1) relocated
		// value will be out of range according to the MapAddress check (2) the
		// offset value can go beyond the end of the .debug_macro data slice
		if secBeingRelocated.name == ".debug_macro" {
			logger.Logf("ELF", "not relocating %s", secBeingRelocated.name)
			continue
		} else {
			logger.Logf("ELF", "relocating %s", secBeingRelocated.name)
		}

		// relocation data. we walk over the data and extract the relocation
		// entry manually. there is no explicit entry type in the Go library
		// (for some reason)
		relsecData, err := relsec.Data()
		if err != nil {
			return nil, curated.Errorf("ELF: %v", err)
		}

		// every relocation entry
		for i := 0; i < len(relsecData); i += 8 {
			var v uint32

			// the relocation entry fields
			offset := ef.ByteOrder.Uint32(relsecData[i : i+4])
			info := ef.ByteOrder.Uint32(relsecData[i+4 : i+8])

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
					v = mem.relocateStrongArmFunction(vcsWrite3, false)
					mem.usesBusStuffing = true
				case "vcsPlp4Ex":
					v = mem.relocateStrongArmFunction(vcsPlp4Ex, false)
					mem.usesBusStuffing = true
				case "vcsPla4Ex":
					v = mem.relocateStrongArmFunction(vcsPla4Ex, false)
					mem.usesBusStuffing = true
				case "vcsJmp3":
					v = mem.relocateStrongArmFunction(vcsJmp3, false)
				case "vcsLda2":
					v = mem.relocateStrongArmFunction(vcsLda2, false)
				case "vcsSta3":
					v = mem.relocateStrongArmFunction(vcsSta3, false)
				case "SnoopDataBus":
					v = mem.relocateStrongArmFunction(snoopDataBus, false)
				case "vcsRead4":
					v = mem.relocateStrongArmFunction(vcsRead4, false)
				case "vcsStartOverblank":
					v = mem.relocateStrongArmFunction(vcsStartOverblank, false)
				case "vcsEndOverblank":
					v = mem.relocateStrongArmFunction(vcsEndOverblank, false)
				case "vcsLdaForBusStuff2":
					v = mem.relocateStrongArmFunction(vcsLdaForBusStuff2, false)
				case "vcsLdxForBusStuff2":
					v = mem.relocateStrongArmFunction(vcsLdxForBusStuff2, false)
				case "vcsLdyForBusStuff2":
					v = mem.relocateStrongArmFunction(vcsLdyForBusStuff2, false)
				case "vcsWrite5":
					v = mem.relocateStrongArmFunction(vcsWrite5, false)
				case "vcsLdx2":
					v = mem.relocateStrongArmFunction(vcsLdx2, false)
				case "vcsLdy2":
					v = mem.relocateStrongArmFunction(vcsLdy2, false)
				case "vcsSta4":
					v = mem.relocateStrongArmFunction(vcsSta4, false)
				case "vcsStx3":
					v = mem.relocateStrongArmFunction(vcsStx3, false)
				case "vcsStx4":
					v = mem.relocateStrongArmFunction(vcsStx4, false)
				case "vcsSty3":
					v = mem.relocateStrongArmFunction(vcsSty3, false)
				case "vcsSty4":
					v = mem.relocateStrongArmFunction(vcsSty4, false)
				case "vcsSax3":
					v = mem.relocateStrongArmFunction(vcsSax3, false)
				case "vcsTxs2":
					v = mem.relocateStrongArmFunction(vcsTxs2, false)
				case "vcsJsr6":
					v = mem.relocateStrongArmFunction(vcsJsr6, false)
				case "vcsNop2":
					v = mem.relocateStrongArmFunction(vcsNop2, false)
				case "vcsNop2n":
					v = mem.relocateStrongArmFunction(vcsNop2n, false)
				case "vcsPhp3":
					v = mem.relocateStrongArmFunction(vcsPhp3, false)
				case "vcsPlp4":
					v = mem.relocateStrongArmFunction(vcsPlp4, false)
				case "vcsPla4":
					v = mem.relocateStrongArmFunction(vcsPla4, false)
				case "vcsCopyOverblankToRiotRam":
					v = mem.relocateStrongArmFunction(vcsCopyOverblankToRiotRam, false)

				// C library functions that are often not linked but required
				case "randint":
					v = mem.relocateStrongArmFunction(randint, true)
				case "memset":
					v = mem.relocateStrongArmFunction(memset, true)
				case "memcpy":
					v = mem.relocateStrongArmFunction(memcpy, true)
				case "__aeabi_idiv":
					// sometimes linked when building for ARMv6-M target
					v = mem.relocateStrongArmFunction(__aeabi_idiv, true)

				// strongARM tables
				case "ReverseByte":
					v = mem.relocateStrongArmTable(reverseByteTable)
				case "ColorLookup":
					v = mem.relocateStrongArmTable(ntscColorTable)

				default:
					if sym.Section == elf.SHN_UNDEF {
						return nil, curated.Errorf("ELF: %s is undefined", sym.Name)
					}

					n := ef.Sections[sym.Section].Name
					if p, ok := mem.sections[n]; !ok {
						// ignore unknown sections
						logger.Logf("ELF", "can not find section (%s) while relocating %s", n, sym.Name)
						continue // for loop
					} else {
						v = p.origin
					}
					v += uint32(sym.Value)
				}

				// add placeholder value to relocation address
				addend := ef.ByteOrder.Uint32(secBeingRelocated.data[offset : offset+4])
				v += addend

				// check address is recognised
				if mappedData, _ := mem.MapAddress(v, false); mappedData == nil {
					continue
				}

				// commit write
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:offset+4], v)

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

				n := ef.Sections[sym.Section].Name
				if p, ok := mem.sections[n]; !ok {
					return nil, curated.Errorf("ELF: can not find section (%s)", n)
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

				// commit write
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:offset+4], opcode)

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

func (mem *elfMemory) relocateStrongArmFunction(f strongArmFunction, support bool) uint32 {
	// address of new function in memory
	addr := mem.strongArmMemtop + 3

	// function ID of this strongArm function (for this ROM)
	mem.strongArmFunctions[addr] = f

	// add null function to end of strongArmProgram array
	mem.strongArmProgram = append(mem.strongArmProgram, strongArmStub...)

	// update memtop of strongArm program
	mem.strongArmMemtop += uint32(len(strongArmStub))

	// note whether this function expects the ARM emulation to resume
	// immediately
	mem.strongArmResumeImmediately[addr] = support

	return addr - 2
}

// Snapshot implements the mapper.CartMapper interface.
func (mem *elfMemory) Snapshot() *elfMemory {
	m := *mem

	m.gpio = mem.gpio.Snapshot()

	m.sections = make(map[string]*elfSection)
	for k := range mem.sections {
		m.sections[k] = mem.sections[k].Snapshot()
	}

	m.sram = make([]byte, len(mem.sram))
	copy(m.sram, mem.sram)

	m.strongArmProgram = make([]byte, len(mem.strongArmProgram))
	copy(m.strongArmProgram, mem.strongArmProgram)

	m.strongArmFunctions = make(map[uint32]strongArmFunction)
	for k := range mem.strongArmFunctions {
		m.strongArmFunctions[k] = mem.strongArmFunctions[k]
	}

	// not sure we need to copy args because they shouldn't change after the
	// initial setup of the ARM - the setup will never run again even if the
	// rewind reaches the very beginning of the history
	m.args = make([]byte, len(mem.args))
	copy(m.args, mem.args)

	return &m
}

// Plumb implements the mapper.CartMapper interface.
func (mem *elfMemory) Plumb(arm interruptARM) {
	mem.arm = arm
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *elfMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr >= mem.gpio.AOrigin && addr <= mem.gpio.AMemtop {
		if !write && addr == mem.gpio.AOrigin|toArm_address {
			mem.arm.Interrupt()
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
			if write && s.readOnly {
				return nil, addr
			}
			return &s.data, addr - s.origin
		}
	}

	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, addr - mem.sramOrigin
	}
	if addr >= mem.strongArmOrigin && addr <= mem.strongArmMemtop {
		if f, ok := mem.strongArmFunctions[addr+1]; ok {
			mem.setStrongArmFunction(f)
			mem.arm.Interrupt()

			mem.resumeARMimmediately = mem.strongArmResumeImmediately[addr+1]
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

// IsExecutable implements the arm.SharedMemory interface.
func (mem *elfMemory) IsExecutable(addr uint32) bool {
	// TODO: check executable flag for address
	return true
}

// Segments implements the mapper.CartStatic interface
func (e *elfMemory) Segments() []mapper.CartStaticSegment {
	return []mapper.CartStaticSegment{
		mapper.CartStaticSegment{
			Name:   "SRAM",
			Origin: e.sramOrigin,
			Memtop: e.sramMemtop,
		},
		mapper.CartStaticSegment{
			Name:   "StrongARM Program",
			Origin: e.strongArmOrigin,
			Memtop: e.strongArmMemtop,
		},
	}
}

// Reference implements the mapper.CartStatic interface
func (e *elfMemory) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case "SRAM":
		return e.sram, true
	case "StrongARM Program":
		return e.strongArmProgram, true
	}
	return []uint8{}, false
}

// Read8bit implements the mapper.CartStatic interface
func (e *elfMemory) Read8bit(addr uint32) (uint8, bool) {
	mem, addr := e.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

// Read16bit implements the mapper.CartStatic interface
func (e *elfMemory) Read16bit(addr uint32) (uint16, bool) {
	mem, addr := e.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

// Read32bit implements the mapper.CartStatic interface
func (e *elfMemory) Read32bit(addr uint32) (uint32, bool) {
	mem, addr := e.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
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
