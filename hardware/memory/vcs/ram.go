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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// RAM represents the 128bytes of RAM in the PIA 6532 chip, found in the Atari VCS
type RAM struct {
	env *environment.Environment
	RAM []uint8
}

// NewRAM is the preferred method of initialisation for the RAM memory area
func NewRAM(env *environment.Environment) *RAM {
	ram := &RAM{
		env: env,
		RAM: make([]uint8, memorymap.MemtopRAM-memorymap.OriginRAM+1),
	}
	return ram
}

// Snapshot creates a copy of RAM in its current state
func (ram *RAM) Snapshot() *RAM {
	n := *ram
	n.RAM = make([]uint8, len(ram.RAM))
	copy(n.RAM, ram.RAM)
	return &n
}

// Reset contents of RAM
func (ram *RAM) Reset() {
	for i := range ram.RAM {
		if ram.env != nil && ram.env.Prefs.RandomState.Get().(bool) {
			ram.RAM[i] = uint8(ram.env.Random.NoRewind(0xff))
		} else {
			ram.RAM[i] = 0
		}
	}
}

func (ram *RAM) String() string {
	return hex.Dump(ram.RAM)
}

// Peek is an implementation of the memory.Area interface
func (ram *RAM) Peek(address uint16) (uint8, error) {
	v, _, err := ram.Read(address)
	return v, err
}

// Poke is an implementation of the memory.Area interface
func (ram *RAM) Poke(address uint16, value uint8) error {
	return ram.Write(address, value)
}

// Read is an implementation of the memory.Area interface
func (ram *RAM) Read(address uint16) (uint8, uint8, error) {
	return ram.RAM[address^memorymap.OriginRAM], 0xff, nil
}

// Write is an implementation of the memory.Area interface
func (ram *RAM) Write(address uint16, data uint8) error {
	ram.RAM[address^memorymap.OriginRAM] = data
	return nil
}
