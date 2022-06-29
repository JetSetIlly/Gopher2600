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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
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

	sram       []byte
	sramOrigin uint32
	sramMemtop uint32

	arm yieldARM
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

	// offset into the data array for start of ARM program. not entirely sure
	// of the significance of the jumpVector value or what it refers to
	const jumpVector = 0x08020201
	dataOffset := mem.resetPC - jumpVector - mem.model.FlashOrigin

	// align reset PC value (maybe this should be done in the ARM package once
	// the value has been received - on second thoughts, no we shouldn't do
	// this because we need to know the true value of resetPC in MapAddress()
	// below)
	mem.resetPC &= 0xfffffffe

	// copy vcs program, leaving room for virtual arguments
	mem.vcsProgram = make([]byte, len(data))
	copy(mem.vcsProgram[aceStartOfVCSProgram:], data)
	mem.vcsOrigin = mem.model.FlashOrigin
	mem.vcsMemtop = mem.vcsOrigin + uint32(len(mem.vcsProgram))

	// copy arm program
	mem.armProgram = make([]byte, romSize)
	copy(mem.armProgram, data[dataOffset:])
	mem.armOrigin = mem.resetPC
	mem.armMemtop = mem.armOrigin + uint32(len(mem.armProgram))

	// define the Thumb-2 bytecode fo a function whose only purpose is to jump
	// back to where it came from bytecode is for instruction "BX LR" with a
	// "true" value in R0
	nullFunction := []byte{
		0x00,       // for alignment
		0x01, 0x20, // MOV R1, #1 (the function returns true)
		0x70, 0x47, // BX LR
	}

	// not sure why we need the +3 but the incorrect address is loaded by the
	// BLX function. I think it must be due to how the ARM pipeline and
	// alignment works but I'm not sure
	nullFunctionAddress := mem.resetPC + uint32(len(mem.armProgram)) + 4

	// append function to end of flash
	mem.armProgram = append(mem.armProgram, nullFunction...)

	// set virtual arguments. values and information in the PlusCart firmware
	// source:
	//
	// atari-2600-pluscart-master/source/STM32firmware/PlusCart/Src

	startOfVCSProgram := mem.vcsOrigin + aceStartOfVCSProgram
	mem.vcsProgram[0] = uint8(startOfVCSProgram)
	mem.vcsProgram[1] = uint8(startOfVCSProgram >> 8)
	mem.vcsProgram[2] = uint8(startOfVCSProgram >> 16)
	mem.vcsProgram[3] = uint8(startOfVCSProgram >> 24)
	mem.vcsProgram[4] = uint8(mem.model.SRAMOrigin)
	mem.vcsProgram[5] = uint8(mem.model.SRAMOrigin >> 8)
	mem.vcsProgram[6] = uint8(mem.model.SRAMOrigin >> 16)
	mem.vcsProgram[7] = uint8(mem.model.SRAMOrigin >> 24)

	// addresses of func_reboot_into_cartridge() and emulate_firmware_cartridge()
	// for our purposes, the function needs only to jump back to the link address
	mem.vcsProgram[8] = uint8(nullFunctionAddress)
	mem.vcsProgram[9] = uint8(nullFunctionAddress >> 8)
	mem.vcsProgram[10] = uint8(nullFunctionAddress >> 16)
	mem.vcsProgram[11] = uint8(nullFunctionAddress >> 24)
	mem.vcsProgram[12] = uint8(nullFunctionAddress)
	mem.vcsProgram[13] = uint8(nullFunctionAddress >> 8)
	mem.vcsProgram[14] = uint8(nullFunctionAddress >> 16)
	mem.vcsProgram[15] = uint8(nullFunctionAddress >> 24)

	// not setting system clock or version arguments
	copy(mem.vcsProgram[16:20], []byte{0x00, 0x00, 0x00, 0x01})
	copy(mem.vcsProgram[20:24], []byte{0x00, 0x00, 0x00, 0x02})
	copy(mem.vcsProgram[24:28], []byte{0x00, 0x00, 0x00, 0x03})

	// list termination value
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

	// SRAM creation
	mem.sram = make([]byte, mem.resetSP-mem.model.SRAMOrigin)
	mem.sramOrigin = mem.model.SRAMOrigin
	mem.sramMemtop = mem.sramOrigin + uint32(len(mem.sram))

	return mem, nil
}

func (mem *aceMemory) Snapshot() *aceMemory {
	m := *mem
	return &m
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
func (mem *aceMemory) ResetVectors() (uint32, uint32, uint32) {
	return mem.resetSP, mem.resetLR, mem.resetPC
}

// Ace implements the mapper.CartMapper interface.
type Ace struct {
	instance  *instance.Instance
	version   string
	pathToROM string
	arm       *arm.ARM
	mem       *aceMemory
}

// NewAce is the preferred method of initialisation for the Ace type.
func NewAce(instance *instance.Instance, pathToROM string, version string, data []byte) (mapper.CartMapper, error) {
	cart := &Ace{
		instance:  instance,
		version:   version,
		pathToROM: pathToROM,
	}

	var err error
	cart.mem, err = newAceMemory(version, data)
	if err != nil {
		return nil, err
	}

	cart.arm = arm.NewARM(arm.ARMv7_M, arm.MAMfull, cart.mem.model, cart.instance.Prefs.ARM, cart.mem, cart, cart.pathToROM)
	cart.mem.arm = cart.arm

	logger.Logf("ACE", "vcs program: %08x to %08x", cart.mem.vcsOrigin, cart.mem.vcsMemtop)
	logger.Logf("ACE", "arm program: %08x to %08x", cart.mem.armOrigin, cart.mem.armMemtop)
	logger.Logf("ACE", "GPIO IN: %08x to %08x", cart.mem.gpioAOrigin, cart.mem.gpioAMemtop)
	logger.Logf("ACE", "GPIO OUT: %08x to %08x", cart.mem.gpioBOrigin, cart.mem.gpioBMemtop)

	cart.arm.Run()

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Ace) MappedBanks() string {
	return fmt.Sprintf("Bank: none")
}

// ID implements the mapper.CartMapper interface.
func (cart *Ace) ID() string {
	return fmt.Sprintf("ACE (%s)", cart.version)
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Ace) Snapshot() mapper.CartMapper {
	n := *cart
	n.mem = cart.mem.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Ace) Plumb() {
	cart.arm.Plumb(cart.mem, cart)
}

// Reset implements the mapper.CartMapper interface.
func (cart *Ace) Reset() {
}

// Read implements the mapper.CartMapper interface.
func (cart *Ace) Read(addr uint16, passive bool) (uint8, error) {
	if passive {
		cart.Listen(addr|memorymap.OriginCart, 0x00)
	}
	return cart.mem.gpioB[fromArm_Opcode], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *Ace) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if passive || poke {
		return nil
	}

	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Ace) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Ace) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *Ace) Patch(_ int, _ uint8) error {
	return curated.Errorf("ACE: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *Ace) Listen(addr uint16, data uint8) {
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
func (cart *Ace) Step(clock float32) {
	cart.arm.Step(clock)
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *Ace) CopyBanks() []mapper.BankContent {
	return nil
}

// implements arm.CartridgeHook interface.
func (cart *Ace) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}
