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
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/faults"
	"github.com/jetsetilly/gopher2600/crunched"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

type elfMemoryARM interface {
	Interrupt()
	MemoryFault(event string, fault faults.Category)
	CoreRegisters() [arm.NumCoreRegisters]uint32
	RegisterSet(int, uint32) bool
}

type elfSection struct {
	name      string
	flags     elf.SectionFlag
	typ       elf.SectionType
	debugging bool

	data   []byte
	origin uint32
	memtop uint32

	// trailing bytes are placed after each section in memory to ensure
	// alignment and also to ensure that executable memory can be indexed by the
	// ARM emulation without worrying about reading past the end of the array.
	// this can happen when trying to execute the last instruction in the
	// program
	trailingBytes uint32
}

func (sec elfSection) readOnly() bool {
	return sec.flags&elf.SHF_WRITE != elf.SHF_WRITE
}

func (sec elfSection) executable() bool {
	return sec.flags&elf.SHF_EXECINSTR == elf.SHF_EXECINSTR
}

func (sec elfSection) inMemory() bool {
	return (sec.typ == elf.SHT_INIT_ARRAY ||
		sec.typ == elf.SHT_NOBITS ||
		sec.typ == elf.SHT_PROGBITS) &&
		sec.debugging == false
}

func (s *elfSection) String() string {
	return fmt.Sprintf("%s %d %08x %08x", s.name, len(s.data), s.origin, s.memtop)
}

func (s *elfSection) isEmpty() bool {
	return s.origin == s.memtop && s.origin == 0
}

// Snapshot implements the mapper.CartMapper interface.
func (s *elfSection) Snapshot() *elfSection {
	n := *s
	n.data = make([]byte, len(s.data))
	copy(n.data, s.data)
	return &n
}

const pxeSection = ".bbpxe"

type elfMemory struct {
	env *environment.Environment

	model   architecture.Map
	resetSP uint32
	resetLR uint32
	resetPC uint32

	// the order in which data is held in the elf file and in memory
	byteOrder binary.ByteOrder

	// input/output pins
	gpio *gpio

	// RAM memory for the ARM
	sram       crunched.Data
	sramOrigin uint32
	sramMemtop uint32

	// the different sectionsByName of the loaded ELF binary
	//
	// note that there is no single block of flash memory. instead, flash memory
	// is built of individual data arrays in each elfSections
	sections       []*elfSection
	sectionNames   []string
	sectionsByName map[string]*elfSection

	// .bbpxe section if available
	pxe pxe

	symbols []elf.Symbol

	// this field will be true if some symbols could not be completely resolved
	// during loading
	unresolvedSymbols bool

	// strongARM support. like the elf sections, the strongARM program is placed
	// in flash memory
	strongArmProgram []byte
	strongArmOrigin  uint32
	strongArmMemtop  uint32

	// for performance reasons this is a sparse array and not a map. this means
	// that when indexing the address should be adjusted by strongArmOrigin
	strongArmFunctions       []*strongArmFunctionSpec
	strongArmFunctionsByName map[string]*strongArmFunctionSpec

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
	arm       elfMemoryARM
	strongarm strongArmState

	// args is a special memory area that is used for the arguments passed to
	// the main function on startup
	args []byte

	// parallelARM is true whenever the address bus is not a cartridge address (ie.
	// a TIA or RIOT address). this means that the arm is running unhindered
	// and will not have yielded for that colour clock
	parallelARM bool

	// most recent yield from the coprocessor
	yield coprocessor.CoProcYield

	// byte stream support
	stream stream

	// last mapping infomation. the address and whether the memory space
	// contains executable instructions. used by IsExecutable() to return
	// additional information about the address
	lastMappedAddress    uint32
	lastMappedExecutable bool

	// a call to MapAddress() from the ARM may sometimes be to find the address
	// of a strongarm function. the act of mapping will cause the strongarm
	// injection process to begin, the assumption being that the ARM will go on
	// to execute the function.
	//
	// however, in the instance of Plumbing() a previous state, the ARM will
	// call MapAddress() (with the current PC register) to make sure it has the
	// pointer to a valid/current area of (real) memory. if the PC register is
	// pointing to a strongarm function then this means erroneous bytes will be
	// injected into the strongarm stream. this will also happen if streaming is
	// disabled but it seems to be less of a problem in that instance
	//
	// the inhibitStrongAccess boolean controls how the MapAddress() function
	// will react to the accessing of strongarm addresses
	inhibitStrongarmAccess bool
}

