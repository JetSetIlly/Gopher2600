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

const (
	gpio_mode      = 0x00 // gpioB
	toArm_address  = 0x10 // gpioA
	toArm_data     = 0x10 // gpioB
	fromArm_Opcode = 0x14 // gpioB
	gpio_memtop    = 0x18
)

type gpio struct {
	A       []byte
	AOrigin uint32
	AMemtop uint32

	B       []byte
	BOrigin uint32
	BMemtop uint32

	lookup       []byte
	lookupOrigin uint32
	lookupMemtop uint32
}

// Snapshot implements the mapper.CartMapper interface.
func (g *gpio) Snapshot() *gpio {
	n := *g
	n.A = make([]byte, gpio_memtop)
	n.B = make([]byte, gpio_memtop)
	n.lookup = make([]byte, gpio_memtop)
	copy(n.A, g.A)
	copy(n.B, g.B)
	copy(n.lookup, g.lookup)
	return &n
}

func newGPIO() *gpio {
	g := gpio{
		A:       make([]byte, gpio_memtop),
		AOrigin: 0x0000100,
		AMemtop: 0x0000100 | gpio_memtop,

		B:       make([]byte, gpio_memtop),
		BOrigin: 0x4000100,
		BMemtop: 0x4000100 | gpio_memtop,

		lookup:       make([]byte, gpio_memtop),
		lookupOrigin: 0x40000200,
		lookupMemtop: 0x40000200 | gpio_memtop,
	}

	offset := toArm_address
	val := g.AOrigin | toArm_address
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	offset = fromArm_Opcode
	val = g.BOrigin | fromArm_Opcode
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	// default NOP instruction for opcode
	g.B[fromArm_Opcode] = 0xea

	return &g
}
