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

	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
)

// Collisions represents the various collision registers in the VCS.
type Collisions struct {
	mem chipbus.Memory

	// LastColorClock records the combination of collision bits for the most recent
	// video cycle. Facilitates production of string information.
	LastColorClock CollisionEvent
}

// CollisionEvent is an emulator specific value that records the collision
// events that occurred in the immediately preceding videocycle.
//
// The VCS doesn't care about this and the collision registers instead record
// all collisions since the last CXCLR, which can be many hundreds of
// videocycles later. For debugging purposes however, it can be quite useful to
// know what collisions occurred on a single videocycle one.
//
// The trick is to do this as efficiently as possible. Collision event is
// therefore a bitmask that is reset() every videocycle and the bit set for
// each collision that occurs during the collision tick().
//
// It seems clumsy and it probably is, but it's the most efficient way I can
// think of right now. Certainly, it postpones the interpretation of the event
// in the form of a String() to when it is actually needed.
//
// Note that multiple collisions can occur in a single videocycle. If this
// wasn't the case we could simplify the CollisionEvent type but as it is we
// need to cater for all circumstances.
type CollisionEvent uint16

// bitmasks for individual collision events. They are ORed together to record
// multiple collisions.
const (
	m0p1  = 0b0000000000000001
	m0p0  = 0b0000000000000010
	m0pf  = 0b0000000000000100
	m0bl  = 0b0000000000001000
	m1p0  = 0b0000000000010000
	m1p1  = 0b0000000000100000
	m1pf  = 0b0000000001000000
	m1bl  = 0b0000000010000000
	p0pf  = 0b0000000100000000
	p0bl  = 0b0000001000000000
	p1pf  = 0b0000010000000000
	p1bl  = 0b0000100000000000
	blpf  = 0b0001000000000000
	p0p1  = 0b0010000000000000
	m0m1  = 0b0100000000000000
	cxclr = 0b1000000000000000
)

// reset CollisionEvent. should be called every video cycle. see comment for
// Collisions.tick() function.
func (col *CollisionEvent) reset() {
	*col = 0
}

// IsNothing returns true if no new collision event occurred.
func (col CollisionEvent) IsNothing() bool {
	return col == 0x0000
}

// IsCleared returns true if CollisionEvent is CXCLR.
func (col CollisionEvent) IsCXCLR() bool {
	return col&cxclr == cxclr
}

// String returns a string representation of a CollisionEvent.
func (col CollisionEvent) String() string {
	if col&cxclr == cxclr {
		return "collisions cleared"
	}

	s := strings.Builder{}

	if col&m0p1 == m0p1 {
		s.WriteString("M0 and P1\n")
	}
	if col&m0p0 == m0p0 {
		s.WriteString("M0 and P0\n")
	}
	if col&m0pf == m0pf {
		s.WriteString("M0 and PF\n")
	}
	if col&m0bl == m0bl {
		s.WriteString("M0 and BL\n")
	}
	if col&m1p0 == m1p0 {
		s.WriteString("M1 and P0\n")
	}
	if col&m1p1 == m1p1 {
		s.WriteString("M1 and P1\n")
	}
	if col&m1pf == m1pf {
		s.WriteString("M1 and PF\n")
	}
	if col&m1bl == m1bl {
		s.WriteString("M1 and BL\n")
	}
	if col&p0pf == p0pf {
		s.WriteString("P0 and PF\n")
	}
	if col&p0bl == p0bl {
		s.WriteString("P0 and BL\n")
	}
	if col&p1pf == p1pf {
		s.WriteString("P1 and PF\n")
	}
	if col&p1bl == p1bl {
		s.WriteString("P1 and BL\n")
	}
	if col&blpf == blpf {
		s.WriteString("BL and PF\n")
	}
	if col&p0p1 == p0p1 {
		s.WriteString("P0 and P1\n")
	}
	if col&m0m1 == m0m1 {
		s.WriteString("M0 and M1\n")
	}

	return strings.TrimSuffix(s.String(), "\n")
}

func newCollisions(mem chipbus.Memory) *Collisions {
	col := &Collisions{mem: mem}
	col.Clear()
	return col
}

// Snapshot creates a copy of the Collisions sub-system in its current state.
func (col *Collisions) Snapshot() *Collisions {
	n := *col
	return &n
}

// Plumb a new ChipBus into the collision system.
func (col *Collisions) Plumb(mem chipbus.Memory) {
	col.mem = mem
}

