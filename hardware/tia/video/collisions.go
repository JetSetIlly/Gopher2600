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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package video

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

type Collisions struct {
	mem bus.ChipBus

	CXM0P  uint8
	CXM1P  uint8
	CXP0FB uint8
	CXP1FB uint8
	CXM0FB uint8
	CXM1FB uint8
	CXBLPF uint8
	CXPPMM uint8

	// Active is set if there is any collision at all
	Active bool
}

func newCollisions(mem bus.ChipBus) *Collisions {
	col := &Collisions{mem: mem}
	col.clear()
	return col
}

func (col *Collisions) clear() {
	col.CXM0P = 0
	col.CXM1P = 0
	col.CXP0FB = 0
	col.CXP1FB = 0
	col.CXM0FB = 0
	col.CXM1FB = 0
	col.CXBLPF = 0
	col.CXPPMM = 0
	col.mem.ChipWrite(addresses.CXM0P, 0)
	col.mem.ChipWrite(addresses.CXM1P, 0)
	col.mem.ChipWrite(addresses.CXP0FB, 0)
	col.mem.ChipWrite(addresses.CXP1FB, 0)
	col.mem.ChipWrite(addresses.CXM0FB, 0)
	col.mem.ChipWrite(addresses.CXM1FB, 0)
	col.mem.ChipWrite(addresses.CXBLPF, 0)
	col.mem.ChipWrite(addresses.CXPPMM, 0)
}

func (col *Collisions) setMemory(collisionRegister addresses.ChipRegister) {
	switch collisionRegister {
	case addresses.CXM0P:
		col.mem.ChipWrite(addresses.CXM0P, col.CXM0P)
	case addresses.CXM1P:
		col.mem.ChipWrite(addresses.CXM1P, col.CXM1P)
	case addresses.CXP0FB:
		col.mem.ChipWrite(addresses.CXP0FB, col.CXP0FB)
	case addresses.CXP1FB:
		col.mem.ChipWrite(addresses.CXP1FB, col.CXP1FB)
	case addresses.CXM0FB:
		col.mem.ChipWrite(addresses.CXM0FB, col.CXM0FB)
	case addresses.CXM1FB:
		col.mem.ChipWrite(addresses.CXM1FB, col.CXM1FB)
	case addresses.CXBLPF:
		col.mem.ChipWrite(addresses.CXBLPF, col.CXBLPF)
	case addresses.CXPPMM:
		col.mem.ChipWrite(addresses.CXPPMM, col.CXPPMM)
	default:
		// it would be nice to get rid of this panic() but it's doing no harm
		// and returning an error from here would be ugly.
		//
		// Best solution is to constrain collisioin registers by type...
		//
		// !!TODO: collisiion register type (subset of ChipRegister type)
		panic(fmt.Sprintf("not a collision register (%02x)", collisionRegister))
	}
}