func newElfMemory(env *environment.Environment) *elfMemory {
	mem := &elfMemory{
		env:                      env,
		gpio:                     newGPIO(),
		sectionsByName:           make(map[string]*elfSection),
		strongArmFunctionsByName: make(map[string]*strongArmFunctionSpec),
		args:                     make([]byte, argMemtop-argOrigin),
		stream: stream{
			env: env,
		},
	}

	// always using PlusCart model for now
	mem.model = architecture.NewMap(architecture.PlusCart)

	// SRAM creation
	const sramSize = 0x10000 // 64kb of SRAM
	mem.sram = crunched.NewQuick(sramSize)
	mem.sramOrigin = mem.model.Regions["SRAM"].Origin
	mem.sramMemtop = mem.sramOrigin + sramSize

	// randomise sram data
	if mem.env.Prefs.RandomState.Get().(bool) {
		data := mem.sram.Data()
		for i := range *data {
			(*data)[i] = uint8(mem.env.Random.Intn(0xff))
		}
	}

	return mem
}

// Snapshot implements the mapper.CartMapper interface.
func (mem *elfMemory) Snapshot() *elfMemory {
	m := *mem

	m.gpio = mem.gpio.Snapshot()

	// snapshot sections. the Snapshot() function of the elfSection type decides
	// how best to deal with the request
	m.sections = make([]*elfSection, len(mem.sections))
	m.sectionsByName = make(map[string]*elfSection)
	for i := range mem.sections {
		if mem.sections[i].readOnly() {
			m.sections[i] = mem.sections[i]
		} else {
			m.sections[i] = mem.sections[i].Snapshot()
		}
		m.sectionsByName[mem.sections[i].name] = m.sections[i]
	}

	// sram is likely to have changed
	m.sram = mem.sram.Snapshot()

	// strongarm program is read-only by defintion

	// strongarm functions is a map but the data pointed to by the map to is
	// read-only

	// not sure we need to copy args because they shouldn't change after the
	// initial setup of the ARM - the setup will never run again even if the
	// rewind reaches the very beginning of the history

	return &m
}

// Plumb implements the mapper.CartMapper interface.
func (mem *elfMemory) Plumb(env *environment.Environment, arm elfMemoryARM) {
	mem.env = env
	mem.arm = arm
}

