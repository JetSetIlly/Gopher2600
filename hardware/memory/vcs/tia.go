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

package vcs

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// All TIA addresses when read by the CPU should be masked by the TIADrivenPins
// value. This is because only the pin 6 and 7 are connected to the data bus.
//
// Note that although the "Stella's Programmer's Guide" suggests that the
// CXBLPF and CXPPMM only have pin 7 connected, in actual fact they are wired
// just like the other collision registers.
//
// Similarly for the INPTx reguisters. As reported in this bug report:
// https://github.com/JetSetIlly/Gopher2600/issues/16#issue-1083935291
//
// the last two address location in TIA memory are "undefined" according to the
// "Stella Programmer's Guide" but they are readable anyway and are wired the
// same as the collision and INPTx registers.
//
// Explanation of how the TIADrivenPins mask affects the read value
// ----------------------------------------------------------------
//
// If the CPU wants to read the contents of the CXM1P register, it can use the
// address 0x0d to do so.
//
//	LDA 0x01
//
// If there are no collisions (between missile 1 and either player, in this
// case) than the value of the most significant bits are zero. The lower six
// bits are not part of the CXM1P register and are left undefined by the TIA
// when the data is put on the bus. The lower bits of the LDA operation are in
// fact "left over" from the address. In our example, the lowest six bits are
//
//	0bxx000001
//
// meaning the the returned data is in fact 0x01 and not 0x00, as you might
// expect.  Things get interesting when we use mirrored addresses. If instead
// of 0x01 we used the mirror address 0x11, the lowest six bits are:
//
//	0bxx01001
//
// meaning that the returned value is 0x11 and not (again, as you might expect)
// 0x00 or even 0x01.
//
// So what happens if there is sprite collision information in the register?
// Meaning that the top bits are not necessarily zero. Let's say there is a
// collusion between missile 1 and player 0, the data before masking will be
//
//	0b01000000
//
// If we used address 0x11 to load this value, we would in fact, get this
// pattern (0x51 in hex):
//
//	0b01010001
//
// Now, if all ROMs read and interpreted chip registers only as they're
// supposed to (defails in the 2600 programmer's guide) then none of this would
// matter but some ROMs do make use of the extra bits, and so we must account
// for it in emulation.
//
// It's worth noting that the above is implicitly talking about zero-page
// addressing; but masking also occurs with regular two-byte addressing. The
// key to understanding is that the masking is applied to the most recent byte
// of the address to be put on the address bus*. In all cases, this is the
// most-significant byte. So, if the requested address is 0x171, the bit
// pattern for the address is:
//
//	0x0000000101110001
//
// the most significant byte in this pattern is 0x00000001 and so the data
// retreived is AND-ed with that. The mapped address for 0x171 incidentally, is
// 0x01, which is the CXM1P register also used in the examples above.
const TIADrivenPins = uint8(0b11000000)

// TIAMemory defines the information for and operations allowed for those
// memory mems accessed by the VCS chips as well as the CPU.
type TIAMemory struct {
	instance *instance.Instance

	// memory stores the values read by the CPU and written to by the TIA
	memory []uint8

	// addresses used by Peek(), Write(), etc. are normalised by we still need
	// to reduce the address to the array size. we can do this by XORing with
	// the origin value
	origin uint16

	// when the CPU writes to a TIA address it is not writing to memory in the
	// way we might expect. instead we note the address that has been written
	// to, and a boolean true to indicate that a write has been performed by
	// the CPU
	writeSignal  bool
	writeAddress uint16
	writeData    uint8
}

// NewTIAMemory is the preferred method of initialisation for the TIA memory chip.
func NewTIAMemory(instance *instance.Instance) *TIAMemory {
	chip := &TIAMemory{
		instance: instance,
		origin:   memorymap.OriginTIA,
	}

	// allocate the minimal amount of memory
	chip.memory = make([]uint8, memorymap.MemtopTIA-memorymap.OriginTIA+1)

	// initial values
	chip.memory[chipbus.INPT1] = 0x00
	chip.memory[chipbus.INPT2] = 0x00
	chip.memory[chipbus.INPT3] = 0x00
	chip.memory[chipbus.INPT4] = 0x80
	chip.memory[chipbus.INPT5] = 0x80

	return chip
}

// Snapshot creates a copy of TIARegisters in its current state.
func (mem *TIAMemory) Snapshot() *TIAMemory {
	n := *mem
	n.memory = make([]uint8, len(mem.memory))
	copy(n.memory, mem.memory)
	return &n
}

// Reset contents of TIARegisters.
func (mem *TIAMemory) Reset() {
	for i := range mem.memory {
		mem.memory[i] = 0
	}
}

// Peek is an implementation of memory.DebugBus. Address must be normalised.
func (mem *TIAMemory) Peek(address uint16) (uint8, error) {
	if cpubus.Read[address] == cpubus.NotACPUBusRegister {
		return 0, fmt.Errorf("%w: %04x", cpubus.AddressError, address)
	}
	return mem.memory[address^mem.origin], nil
}

// Poke is an implementation of memory.DebugBus. Address must be normalised.
func (mem *TIAMemory) Poke(address uint16, value uint8) error {
	mem.memory[address^mem.origin] = value
	return nil
}

// ChipRead is an implementation of memory.ChipBus.
func (mem *TIAMemory) ChipHasChanged() (chipbus.ChangedRegister, bool) {
	if mem.writeSignal {
		mem.writeSignal = false
		return chipbus.ChangedRegister{Address: mem.writeAddress, Value: mem.writeData, Register: cpubus.Write[mem.writeAddress]}, true
	}

	return chipbus.ChangedRegister{}, false
}

// ChipWrite is an implementation of memory.ChipBus
func (mem *TIAMemory) ChipWrite(reg chipbus.Register, data uint8) {
	mem.memory[reg] = data
}

// ChipRefer is an implementation of memory.ChipBus.
func (mem *TIAMemory) ChipRefer(reg chipbus.Register) uint8 {
	return mem.memory[reg]
}

// LatsReadAddress is an implementation of memory.ChipBus.
func (mem *TIAMemory) LastReadAddress() (bool, uint16) {
	return false, 0
}

// Read is an implementation of memory.CPUBus. Address must be mapped.
//
// Returned data should be masked and randomised as appropriate according to
// the TIADrivenPins mask.
func (mem *TIAMemory) Read(address uint16) (uint8, uint8, error) {
	return mem.memory[address^mem.origin], TIADrivenPins, nil
}

// Write is an implementation of memory.CPUBus. Address must be mapped.
func (mem *TIAMemory) Write(address uint16, data uint8) error {
	// signal that chip memory has been changed. see ChipHasChanged() function
	mem.writeAddress = address
	mem.writeSignal = true
	mem.writeData = data

	return nil
}
