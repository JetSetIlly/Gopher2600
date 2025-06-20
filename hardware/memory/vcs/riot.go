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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type RIOTMemory struct {
	env *environment.Environment

	// memory stores the values read by the CPU and written to by the RIOT
	memory []uint8

	// addresses used by Peek(), Write(), etc. are normalised by we still need
	// to reduce the address to the array size. we can do this by XORing with
	// the origin value
	origin uint16

	// when the CPU writes to chip memory it is not writing to memory in the
	// way we might expect. instead we note the address that has been written
	// to, and a boolean true to indicate that a write has been performed by
	// the CPU
	writeSignal  bool
	writeAddress uint16
	writeData    uint8

	readSignal  bool
	readAddress uint16

	// after poke is called after a poke operation
	pokeNotify chipbus.PokeNotify
}

// NewRIOTMemory is the preferred method of initialisation for the RIOT memory mem
func NewRIOTMemory(env *environment.Environment) *RIOTMemory {
	chip := &RIOTMemory{
		env:    env,
		origin: memorymap.OriginRIOT,
	}

	// allocate the minimal amount of memory
	chip.memory = make([]uint8, memorymap.MemtopRIOT-memorymap.OriginRIOT+1)

	// SWCHA should be set when a peripheral is attached

	// SWCHB is set in panel peripheral

	return chip
}

// SetPokeNotify implements the chipbus.Memory interface
func (mem *RIOTMemory) SetPokeNotify(pokeNotify chipbus.PokeNotify) {
	mem.pokeNotify = pokeNotify
}

// Snapshot creates a copy of RIOTMemory in its current state
func (mem *RIOTMemory) Snapshot() *RIOTMemory {
	n := *mem
	n.memory = make([]uint8, len(mem.memory))
	copy(n.memory, mem.memory)
	return &n
}

// Reset contents of RIOTMemory
func (mem *RIOTMemory) Reset() {
	for i := range mem.memory {
		mem.memory[i] = 0
	}
}

// Peek is an implementation of the memory.Area interface
func (mem *RIOTMemory) Peek(address uint16) (uint8, error) {
	if cpubus.ReadAddress[address] == cpubus.UnnamedAddress {
		return 0, fmt.Errorf("%w: %04x", cpubus.AddressError, address)
	}
	return mem.memory[address^mem.origin], nil
}

// Poke is an implementation of the memory.Area interface
func (mem *RIOTMemory) Poke(address uint16, value uint8) error {
	if cpubus.WriteAddress[address] == cpubus.UnnamedAddress {
		return fmt.Errorf("%w: %04x", cpubus.AddressError, address)
	}
	mem.memory[address^mem.origin] = value

	if mem.pokeNotify != nil {
		mem.pokeNotify.AfterPoke(chipbus.ChangedRegister{
			Address:  address,
			Value:    value,
			Register: cpubus.WriteAddress[address],
		})
	}
	return nil
}

// ChipRead is an implementation of memory.ChipBus
func (mem *RIOTMemory) ChipHasChanged() (chipbus.ChangedRegister, bool) {
	if mem.writeSignal {
		mem.writeSignal = false
		return chipbus.ChangedRegister{Address: mem.writeAddress, Value: mem.writeData, Register: cpubus.WriteAddress[mem.writeAddress]}, true
	}

	return chipbus.ChangedRegister{}, false
}

// ChipWrite implements the chipbus.Memory interface
func (mem *RIOTMemory) ChipWrite(reg chipbus.Register, data uint8) {
	mem.memory[reg] = data
}

// ChipRefer implements the chipbus.Memory interface
func (mem *RIOTMemory) ChipRefer(reg chipbus.Register) uint8 {
	return mem.memory[reg]
}

// LastReadAddress implements the chipbus.Memory interface
func (mem *RIOTMemory) LastReadAddress() (bool, uint16) {
	if mem.readSignal {
		mem.readSignal = false
		return true, mem.readAddress
	}
	return false, 0
}

// Read is an implementation of the memory.Area interface
func (mem *RIOTMemory) Read(address uint16) (uint8, uint8, error) {
	mem.readAddress = address
	mem.readSignal = true

	// if the address is not a valid read adress for the CPU, then we blindly
	// return the value in the array, which in that case will be zero
	//
	// I'm not sure if the pins are driven in that case. if they aren't then we
	// should return zero for the value AND the mask. a value other than 0xff
	// for the mask instructs the memory package to mutate the value returned to
	// the CPU
	return mem.memory[address^mem.origin], 0xff, nil
}

// Write is an implementation of the memory.Area interface
func (mem *RIOTMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory mem has been serviced. this
	// shouldn't ever happen.
	//
	// UPDATE: this is a protection against an imcomplete RIOT implementation. it
	// is complete and this code path has never run to my knowledge. removing
	// for performance reasons (23/05/2022)
	//
	// if mem.writeSignal {
	// 	panic(fmt.Sprintf("unserviced write to RIOT memory (%#04x)", mem.writeAddress))
	// }

	// signal that chip memory has been changed. see ChipHasChanged() function
	mem.writeAddress = address
	mem.writeSignal = true
	mem.writeData = data

	return nil
}