func (mem *elfMemory) decode(ef *elf.File) error {
	// note byte order
	mem.byteOrder = ef.ByteOrder

	// load sections
	origin := mem.model.Regions["CCM"].Origin
	for _, sec := range ef.Sections {
		section := &elfSection{
			name:      sec.Name,
			flags:     sec.Flags,
			typ:       sec.Type,
			debugging: strings.Contains(sec.Name, ".debug"),
		}

		var err error

		// starting with go1.20 reading from a NOBITS section does not return section data.
		// we must now do that ourselves
		if sec.SectionHeader.Type == elf.SHT_NOBITS {
			section.data = make([]uint8, sec.FileSize)
		} else {
			section.data, err = sec.Data()
			if err != nil {
				return err
			}
		}

		// we know about and record data for all sections but we don't load all of them into the corprocessor's memory
		if section.inMemory() {
			section.origin = origin
			section.memtop = section.origin + uint32(len(section.data))

			// prepare origin of next section and use that to  extend memtop so
			// that it is continuous with the following section
			origin = (section.memtop + 4) & 0xfffffffc
			section.trailingBytes = origin - section.memtop
			if section.trailingBytes > 0 {
				extend := make([]byte, section.trailingBytes)
				section.data = append(section.data, extend...)
				section.memtop += section.trailingBytes - 1
			}

			logger.Logf(mem.env, "ELF", "%s: %08x to %08x (%d) [%d trailing bytes]",
				section.name, section.origin, section.memtop, len(section.data), section.trailingBytes)
			if section.readOnly() {
				logger.Logf(mem.env, "ELF", "%s: is readonly", section.name)
			}
			if section.executable() {
				logger.Logf(mem.env, "ELF", "%s: is executable", section.name)
			}
		}

		// don't add duplicate sections
		//
		// I'm not sure why we would ever have a duplicate section so I'm not
		// sure what affect this will have in the future
		if _, ok := mem.sectionsByName[section.name]; !ok {
			mem.sections = append(mem.sections, section)
			mem.sectionNames = append(mem.sectionNames, section.name)
			mem.sectionsByName[section.name] = section
		}

		// note the existence of PXE
		if section.name == pxeSection {
			mem.pxe.enabled = true
			logger.Logf(mem.env, "ELF", "PXE section found")
		}
	}

	// sort section names
	sort.Strings(mem.sectionNames)

	// strongarm functions are added during relocation
	mem.strongArmOrigin = origin
	mem.strongArmMemtop = mem.strongArmOrigin

	var err error

	// symbols used during relocation
	mem.symbols, err = ef.Symbols()
	if err != nil {
		return err // the error value already has the 'ELF:' prefix
	}

	// relocate all sections
	for _, rel := range ef.Sections {
		// ignore non-relocation sections for now
		if rel.Type != elf.SHT_REL {
			continue
		}

		// section being relocated
		secBeingRelocated, ok := mem.sectionsByName[rel.Name[4:]]
		if !ok {
			return fmt.Errorf("could not find section corresponding to %s", rel.Name)
		}

		// I'm not sure how to handle .debug_macro. it seems to be very
		// different to other sections. problems I've seen so far (1) relocated
		// value will be out of range according to the MapAddress check (2) the
		// offset value can go beyond the end of the .debug_macro data slice
		if secBeingRelocated.name == ".debug_macro" {
			logger.Logf(mem.env, "ELF", "not relocating %s", secBeingRelocated.name)
			continue
		} else {
			logger.Logf(mem.env, "ELF", "relocating %s", secBeingRelocated.name)
		}

		// relocation data. we walk over the data and extract the relocation
		// entry manually. there is no explicit entry type in the Go library
		// (for some reason)
		relData, err := rel.Data()
		if err != nil {
			return err
		}

		// every relocation entry
		for i := 0; i < len(relData); i += 8 {
			// the relocation entry fields
			offset := ef.ByteOrder.Uint32(relData[i:])
			info := ef.ByteOrder.Uint32(relData[i+4:])

			// symbol is encoded in the info value
			symbolIdx := info >> 8
			sym := mem.symbols[symbolIdx-1]

			// reltype is encoded in the info value
			relType := info & 0xff

			switch elf.R_ARM(relType) {
			case elf.R_ARM_TARGET1, elf.R_ARM_ABS32:
				ok, tgt, err := getStrongArmDefinition(mem, sym.Name)
				if err != nil {
					return fmt.Errorf("%s: %w", sym.Name, err)
				}
				if !ok {
					switch sym.Name {
					// GPIO pins
					case "ADDR_IDR":
						tgt = mem.gpio.lookupOrigin | ADDR_IDR
					case "DATA_ODR":
						tgt = mem.gpio.lookupOrigin | DATA_ODR
					case "DATA_MODER":
						tgt = mem.gpio.lookupOrigin | DATA_MODER
					case "DATA_IDR":
						tgt = mem.gpio.lookupOrigin | DATA_IDR

					default:
						if sym.Section == elf.SHN_UNDEF {
							// for R_ARM_ABS32 type symbols we create a stub function and use it to
							// generate a memory fault when it's accessed
							logger.Logf(mem.env, "ELF", "using stub for %s (will cause memory fault when called)", sym.Name)
							tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
								function: func(mem *elfMemory) {
									mem.arm.MemoryFault(sym.Name, faults.UndefinedSymbol)
								},
								support: true,
							})

							// flag presence of unresolved symbols
							mem.unresolvedSymbols = true
						} else {
							n := ef.Sections[sym.Section].Name
							if sec, ok := mem.sectionsByName[n]; !ok {
								return fmt.Errorf("can not find section (%s) while relocating %s", n, sym.Name)
							} else {
								tgt = sec.origin
								tgt += uint32(sym.Value)
							}
						}
					}
				}

				if err != nil {
					return err
				}

				// add placeholder value to relocation address
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				addend := ef.ByteOrder.Uint32(secBeingRelocated.data[offset:])
				tgt += addend

				// check address is recognised
				if mappedData, _ := mem.mapAddress(tgt, false); mappedData == nil {
					continue
				}

				// commit write
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:], tgt)
				if d, err := ef.Section(secBeingRelocated.name).Data(); err == nil {
					if offset >= uint32(len(d)) {
						return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
					}
					ef.ByteOrder.PutUint32(d[offset:], tgt)
				}

				// log relocation address. note that in the case of strongarm
				// functions, because of how BLX works, the target address
				// printed below is not the address the execution will start at
				name := sym.Name
				if name == "" {
					name = "anonymous"
				}

				// log message is dependent on specific relocation type
				var typ string
				switch elf.R_ARM(relType) {
				case elf.R_ARM_TARGET1:
					typ = "TARGET1"
				case elf.R_ARM_ABS32:
					typ = "ABS32"
				}

				logger.Logf(mem.env, "ELF", "%s %s (%08x) => %08x", typ, name, secBeingRelocated.origin+offset, tgt)

			case elf.R_ARM_THM_PC22, elf.R_ARM_THM_JUMP24:
				// this value is labelled R_ARM_THM_CALL in objdump output
				//
				// "R_ARM_THM_PC22 Bits 0-10 encode the 11 most significant bits of
				// the branch offset, bits 0-10 of the next instruction word the 11
				// least significant bits. The unit is 2-byte Thumb instructions."
				// page 32 of "SWS ESPC 0003 A-08"

				ok, tgt, err := getStrongArmDefinition(mem, sym.Name)
				if err != nil {
					return fmt.Errorf("%s: %w", sym.Name, err)
				}
				if !ok {
					if sym.Section == elf.SHN_UNDEF {
						switch elf.R_ARM(relType) {
						case elf.R_ARM_THM_PC22:
							logger.Logf(logger.Allow, "ELF", "THM_PC22 section is undefined")
						case elf.R_ARM_THM_JUMP24:
							logger.Logf(logger.Allow, "ELF", "THM_JUMP24 section is undefined")
						}
						continue
					}

					n := ef.Sections[sym.Section].Name
					sec, ok := mem.sectionsByName[n]
					if !ok {
						return fmt.Errorf("can not find section (%s)", n)
					}

					tgt = sec.origin
					tgt += uint32(sym.Value)
				}

				tgt &= 0xfffffffe
				tgt -= (secBeingRelocated.origin + offset + 4)

				imm11 := (tgt >> 1) & 0x7ff
				imm10 := (tgt >> 12) & 0x3ff
				t1 := (tgt >> 22) & 0x01
				t2 := (tgt >> 23) & 0x01
				s := (tgt >> 24) & 0x01
				j1 := s
				j2 := s
				if t1 != 0x01 {
					j1 ^= 0x01
				}
				if t2 != 0x01 {
					j2 ^= 0x01
				}

				lo := uint16(0xf000 | (s << 10) | imm10)
				hi := uint16(0xd000 | (j1 << 13) | (j2 << 11) | imm11)
				opcode := uint32(lo) | (uint32(hi) << 16)

				// THM_JUMP24 seems to differ only in that it's a branch without link
				if elf.R_ARM(relType) == elf.R_ARM_THM_JUMP24 {
					opcode &= 0xbfffffff
				}

				// commit write
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:], opcode)

				// log relocated opcode depending on relocation type
				name := sym.Name
				if name == "" {
					name = "anonymous"
				}

				if elf.R_ARM(relType) == elf.R_ARM_THM_PC22 {
					logger.Logf(logger.Allow, "ELF", "THM_PC22 %s (%08x) => opcode %08x", name, secBeingRelocated.origin+offset, opcode)
				} else {
					logger.Logf(logger.Allow, "ELF", "THM_JUMP24 %s (%08x) no veneer => opcode %08x", name, secBeingRelocated.origin+offset, opcode)
				}

			case elf.R_ARM_REL32:
				if sym.Section == elf.SHN_UNDEF {
					logger.Logf(mem.env, "ELF", "REL32 section is undefined")
					continue
				}

				n := ef.Sections[sym.Section].Name
				sec, ok := mem.sectionsByName[n]
				if !ok {
					return fmt.Errorf("can not find section (%s) while relocating %s", n, sym.Name)
				}

				tgt := sec.origin
				tgt += uint32(sym.Value)

				// check address is recognised
				if mappedData, _ := mem.mapAddress(tgt, false); mappedData == nil {
					continue
				}

				// commit write
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:], tgt)
				if d, err := ef.Section(secBeingRelocated.name).Data(); err == nil {
					if offset >= uint32(len(d)) {
						return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
					}
					ef.ByteOrder.PutUint32(d[offset:], tgt)
				}

				logger.Logf(mem.env, "ELF", "REL32 %s (%08x) => %08x", sym.Name, secBeingRelocated.origin+offset, tgt)

			case elf.R_ARM_PREL31:
				if sym.Section&0xff00 == elf.SHN_UNDEF {
					logger.Logf(mem.env, "ELF", "PREL31 section is undefined")
					continue
				}
				return fmt.Errorf("PREL31 not fully supported")

			case elf.R_ARM_THM_MOVW_ABS_NC, elf.R_ARM_THM_MOVT_ABS:
				if sym.Section == elf.SHN_UNDEF {
					logger.Logf(logger.Allow, "ELF", "THM_MOVW/MOVT symbol is undefined")
					continue
				}

				n := ef.Sections[sym.Section].Name
				sec, ok := mem.sectionsByName[n]
				if !ok {
					return fmt.Errorf("cannot find section (%s) while relocating %s", n, sym.Name)
				}

				tgt := sec.origin
				tgt += uint32(sym.Value)
				tgt &= 0xfffffffe

				// "4.6.1.6 Static Thumb32 relocations"
				// of "ELF for the ARM Architecture, 24th November 2015"
				switch elf.R_ARM(relType) {
				case elf.R_ARM_THM_MOVW_ABS_NC:
					tgt &= 0x0000ffff
				case elf.R_ARM_THM_MOVT_ABS:
					tgt >>= 16
				}

				// extract fields. opposite of this (from thumb2_32bit.go)
				// 		imm16 := uint16((imm4 << 12) | (i << 11) | (imm3 << 8) | imm8)
				imm4 := (tgt >> 12) & 0x000f
				i := (tgt >> 11) & 0x0001
				imm3 := (tgt >> 8) & 0x0007
				imm8 := tgt & 0x00ff

				// opcode to be transformed
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				op := ef.ByteOrder.Uint32(secBeingRelocated.data[offset:])
				op = (op << 16) | (op >> 16)

				// clear bits
				op &= 0b11111011111100001000111100000000

				// set bits
				switch elf.R_ARM(relType) {
				case elf.R_ARM_THM_MOVT_ABS:
					op |= (imm4 << 16)
					op |= (i << 26)
					op |= (imm3 << 12)
					op |= imm8
				case elf.R_ARM_THM_MOVW_ABS_NC:
					op |= (imm4 << 16)
					op |= (i << 26)
					op |= (imm3 << 12)
					op |= imm8
				}

				// commit write
				if offset >= uint32(len(secBeingRelocated.data)) {
					return fmt.Errorf("relocation out-of-bounds (%s)", sym.Name)
				}
				ef.ByteOrder.PutUint32(secBeingRelocated.data[offset:], (op<<16)|(op>>16))

				switch elf.R_ARM(relType) {
				case elf.R_ARM_THM_MOVW_ABS_NC:
					logger.Logf(logger.Allow, "ELF", "THM_MOVW_ABS_NC %s (%08x) => %08x", sym.Name, secBeingRelocated.origin+offset, tgt)
				case elf.R_ARM_THM_MOVT_ABS:
					logger.Logf(logger.Allow, "ELF", "THM_MOVT_ABS %s (%08x) => %08x", sym.Name, secBeingRelocated.origin+offset, tgt)
				}

			default:
				return fmt.Errorf("unhandled ARM relocation type (%v)", relType)
			}
		}
	}

	// get pram
	if mem.pxe.enabled {
		// search for pRAM. if it can't be found the hasPXE is set to false
		mem.pxe.enabled = false
		for _, e := range mem.symbols {
			if e.Name == "pRAM" {
				mem.pxe.pRAM = uint32(e.Value)
				mem.pxe.enabled = true
				logger.Logf(mem.env, "ELF", "pRAM symbol found")
				logger.Logf(mem.env, "ELF", "PXE confirmed")
			}
		}
	}

	// strongarm program has been created so we adjust the memtop value
	mem.strongArmMemtop -= 1

	// strongarm address information
	logger.Logf(mem.env, "ELF", "strongarm: %08x to %08x (%d)",
		mem.strongArmOrigin, mem.strongArmMemtop, len(mem.strongArmProgram))

	// runInitialisation() must be run once ARM has been created

	return nil
}

