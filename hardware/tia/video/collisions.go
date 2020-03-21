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

type collisions struct {
	mem bus.ChipBus

	cxm0p  uint8
	cxm1p  uint8
	cxp0fb uint8
	cxp1fb uint8
	cxm0fb uint8
	cxm1fb uint8
	cxblpf uint8
	cxppmm uint8
}

func newCollisions(mem bus.ChipBus) *collisions {
	col := &collisions{mem: mem}
	col.clear()
	return col
}

func (col *collisions) clear() {
	col.cxm0p = 0
	col.cxm1p = 0
	col.cxp0fb = 0
	col.cxp1fb = 0
	col.cxm0fb = 0
	col.cxm1fb = 0
	col.cxblpf = 0
	col.cxppmm = 0
	col.mem.ChipWrite(addresses.CXM0P, 0)
	col.mem.ChipWrite(addresses.CXM1P, 0)
	col.mem.ChipWrite(addresses.CXP0FB, 0)
	col.mem.ChipWrite(addresses.CXP1FB, 0)
	col.mem.ChipWrite(addresses.CXM0FB, 0)
	col.mem.ChipWrite(addresses.CXM1FB, 0)
	col.mem.ChipWrite(addresses.CXBLPF, 0)
	col.mem.ChipWrite(addresses.CXPPMM, 0)
}

func (col *collisions) setMemory(collisionRegister addresses.ChipRegister) {
	switch collisionRegister {
	case addresses.CXM0P:
		col.mem.ChipWrite(addresses.CXM0P, col.cxm0p)
	case addresses.CXM1P:
		col.mem.ChipWrite(addresses.CXM1P, col.cxm1p)
	case addresses.CXP0FB:
		col.mem.ChipWrite(addresses.CXP0FB, col.cxp0fb)
	case addresses.CXP1FB:
		col.mem.ChipWrite(addresses.CXP1FB, col.cxp1fb)
	case addresses.CXM0FB:
		col.mem.ChipWrite(addresses.CXM0FB, col.cxm0fb)
	case addresses.CXM1FB:
		col.mem.ChipWrite(addresses.CXM1FB, col.cxm1fb)
	case addresses.CXBLPF:
		col.mem.ChipWrite(addresses.CXBLPF, col.cxblpf)
	case addresses.CXPPMM:
		col.mem.ChipWrite(addresses.CXPPMM, col.cxppmm)
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
