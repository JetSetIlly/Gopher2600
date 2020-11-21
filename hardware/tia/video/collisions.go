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

	// collisions is a bit mask of the collision combination from
	// the last video cycle
	collisions uint16
}

// bits used in the collisions bit-mask. this has nothing to do with the VCS
// but it is the cheapest way of representing the combinations. the string
// value interprets these bits into a human-useful form.
const (
	M0P1 = 0b0000000000000001
	M0P0 = 0b0000000000000010
	M0PF = 0b0000000000000100
	M0BL = 0b0000000000001000
	M1P0 = 0b0000000000010000
	M1P1 = 0b0000000000100000
	M1PF = 0b0000000001000000
	M1BL = 0b0000000010000000
	P0PF = 0b0000000100000000
	P0BL = 0b0000001000000000
	P1PF = 0b0000010000000000
	P1BL = 0b0000100000000000
	BLPF = 0b0001000000000000
	P0P1 = 0b0010000000000000
	M0M1 = 0b0100000000000000
)

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

// String returns a string representation of all the collision bits from the
// last video cycle.
func (col *Collisions) String() string {
	s := strings.Builder{}

	if col.collisions&M0P1 == 0b0000000000000001 {
		s.WriteString("M0 ^ P1")
	}
	if col.collisions&M0P0 == 0b0000000000000010 {
		s.WriteString("M0 ^ P0")
	}
	if col.collisions&M0PF == 0b0000000000000100 {
		s.WriteString("M0 ^ PF")
	}
	if col.collisions&M0BL == 0b0000000000001000 {
		s.WriteString("M0 ^ BL")
	}
	if col.collisions&M1P0 == 0b0000000000010000 {
		s.WriteString("M1 ^ P0")
	}
	if col.collisions&M1P1 == 0b0000000000100000 {
		s.WriteString("M1 ^ P1")
	}
	if col.collisions&M1PF == 0b0000000001000000 {
		s.WriteString("M1 ^ PF")
	}
	if col.collisions&M1BL == 0b0000000010000000 {
		s.WriteString("M1 ^ BL")
	}
	if col.collisions&P0PF == 0b0000000100000000 {
		s.WriteString("P0 ^ PF")
	}
	if col.collisions&P0BL == 0b0000001000000000 {
		s.WriteString("P0 ^ BL")
	}
	if col.collisions&P1PF == 0b0000010000000000 {
		s.WriteString("P1 ^ PF")
	}
	if col.collisions&P1BL == 0b0000100000000000 {
		s.WriteString("P1 ^ BL")
	}
	if col.collisions&BLPF == 0b0001000000000000 {
		s.WriteString("BL ^ PF")
	}
	if col.collisions&P0P1 == 0b0010000000000000 {
		s.WriteString("P0 ^ P1")
	}
	if col.collisions&M0M1 == 0b0100000000000000 {
		s.WriteString("M0 ^ M1")
	}

	return s.String()
}

// optimised tick of collision registers. memory is only written to when
// necessary.
func (col *Collisions) tick(p0, p1, m0, m1, bl, pf bool) {
	col.collisions = 0

	if m0 {
		if p1 || p0 {
			if p1 {
				col.CXM0P |= 0x80
				col.collisions |= M0P1
			}
			if p0 {
				col.CXM0P |= 0x40
				col.collisions |= M0P0
			}
			col.mem.ChipWrite(addresses.CXM0P, col.CXM0P)
		}

		if pf || bl {
			if pf {
				col.CXM0FB |= 0x80
				col.collisions |= M0PF
			}
			if bl {
				col.CXM0FB |= 0x40
				col.collisions |= M0BL
			}
			col.mem.ChipWrite(addresses.CXM0FB, col.CXM0FB)
		}
	}

	if m1 {
		if p1 || p0 {
			if p0 {
				col.CXM1P |= 0x80
				col.collisions |= M1P0
			}
			if p1 {
				col.CXM1P |= 0x40
				col.collisions |= M1P1
			}
			col.mem.ChipWrite(addresses.CXM1P, col.CXM1P)
		}

		if pf || bl {
			if pf {
				col.CXM1FB |= 0x80
				col.collisions |= M1PF
			}
			if bl {
				col.CXM1FB |= 0x40
				col.collisions |= M1BL
			}
			col.mem.ChipWrite(addresses.CXM1FB, col.CXM1FB)
		}
	}

	if p0 {
		if pf || bl {
			if pf {
				col.CXP0FB |= 0x80
				col.collisions |= P0PF
			}
			if bl {
				col.CXP0FB |= 0x40
				col.collisions |= P0BL
			}
			col.mem.ChipWrite(addresses.CXP0FB, col.CXP0FB)
		}
	}

	if p1 {
		if pf || bl {
			if pf {
				col.CXP1FB |= 0x80
				col.collisions |= P1PF
			}
			if bl {
				col.CXP1FB |= 0x40
				col.collisions |= P1BL
			}
			col.mem.ChipWrite(addresses.CXP1FB, col.CXP1FB)
		}
	}

	if bl && pf {
		col.CXBLPF |= 0x80
		col.collisions |= BLPF
		col.mem.ChipWrite(addresses.CXBLPF, col.CXBLPF)
	}
	// no bit 6 for CXBLPF

	if (p0 && p1) || (m0 && m1) {
		if p0 && p1 {
			col.CXPPMM |= 0x80
			col.collisions |= P0P1
		}
		if m0 && m1 {
			col.CXPPMM |= 0x40
			col.collisions |= M0M1
		}
		col.mem.ChipWrite(addresses.CXPPMM, col.CXPPMM)
	}
}

// this is a naive implementation of the collision registers checking. the
// version above is "optimised" but the reference implementation below is maybe
// easier to understand.
func (col *Collisions) tickReference(p0, p1, m0, m1, bl, pf bool) { // nolint: unused
	col.collisions = 0

	if m0 && p1 {
		col.CXM0P |= 0x80
		col.collisions |= M0P1
	}
	if m0 && p0 {
		col.CXM0P |= 0x40
		col.collisions |= M0P0
	}

	if m1 && p0 {
		col.CXM1P |= 0x80
		col.collisions |= M1P0
	}
	if m1 && p1 {
		col.CXM1P |= 0x40
		col.collisions |= M1P1
	}

	// use active bit when comparing with playfield
	if p0 && pf {
		col.CXP0FB |= 0x80
		col.collisions |= P0PF
	}
	if p0 && bl {
		col.CXP0FB |= 0x40
		col.collisions |= P0BL
	}

	// use active bit when comparing with playfield
	if p1 && pf {
		col.CXP1FB |= 0x80
		col.collisions |= P1PF
	}
	if p1 && bl {
		col.CXP1FB |= 0x40
		col.collisions |= P1BL
	}

	// use active bit when comparing with playfield
	if m0 && pf {
		col.CXM0FB |= 0x80
		col.collisions |= M0PF
	}
	if m0 && bl {
		col.CXM0FB |= 0x40
		col.collisions |= M0BL
	}

	// use active bit when comparing with playfield
	if m1 && pf {
		col.CXM1FB |= 0x80
		col.collisions |= M1PF
	}
	if m1 && bl {
		col.CXM1FB |= 0x40
		col.collisions |= M1BL
	}

	if bl && pf {
		col.CXBLPF |= 0x80
		col.collisions |= BLPF
	}
	// no bit 6 for CXBLPF

	if p0 && p1 {
		col.CXPPMM |= 0x80
		col.collisions |= P0P1
	}
	if m0 && m1 {
		col.CXPPMM |= 0x40
		col.collisions |= M0M1
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
