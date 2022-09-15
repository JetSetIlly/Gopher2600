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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

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
	logger.Logf("ACE", "sram: %08x to %08x (%dbytes)", cart.mem.sramOrigin, cart.mem.sramMemtop, len(cart.mem.sram))
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
	cart.arm.Plumb(nil, cart.mem, cart)
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
