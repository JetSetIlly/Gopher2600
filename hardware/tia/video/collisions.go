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

package video

import (
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// Collisions registers do not use all their bits only the top two bits, or in
// the case of CXBLPF the top bit only.
const (
	CollisionMask       uint8 = 0xc0
	CollisionCXBLPFMask uint8 = 0x80
)

type Collisions struct {
	mem bus.ChipBus

	// top two bits are significant excepty where noted
	CXM0P  uint8
	CXM1P  uint8
	CXP0FB uint8
	CXP1FB uint8
	CXM0FB uint8
	CXM1FB uint8
	CXBLPF uint8
	CXPPMM uint8

	// Active is set if there is any collision at all
	Activity strings.Builder
}

func newCollisions(mem bus.ChipBus) *Collisions {
	col := &Collisions{mem: mem}
	col.Clear()
	return col
}

// Snapshot creates a copy of the Collisions sub-system in its current state.
func (col *Collisions) Snapshot() *Collisions {
	n := *col
	return &n
}

func (col *Collisions) Plumb(mem bus.ChipBus) {
	col.mem = mem
}

// Clear all bits in the collision registers.
func (col *Collisions) Clear() {
	col.CXM0P = 0
	col.CXM1P = 0
	col.CXP0FB = 0
	col.CXP1FB = 0
	col.CXM0FB = 0
	col.CXM1FB = 0
	col.CXBLPF = 0
	col.CXPPMM = 0
	col.mem.ChipWrite(addresses.CXM0P, col.CXM0P)
	col.mem.ChipWrite(addresses.CXM1P, col.CXM1P)
	col.mem.ChipWrite(addresses.CXP0FB, col.CXP0FB)
	col.mem.ChipWrite(addresses.CXP1FB, col.CXP1FB)
	col.mem.ChipWrite(addresses.CXM0FB, col.CXM0FB)
	col.mem.ChipWrite(addresses.CXM1FB, col.CXM1FB)
	col.mem.ChipWrite(addresses.CXBLPF, col.CXBLPF)
	col.mem.ChipWrite(addresses.CXPPMM, col.CXPPMM)
}

// optimised tick of collision registers. memory is only written to when
// necessary.
func (col *Collisions) tick(p0, p1, m0, m1, bl, pf bool) {
	col.Activity.Reset()

	if m0 {
		if p1 || p0 {
			if p1 {
				col.CXM0P |= 0x80
				col.Activity.WriteString("M0 ^ P1")
			}
			if p0 {
				col.CXM0P |= 0x40
				col.Activity.WriteString("M0 ^ P0")
			}
			col.mem.ChipWrite(addresses.CXM0P, col.CXM0P)
		}

		if pf || bl {
			if pf {
				col.CXM0FB |= 0x80
				col.Activity.WriteString("M0 ^ PF")
			}
			if bl {
				col.CXM0FB |= 0x40
				col.Activity.WriteString("M1 ^ BL")
			}
			col.mem.ChipWrite(addresses.CXM0FB, col.CXM0FB)
		}
	}

	if m1 {
		if p1 || p0 {
			if p0 {
				col.CXM1P |= 0x80
				col.Activity.WriteString("M1 ^ P0")
			}
			if p1 {
				col.CXM1P |= 0x40
				col.Activity.WriteString("M1 ^ P1")
			}
			col.mem.ChipWrite(addresses.CXM1P, col.CXM1P)
		}

		if pf || bl {
			if pf {
				col.CXM1FB |= 0x80
				col.Activity.WriteString("M1 ^ PF")
			}
			if bl {
				col.CXM1FB |= 0x40
				col.Activity.WriteString("M1 ^ BL")
			}
			col.mem.ChipWrite(addresses.CXM1FB, col.CXM1FB)
		}
	}

	if p0 {
		if pf || bl {
			if pf {
				col.CXP0FB |= 0x80
				col.Activity.WriteString("P0 ^ PF")
			}
			if bl {
				col.CXP0FB |= 0x40
				col.Activity.WriteString("P0 ^ BL")
			}
			col.mem.ChipWrite(addresses.CXP0FB, col.CXP0FB)
		}
	}

	if p1 {
		if pf || bl {
			if pf {
				col.CXP1FB |= 0x80
				col.Activity.WriteString("P1 ^ PF")
			}
			if bl {
				col.CXP1FB |= 0x40
				col.Activity.WriteString("P1 ^ BL")
			}
			col.mem.ChipWrite(addresses.CXP1FB, col.CXP1FB)
		}
	}

	if bl && pf {
		col.CXBLPF |= 0x80
		col.Activity.WriteString("BL ^ PF")
		col.mem.ChipWrite(addresses.CXBLPF, col.CXBLPF)
	}
	// no bit 6 for CXBLPF

	if (p0 && p1) || (m0 && m1) {
		if p0 && p1 {
			col.CXPPMM |= 0x80
			col.Activity.WriteString("P0 ^ P1")
		}
		if m0 && m1 {
			col.CXPPMM |= 0x40
			col.Activity.WriteString("M0 ^ M1")
		}
		col.mem.ChipWrite(addresses.CXPPMM, col.CXPPMM)
	}
}

// this is a naive implementation of the collision registers checking. the
// version above is "optimised" but the reference implementation below is maybe
// easier to understand.
func (col *Collisions) tickReference(p0, p1, m0, m1, bl, pf bool) { // nolint: unused
	col.Activity.Reset()

	if m0 && p1 {
		col.CXM0P |= 0x80
		col.Activity.WriteString("M0 ^ P1")
	}
	if m0 && p0 {
		col.CXM0P |= 0x40
		col.Activity.WriteString("M0 ^ P0")
	}

	if m1 && p0 {
		col.CXM1P |= 0x80
		col.Activity.WriteString("M1 ^ P0")
	}
	if m1 && p1 {
		col.CXM1P |= 0x40
		col.Activity.WriteString("M1 ^ P1")
	}

	// use active bit when comparing with playfield
	if p0 && pf {
		col.CXP0FB |= 0x80
		col.Activity.WriteString("P0 ^ PF")
	}
	if p0 && bl {
		col.CXP0FB |= 0x40
		col.Activity.WriteString("P0 ^ BL")
	}

	// use active bit when comparing with playfield
	if p1 && pf {
		col.CXP1FB |= 0x80
		col.Activity.WriteString("P1 ^ PF")
	}
	if p1 && bl {
		col.CXP1FB |= 0x40
		col.Activity.WriteString("P1 ^ BL")
	}

	// use active bit when comparing with playfield
	if m0 && pf {
		col.CXM0FB |= 0x80
		col.Activity.WriteString("M0 ^ PF")
	}
	if m0 && bl {
		col.CXM0FB |= 0x40
		col.Activity.WriteString("M1 ^ BL")
	}

	// use active bit when comparing with playfield
	if m1 && pf {
		col.CXM1FB |= 0x80
		col.Activity.WriteString("M1 ^ PF")
	}
	if m1 && bl {
		col.CXM1FB |= 0x40
		col.Activity.WriteString("M1 ^ BL")
	}

	if bl && pf {
		col.CXBLPF |= 0x80
		col.Activity.WriteString("BL ^ PF")
	}
	// no bit 6 for CXBLPF

	if p0 && p1 {
		col.CXPPMM |= 0x80
		col.Activity.WriteString("P0 ^ P1")
	}
	if m0 && m1 {
		col.CXPPMM |= 0x40
		col.Activity.WriteString("M0 ^ M1")
	}

	col.mem.ChipWrite(addresses.CXM0P, col.CXM0P)
	col.mem.ChipWrite(addresses.CXM1P, col.CXM1P)
	col.mem.ChipWrite(addresses.CXP0FB, col.CXP0FB)
	col.mem.ChipWrite(addresses.CXP1FB, col.CXP1FB)
	col.mem.ChipWrite(addresses.CXM0FB, col.CXM0FB)
	col.mem.ChipWrite(addresses.CXM1FB, col.CXM1FB)
	col.mem.ChipWrite(addresses.CXBLPF, col.CXBLPF)
	col.mem.ChipWrite(addresses.CXPPMM, col.CXPPMM)
}