// run any intialisation functions. leave resetPC value pointing to main function
func (mem *elfMemory) runInitialisation(arm *arm.ARM) error {
	// intialise stack pointer and link register
	//
	// the link register should really link to a program that will indicate the
	// program has ended. if we were emulating the real Uno/PlusCart firmware,
	// the link register would point to the resume address in the firmware
	mem.resetSP = mem.model.Regions["SRAM"].Origin | 0x0000ffdc
	mem.resetLR = mem.model.Regions["CCM"].Origin

	for _, typ := range []elf.SectionType{elf.SHT_PREINIT_ARRAY, elf.SHT_INIT_ARRAY} {
		for _, sec := range mem.sections {
			if sec.typ == typ {
				ptr := 0
				for ptr < len(sec.data) {
					mem.resetPC = mem.byteOrder.Uint32(sec.data[ptr:])
					ptr += 4
					if mem.resetPC == 0x00000000 {
						break // for loop
					}
					mem.resetPC &= 0xfffffffe

					logger.Logf(mem.env, "ELF", "running %s at %08x", sec.name, mem.resetPC)
					_, _ = arm.Run()
				}
			}
		}
	}

	// find entry point and use it to set the resetPC value. the Entry field in
	// the elf.File structure is no good for our purposes
	for _, s := range mem.symbols {
		if s.Name == "main" || s.Name == "elf_main" {
			sec := mem.sections[s.Section]
			if sec.typ != elf.SHT_PROGBITS {
				logger.Logf(mem.env, "ELF", "%s not in a section of SHT_PROGBITS type", s.Name)
				return nil
			}
			mem.resetPC = sec.origin + uint32(s.Value)
			mem.resetPC &= 0xfffffffe
			break // for loop
		}
	}

	return nil
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

func (mem *elfMemory) relocateStrongArmFunction(spec strongArmFunctionSpec) (uint32, error) {
	// strongarm functions must be on a 16bit boundary. I don't believe this
	// should ever happen with ELF but if it does we can add a padding byte to
	// correct. but for now, return an error so that we're forced to notice it
	// if it ever arises
	if mem.strongArmOrigin&0x01 == 0x01 {
		return 0, fmt.Errorf("misalignment of executable code. strongarm will be unreachable")
	}

	// address of new function in memory
	spec.origin = mem.strongArmMemtop
	spec.memtop = spec.origin + uint32(len(strongArmStub))

	// add null function to end of strongArmProgram array
	mem.strongArmProgram = append(mem.strongArmProgram, strongArmStub...)

	// update memtop of strongArm program
	mem.strongArmMemtop = spec.memtop

	// add specification for this strongarm function
	extend := int(mem.strongArmMemtop-mem.strongArmOrigin) - len(mem.strongArmFunctions) + 1
	for range extend {
		mem.strongArmFunctions = append(mem.strongArmFunctions, nil)
	}
	mem.strongArmFunctions[spec.origin-mem.strongArmOrigin] = &spec
	mem.strongArmFunctionsByName[spec.name] = &spec

	// although the code location of a strongarm function must be on a 16bit
	// boundary, the code is reached by interwork branching. we're using the
	// Thumb-2 instruction set so this means that the zero bit of the address
	// must be set to one
	//
	// interwork branching uses the BLX instruction. BLX ignores bit zero of the
	// address. this means that the correct (aligned) address will be used when
	// setting the program counter
	return spec.origin | 0b01, nil
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *elfMemory) MapAddress(addr uint32, write bool, executing bool) (*[]byte, uint32) {
	if addr >= mem.strongArmOrigin && addr <= mem.strongArmMemtop {
		// strong arm memory is not writeable
		if write {
			mem.lastMappedAddress = addr
			mem.lastMappedExecutable = false
			return nil, 0
		}

		// we've mapped the address to a strongarm function but we only want to
		// trigger the side-effects of mapping the address if the executing flag
		// is true AND if we've not deliberately inhibited strongarm access
		if executing && !mem.inhibitStrongarmAccess {
			// strongarm functions are indexed by the address of the first
			// instruction in the function
			//
			// however, MapAddress() is called with an address equal to the
			// execution address plus one. the plus one is intended to make sure
			// that the memory area is big enough when decoding 16bit instructions.
			// (ie. if the ARM has jumped to the last address in a memory area then
			// MapAddress() will return nil)
			//
			// with that in mind we must lookup the address minus one to determine
			// if the call address is a strongarm function. if it is not then that
			// means the ARM program has jumped to the wrong address
			//
			// note that we must also adjust the address by the strongArmOrigin
			// value because strongArmFunctions is a sparse array and not a map
			if f := mem.strongArmFunctions[addr-mem.strongArmOrigin-1]; f != nil {
				if f.support {
					mem.runStrongArmFunction(f.function)
				} else {
					if mem.stream.active {
						mem.setStrongArmFunction(f.function)
						for mem.strongarm.running.function != nil {
							mem.strongarm.running.function(mem)
						}
						if mem.stream.drain {
							mem.arm.Interrupt()
						}
					} else {
						mem.setStrongArmFunction(f.function)
						mem.arm.Interrupt()
					}
				}
			} else {
				// if the strongarm function can't be found then the program is
				// simply wanting to read data in the strongARM memory space (for
				// whatever reason)
			}
		}
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = true
		return &mem.strongArmProgram, mem.strongArmOrigin
	}

	return mem.mapAddress(addr, write)
}

func (mem *elfMemory) mapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = false
		return mem.sram.Data(), mem.sramOrigin
	}

	for _, s := range mem.sections {
		if s.isEmpty() || !s.inMemory() {
			continue // for loop
		}

		// special condition for executable sections to handle instances when
		// the address being mapped is at the very beginning of the memory block
		var adjust uint32
		if s.executable() {
			adjust = 1
		}

		if addr >= s.origin-adjust && addr <= s.memtop {
			if write && s.readOnly() {
				mem.lastMappedAddress = addr
				mem.lastMappedExecutable = false
				return nil, 0
			}

			// note pxe palette access
			if !write && mem.pxe.initialised {
				if addr >= mem.pxe.origin+PXEPaletteOrigin && addr <= mem.pxe.origin+PXEPaletteMemtop {
					colour := s.data[addr-s.origin]
					mem.pxe.pushLastPaletteAddr(colour, addr)
				}
			}

			mem.lastMappedAddress = addr
			mem.lastMappedExecutable = s.executable()
			return &s.data, s.origin
		}
	}

	// GPIO access means streaming will be disabled, at which point we start to
	// care less about performance
	if addr >= mem.gpio.dataOrigin && addr <= mem.gpio.dataMemtop {
		if mem.stream.active {
			logger.Logf(mem.env, "ELF", "disabling byte streaming: %d bytes in stream", mem.stream.ptr)
			mem.stream.disabled = true
			mem.stream.active = false
		}
		if !write && addr == mem.gpio.dataOrigin|ADDR_IDR {
			mem.arm.Interrupt()
		}
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = false
		return &mem.gpio.data, mem.gpio.dataOrigin
	}
	if addr >= mem.gpio.lookupOrigin && addr <= mem.gpio.lookupMemtop {
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = false
		return &mem.gpio.lookup, mem.gpio.lookupOrigin
	}

	// strongarm check is tested in MapAddress() so we don't really need to test
	// for it here except for those cases where mapAddress() is called directly.
	// in which case, performance doesn't matter
	if addr >= mem.strongArmOrigin && addr <= mem.strongArmMemtop {
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = true
		return &mem.strongArmProgram, mem.strongArmOrigin
	}

	// arg memory is likely only ever read once on program startup
	if addr >= argOrigin && addr <= argMemtop {
		mem.lastMappedAddress = addr
		mem.lastMappedExecutable = false
		return &mem.args, argOrigin
	}

	mem.lastMappedAddress = addr
	mem.lastMappedExecutable = false
	return nil, 0
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *elfMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}

