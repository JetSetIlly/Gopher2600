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
	"encoding/hex"

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// RAM represents the 128bytes of RAM in the PIA 6532 chip, found in the Atari
// VCS.
type RAM struct {
	bus.DebugBus
	bus.CPUBus
	RAM []uint8
}

// NewRAM is the preferred method of initialisation for the RAM memory area
func NewRAM() *RAM {
	ram := &RAM{}

	// allocate the mininmal amount of memory
	ram.RAM = make([]uint8, memorymap.MemtopRAM-memorymap.OriginRAM+1)

	return ram
}

func (ram RAM) String() string {
	return hex.Dump(ram.RAM)
}

// Peek is the implementation of memory.DebugBus. Address must be
// normalised.
func (ram RAM) Peek(address uint16) (uint8, error) {
	return ram.Read(address)
}

// Poke is the implementation of memory.DebugBus. Address must be
// normalised.
func (ram RAM) Poke(address uint16, value uint8) error {
	return ram.Write(address, value)
}

// Read is an implementatio of memory.ChipBus. Address must be normalised.
func (ram RAM) Read(address uint16) (uint8, error) {
	return ram.RAM[address^memorymap.OriginRAM], nil
}

// Write is an implementatio of memory.ChipBus. Address must be normalised.
func (ram *RAM) Write(address uint16, data uint8) error {
	ram.RAM[address^memorymap.OriginRAM] = data
	return nil
}
