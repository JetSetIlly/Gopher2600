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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

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
	cart.mem.arm = cart.arm

	logger.Logf("ELF", ".text program: %08x to %08x (%d)", cart.mem.textSectionOrigin, cart.mem.textSectionMemtop, cart.mem.textSectionMemtop-cart.mem.textSectionOrigin)
	logger.Logf("ELF", ".data section: %08x to %08x (%d)", cart.mem.dataSectionOrigin, cart.mem.dataSectionMemtop, cart.mem.dataSectionMemtop-cart.mem.dataSectionOrigin)
	logger.Logf("ELF", ".rodata section: %08x to %08x (%d)", cart.mem.rodataSectionOrigin, cart.mem.rodataSectionMemtop, cart.mem.rodataSectionMemtop-cart.mem.rodataSectionOrigin)
	logger.Logf("ELF", ".bss section: %08x to %08x (%d)", cart.mem.bssSectionOrigin, cart.mem.bssSectionMemtop, cart.mem.bssSectionMemtop-cart.mem.bssSectionOrigin)
	logger.Logf("ELF", "GPIO IN: %08x to %08x", cart.mem.gpio.AOrigin, cart.mem.gpio.AMemtop)
	logger.Logf("ELF", "GPIO OUT: %08x to %08x", cart.mem.gpio.BOrigin, cart.mem.gpio.BMemtop)

	cart.mem.busStuffingInit()
	cart.mem.setStrongArmFunction(cart.mem.emulationInit)

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
	return cart.mem.gpio.B[fromArm_Opcode], nil
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

// try to run strongarm function. returns success.
func (cart *Elf) runStrongarm(addr uint16, data uint8) bool {
	if cart.mem.strongarm.running.function != nil {
		cart.mem.gpio.B[toArm_data] = data
		cart.mem.gpio.A[toArm_address] = uint8(addr)
		cart.mem.gpio.A[toArm_address+1] = uint8(addr >> 8)
		cart.mem.strongarm.running.function()

		if cart.mem.strongarm.running.function == nil {
			cart.arm.Run()
			if cart.mem.strongarm.running.function != nil {
				cart.mem.strongarm.running.function()
			}
		}

		return true
	}
	return false
}

// Listen implements the mapper.CartMapper interface.
func (cart *Elf) Listen(addr uint16, data uint8) {
	if cart.runStrongarm(addr, data) {
		return
	}

	// set data first and continue once. this seems to be necessary to allow
	// the PlusROM exit rountine to work correctly
	cart.mem.gpio.B[toArm_data] = data

	cart.arm.Run()
	if cart.runStrongarm(addr, data) {
		return
	}

	// set address and continue
	cart.mem.gpio.A[toArm_address] = uint8(addr)
	cart.mem.gpio.A[toArm_address+1] = uint8(addr >> 8)

	cart.arm.Run()
	if cart.runStrongarm(addr, data) {
		return
	}

	cart.arm.Run()
	if cart.runStrongarm(addr, data) {
		return
	}

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

// BusStuff implements the mapper.CartBusStuff interface.
func (cart *Elf) BusStuff() (uint8, bool) {
	return cart.mem.busStuffData, cart.mem.busStuff
}