// IsExecutable implements the arm.SharedMemory interface.
func (mem *elfMemory) IsExecutable(addr uint32) bool {
	if mem.lastMappedAddress == addr {
		return mem.lastMappedExecutable
	}
	_, _ = mem.mapAddress(addr, false)
	return mem.lastMappedExecutable
}

// Segments implements the mapper.CartStatic interface
func (mem *elfMemory) Segments() []mapper.CartStaticSegment {
	segments := []mapper.CartStaticSegment{
		{
			Name:   "SRAM",
			Origin: mem.sramOrigin,
			Memtop: mem.sramMemtop,
		},
		{
			Name:   "GPIO",
			Origin: mem.gpio.dataOrigin,
			Memtop: mem.gpio.dataMemtop,
		},
		{
			Name:   "StrongARM Program",
			Origin: mem.strongArmOrigin,
			Memtop: mem.strongArmMemtop,
		},
	}

	var sections []mapper.CartStaticSegment

	for _, n := range mem.sectionNames {
		sec := mem.sectionsByName[n]
		if sec.inMemory() {
			sections = append(sections, mapper.CartStaticSegment{
				Name:   sec.name,
				Origin: sec.origin,
				Memtop: sec.memtop,
			})
		}
	}

	segments = append(segments, mapper.CartStaticSegment{
		Name:        "Sections",
		SubSegments: sections,
	})

	return segments
}

