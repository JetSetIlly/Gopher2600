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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
)

// ChipMemory defines the information for and operations allowed for those
// memory areas accessed by the VCS chips as well as the CPU.
type ChipMemory struct {
	bus.DebugBus
	bus.ChipBus
	bus.CPUBus

	prefs *preferences.Preferences

	// because we're servicing two different memory areas with this type, we
	// need to store the origin and memtop values here, rather than using the
	// constants from the memorymap package directly
	origin uint16
	memtop uint16

	memory []uint8

	// when the CPU writes to chip memory it is not writing to memory in the
	// way we might expect. instead we note the address that has been written
	// to, and a boolean true to indicate that a write has been performed by
	// the CPU
	writeAddress uint16
	writeData    uint8
	writeSignal  bool

	// readRegister works slightly different than writeAddress. it stores the
	// register *name* of the last memory location *read* by the CPU
	readRegister string
}

// Snapshot creates a copy of ChipMemory in its current state.
func (area *ChipMemory) Snapshot() *ChipMemory {
	n := *area
	n.memory = make([]uint8, len(area.memory))
	copy(n.memory, area.memory)
	return &n
}

// Peek is an implementation of memory.DebugBus. Address must be normalised.
func (area ChipMemory) Peek(address uint16) (uint8, error) {
	sym := addresses.Read[address]
	if sym == "" {
		return 0, curated.Errorf(bus.AddressError, address)
	}
	return area.memory[address^area.origin], nil
}

// Reset contents of ChipMemory.
func (area *ChipMemory) Reset() {
	for i := range area.memory {
		if area.prefs != nil && area.prefs.RandomState.Get().(bool) {
			area.memory[i] = uint8(area.prefs.RandSrc.Intn(0xff))
		} else {
			area.memory[i] = 0
		}
	}
}

// Poke is an implementation of memory.DebugBus. Address must be normalised.
func (area ChipMemory) Poke(address uint16, value uint8) error {
	area.memory[address^area.origin] = value
	return nil
}

// ChipRead is an implementation of memory.ChipBus.
func (area *ChipMemory) ChipRead() (bool, bus.ChipData) {
	if area.writeSignal {
		area.writeSignal = false
		return true, bus.ChipData{Name: addresses.Write[area.writeAddress], Value: area.writeData}
	}

	return false, bus.ChipData{}
}

// ChipWrite is an implementation of memory.ChipBus.
func (area *ChipMemory) ChipWrite(reg addresses.ChipRegister, data uint8) {
	area.memory[reg] = data
}

// LastReadRegister is an implementation of memory.ChipBus.
func (area *ChipMemory) LastReadRegister() string {
	r := area.readRegister
	area.readRegister = ""
	return r
}

// Read is an implementation of memory.CPUBus. Address must be normalised.
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	// note the name of the register that we are reading
	area.readRegister = addresses.Read[address]

	// do not allow reads from memory that do not have symbol name
	if _, ok := addresses.ReadSymbols[address]; !ok {
		return 0, curated.Errorf(bus.AddressError, address)
	}

	return area.memory[address^area.origin], nil
}

// Write is an implementation of memory.CPUBus. Address must be normalised.
func (area *ChipMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory area has been serviced. this
	// shouldn't ever happen.
	if area.writeSignal {
		panic(fmt.Sprintf("unserviced write to chip memory (%s)", addresses.Write[area.writeAddress]))
	}

	// do not allow writes to memory that do not have symbol name
	if _, ok := addresses.WriteSymbols[address]; !ok {
		return curated.Errorf(bus.AddressError, address)
	}

	// signal the chips that their chip memory has been written to
	area.writeAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}
