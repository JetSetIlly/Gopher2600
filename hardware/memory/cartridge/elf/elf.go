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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
)

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

	sram       []byte
	sramOrigin uint32
	sramMemtop uint32
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
		return nil, curated.Errorf("ELF: .rel.text is not of type SHT_REL")
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
	mem.gpioAOrigin = 0x00000010
	mem.gpioAMemtop = mem.gpioAOrigin | gpio_memtop

	mem.gpioB = make([]byte, gpio_memtop)
	mem.gpioBOrigin = 0x000000a0
	mem.gpioBMemtop = mem.gpioBOrigin | gpio_memtop

	// default NOP instruction for opcode
	mem.gpioB[fromArm_Opcode] = 0xea

	// SRAM creation
	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram))

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
				v := uint32(mem.gpioAOrigin | toArm_address)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "DATA_ODR":
				v := uint32(mem.gpioBOrigin | fromArm_Opcode)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "DATA_MODER":
				v := uint32(mem.gpioBOrigin | gpio_mode)
				write(offset, v)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)
			case "vcsJsr6":
				v := uint32(mem.armMemtop + 3)
				write(offset, v)
				vcsJsr6 := []byte{
					0xb0, 0x27, // MOV R7, #$20
					0x70, 0x47, // BX LR
				}
				mem.armProgram = append(mem.armProgram, vcsJsr6...)
				logger.Logf("ELF", "%08x %s => %08x", offset, n, v)

				mem.armMemtop += uint32(len(vcsJsr6))
			default:
				return nil, curated.Errorf("ELF: unrelocated symbol (%s)", n)
			}
		default:
			return nil, curated.Errorf("ELF: unhandled ARM relocation type")
		}
	}

	return mem, nil
}

func (mem *elfMemory) Snapshot() *elfMemory {
	m := *mem
	return &m
}

// MapAddress implements the arm.SharedMemory interface.
func (mem *elfMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	if addr >= mem.gpioAOrigin && addr <= mem.gpioAMemtop {
		return &mem.gpioA, addr - mem.gpioAOrigin
	}
	if addr >= mem.gpioBOrigin && addr <= mem.gpioBMemtop {
		return &mem.gpioB, addr - mem.gpioBOrigin
	}
	if addr >= mem.armOrigin && addr <= mem.armMemtop {
		return &mem.armProgram, addr - mem.resetPC
	}
	if addr >= mem.vcsOrigin && addr <= mem.vcsMemtop {
		return &mem.vcsProgram, addr - mem.vcsOrigin
	}
	if addr >= mem.sramOrigin && addr <= mem.sramMemtop {
		return &mem.sram, addr - mem.model.SRAMOrigin
	}

	return nil, addr
}

// ResetVectors implements the arm.SharedMemory interface.
func (mem *elfMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}

// Elf implements the mapper.CartMapper interface.
type Elf struct {
	instance  *instance.Instance
	version   string
	pathToROM string
	arm       *arm.ARM
	mem       *elfMemory
}

// NewElf is the preferred method of initialisation for the Elf type.
func NewElf(instance *instance.Instance, pathToROM string) (mapper.CartMapper, error) {
	f, err := elf.Open(pathToROM)
	if err != nil {
		return nil, curated.Errorf("ELF: %v", err)
	}
	defer f.Close()

	// sanity checks on ELF data
	if f.FileHeader.Machine != elf.EM_ARM {
		return nil, curated.Errorf("ELF: is not ARM")
	}
	if f.FileHeader.ByteOrder != binary.LittleEndian {
		return nil, curated.Errorf("ELF: is not little-endian")
	}
	if f.FileHeader.Version != elf.EV_CURRENT {
		return nil, curated.Errorf("ELF: unknown version")
	}
	if f.FileHeader.Type != elf.ET_REL {
		return nil, curated.Errorf("ELF: is not relocatable")
	}

	cart := &Elf{
		instance:  instance,
		pathToROM: pathToROM,
	}

	cart.mem, err = newElfMemory(f)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(arm.ARMv7_M, arm.MAMfull, cart.mem.model, cart.instance.Prefs.ARM, cart.mem, cart, cart.pathToROM)
	cart.arm.AddReadWatch(cart.mem.gpioAOrigin | toArm_address)

	logger.Logf("ELF", "vcs program: %08x to %08x", cart.mem.vcsOrigin, cart.mem.vcsMemtop)
	logger.Logf("ELF", "arm program: %08x to %08x", cart.mem.armOrigin, cart.mem.armMemtop)
	logger.Logf("ELF", "GPIO IN: %08x to %08x", cart.mem.gpioAOrigin, cart.mem.gpioAMemtop)
	logger.Logf("ELF", "GPIO OUT: %08x to %08x", cart.mem.gpioBOrigin, cart.mem.gpioBMemtop)

	cart.arm.Run()
	panic(1)

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Elf) MappedBanks() string {
	return fmt.Sprintf("Bank: none")
}

// ID implements the mapper.CartMapper interface.
func (cart *Elf) ID() string {
	return "ELF"
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Elf) Snapshot() mapper.CartMapper {
	n := *cart
	n.mem = cart.mem.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Elf) Plumb() {
	cart.arm.Plumb(cart.mem, cart)
}

// Reset implements the mapper.CartMapper interface.
func (cart *Elf) Reset() {
}

// Read implements the mapper.CartMapper interface.
func (cart *Elf) Read(addr uint16, passive bool) (uint8, error) {
	if passive {
		cart.Listen(addr|memorymap.OriginCart, 0x00)
	}
	return cart.mem.gpioB[fromArm_Opcode], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *Elf) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if passive || poke {
		return nil
	}

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

// Patch implements the mapper.CartMapper interface.
func (cart *Elf) Patch(_ int, _ uint8) error {
	return curated.Errorf("ELF: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *Elf) Listen(addr uint16, data uint8) {
	// from dpc example:
	// reading of addresses is
	//		ldr	r2, [r1, #16] (opcode 690a)
	// r1 contains 0x40020c00 which is an address in gpioA
	//
	// reading of data is
	//		ldr.w	r0, [lr, #16] (opcode f8de 0010)
	// lr contains 0x40020800 which is an address in gpioB

	// set data first and continue once. this seems to be necessary to allow
	// the PlusROM exit rountine to work correctly
	cart.mem.gpioB[toArm_data] = data
	cart.arm.Run()

	// set address and continue x4
	cart.mem.gpioA[toArm_address] = uint8(addr)
	cart.mem.gpioA[toArm_address+1] = uint8(addr >> 8)
	cart.arm.Run()
	cart.arm.Run()
	cart.arm.Run()
	cart.arm.Run()

	// we must understand that the above synchronisation is almost certainly
	// "wrong" in the general sense. it works for the examples seen so far but
	// that means nothing
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

// NewFrame implements the protocol.NewFrame interface.
func (cart *Elf) NewFrame(_ television.FrameInfo) error {
	cart.arm.UpdatePrefs()
	return nil
}