// Reference implements the mapper.CartStatic interface
func (mem *elfMemory) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case "SRAM":
		return *mem.sram.Data(), true
	case "GPIO":
		return mem.gpio.data, true
	case "StrongARM Program":
		return mem.strongArmProgram, true
	default:
		if sec, ok := mem.sectionsByName[segment]; ok {
			return sec.data, true
		}
	}
	return []uint8{}, false
}

// Read8bit implements the mapper.CartStatic interface
func (m *elfMemory) Read8bit(addr uint32) (uint8, bool) {
	mem, origin := m.mapAddress(addr, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

// Read16bit implements the mapper.CartStatic interface
func (m *elfMemory) Read16bit(addr uint32) (uint16, bool) {
	mem, origin := m.mapAddress(addr, false)
	addr -= origin
	if mem == nil || len(*mem) < 2 || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

// Read32bit implements the mapper.CartStatic interface
func (m *elfMemory) Read32bit(addr uint32) (uint32, bool) {
	mem, origin := m.mapAddress(addr, false)
	addr -= origin
	if mem == nil || len(*mem) < 4 || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// Write8bit implements the mapper.CartStatic interface
func (m *elfMemory) Write8bit(addr uint32, data uint8) bool {
	mem, origin := m.mapAddress(addr, true)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return false
	}
	(*mem)[addr] = data
	return true
}
