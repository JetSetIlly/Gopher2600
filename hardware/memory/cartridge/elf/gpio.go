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
	ADDR_IDR    = 0x10
	DATA_IDR    = 0x20
	DATA_ODR    = 0x30
	DATA_MODER  = 0x40
	GPIO_MEMTOP = 0x50
)

type gpio struct {
	data       []byte
	dataOrigin uint32
	dataMemtop uint32

	lookup       []byte
	lookupOrigin uint32
	lookupMemtop uint32
}

// Snapshot implements the mapper.CartMapper interface.
func (g *gpio) Snapshot() *gpio {
	n := *g
	n.data = make([]byte, GPIO_MEMTOP)
	n.lookup = make([]byte, GPIO_MEMTOP)
	copy(n.data, g.data)
	copy(n.lookup, g.lookup)
	return &n
}

func newGPIO() *gpio {
	g := gpio{
		data:       make([]byte, GPIO_MEMTOP),
		dataOrigin: 0x40000000,
		dataMemtop: 0x40000000 | GPIO_MEMTOP,

		lookup:       make([]byte, GPIO_MEMTOP),
		lookupOrigin: 0x40020000,
		lookupMemtop: 0x40020000 | GPIO_MEMTOP,
	}

	offset := ADDR_IDR
	val := g.dataOrigin | ADDR_IDR
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	offset = DATA_IDR
	val = g.dataOrigin | DATA_IDR
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	offset = DATA_ODR
	val = g.dataOrigin | DATA_ODR
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	offset = DATA_MODER
	val = g.dataOrigin | DATA_MODER
	g.lookup[offset] = uint8(val)
	g.lookup[offset+1] = uint8(val >> 8)
	g.lookup[offset+2] = uint8(val >> 16)
	g.lookup[offset+3] = uint8(val >> 24)

	// default NOP instruction for opcode
	g.data[DATA_ODR] = 0xea

	return &g
}
