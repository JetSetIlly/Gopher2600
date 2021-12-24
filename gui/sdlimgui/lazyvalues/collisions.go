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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// LazyTimer lazily accesses RIOT timer information from the emulator.
type LazyCollisions struct {
	val *LazyValues

	cxm0p  atomic.Value // uint8
	cxm1p  atomic.Value // uint8
	cxp0fb atomic.Value // uint8
	cxp1fb atomic.Value // uint8
	cxm0fb atomic.Value // uint8
	cxm1fb atomic.Value // uint8
	cxblpf atomic.Value // uint8
	cxppmm atomic.Value // uint8

	CXM0P  uint8
	CXM1P  uint8
	CXP0FB uint8
	CXP1FB uint8
	CXM0FB uint8
	CXM1FB uint8
	CXBLPF uint8
	CXPPMM uint8
}

func newLazyCollisions(val *LazyValues) *LazyCollisions {
	return &LazyCollisions{val: val}
}

func (lz *LazyCollisions) push() {
	v, _ := lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXM0P"])
	lz.cxm0p.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXM1P"])
	lz.cxm1p.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXP0FB"])
	lz.cxp0fb.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXP1FB"])
	lz.cxp1fb.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXM0FB"])
	lz.cxm0fb.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXM1FB"])
	lz.cxm1fb.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXBLPF"])
	lz.cxblpf.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["CXPPMM"])
	lz.cxppmm.Store(v)
}

func (lz *LazyCollisions) update() {
	lz.CXM0P, _ = lz.cxm0p.Load().(uint8)
	lz.CXM1P, _ = lz.cxm1p.Load().(uint8)
	lz.CXP0FB, _ = lz.cxp0fb.Load().(uint8)
	lz.CXP1FB, _ = lz.cxp1fb.Load().(uint8)
	lz.CXM0FB, _ = lz.cxm0fb.Load().(uint8)
	lz.CXM1FB, _ = lz.cxm1fb.Load().(uint8)
	lz.CXBLPF, _ = lz.cxblpf.Load().(uint8)
	lz.CXPPMM, _ = lz.cxppmm.Load().(uint8)
}