// Clear all bits in the collision registers.
func (col *Collisions) Clear() {
	col.mem.ChipWrite(chipbus.CXM0P, 0x00)
	col.mem.ChipWrite(chipbus.CXM1P, 0x00)
	col.mem.ChipWrite(chipbus.CXP0FB, 0x00)
	col.mem.ChipWrite(chipbus.CXP1FB, 0x00)
	col.mem.ChipWrite(chipbus.CXM0FB, 0x00)
	col.mem.ChipWrite(chipbus.CXM1FB, 0x00)
	col.mem.ChipWrite(chipbus.CXBLPF, 0x00)
	col.mem.ChipWrite(chipbus.CXPPMM, 0x00)
	col.LastColorClock = cxclr
}

// optimised tick of collision registers. memory is only written to when necessary.
//
// if this function is not called during a video cycle (which is possible for
// reasons of optimisation) then the LastCoorClock value must be reset
// instead.
func (col *Collisions) tick(p0, p1, m0, m1, bl, pf bool) {
	col.LastColorClock.reset()

	if m0 {
		if p1 {
			v := col.mem.ChipRefer(chipbus.CXM0P)
			v |= 0x80
			col.LastColorClock |= m0p1
			col.mem.ChipWrite(chipbus.CXM0P, v)
		}
		if p0 {
			v := col.mem.ChipRefer(chipbus.CXM0P)
			v |= 0x40
			col.LastColorClock |= m0p0
			col.mem.ChipWrite(chipbus.CXM0P, v)
		}

		if pf {
			v := col.mem.ChipRefer(chipbus.CXM0FB)
			v |= 0x80
			col.LastColorClock |= m0pf
			col.mem.ChipWrite(chipbus.CXM0FB, v)
		}
		if bl {
			v := col.mem.ChipRefer(chipbus.CXM0FB)
			v |= 0x40
			col.LastColorClock |= m0bl
			col.mem.ChipWrite(chipbus.CXM0FB, v)
		}
	}

	if m1 {
		if p0 {
			v := col.mem.ChipRefer(chipbus.CXM1P)
			v |= 0x80
			col.LastColorClock |= m1p0
			col.mem.ChipWrite(chipbus.CXM1P, v)
		}
		if p1 {
			v := col.mem.ChipRefer(chipbus.CXM1P)
			v |= 0x40
			col.LastColorClock |= m1p1
			col.mem.ChipWrite(chipbus.CXM1P, v)
		}

		if pf {
			v := col.mem.ChipRefer(chipbus.CXM1FB)
			v |= 0x80
			col.LastColorClock |= m1pf
			col.mem.ChipWrite(chipbus.CXM1FB, v)
		}
		if bl {
			v := col.mem.ChipRefer(chipbus.CXM1FB)
			v |= 0x40
			col.LastColorClock |= m1bl
			col.mem.ChipWrite(chipbus.CXM1FB, v)
		}
	}

	if p0 {
		if pf {
			v := col.mem.ChipRefer(chipbus.CXP0FB)
			v |= 0x80
			col.LastColorClock |= p0pf
			col.mem.ChipWrite(chipbus.CXP0FB, v)
		}
		if bl {
			v := col.mem.ChipRefer(chipbus.CXP0FB)
			v |= 0x40
			col.LastColorClock |= p0bl
			col.mem.ChipWrite(chipbus.CXP0FB, v)
		}
	}

	if p1 {
		if pf {
			v := col.mem.ChipRefer(chipbus.CXP1FB)
			v |= 0x80
			col.LastColorClock |= p1pf
			col.mem.ChipWrite(chipbus.CXP1FB, v)
		}
		if bl {
			v := col.mem.ChipRefer(chipbus.CXP1FB)
			v |= 0x40
			col.LastColorClock |= p1bl
			col.mem.ChipWrite(chipbus.CXP1FB, v)
		}
	}

	if bl && pf {
		v := col.mem.ChipRefer(chipbus.CXBLPF)
		v |= 0x80
		col.LastColorClock |= blpf
		col.mem.ChipWrite(chipbus.CXBLPF, v)
	}
	// no bit 6 for CXBLPF

	if p0 && p1 {
		v := col.mem.ChipRefer(chipbus.CXPPMM)
		v |= 0x80
		col.LastColorClock |= p0p1
		col.mem.ChipWrite(chipbus.CXPPMM, v)
	}

	if m0 && m1 {
		v := col.mem.ChipRefer(chipbus.CXPPMM)
		v |= 0x40
		col.LastColorClock |= m0m1
		col.mem.ChipWrite(chipbus.CXPPMM, v)
	}
}
