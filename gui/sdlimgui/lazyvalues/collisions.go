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
)

// LazyTimer lazily accesses RIOT timer information from the emulator
type LazyCollisions struct {
	val *Lazy

	atomicCXM0P  atomic.Value // uint8
	atomicCXM1P  atomic.Value // uint8
	atomicCXP0FB atomic.Value // uint8
	atomicCXP1FB atomic.Value // uint8
	atomicCXM0FB atomic.Value // uint8
	atomicCXM1FB atomic.Value // uint8
	atomicCXBLPF atomic.Value // uint8
	atomicCXPPMM atomic.Value // uint8

	CXM0P  uint8
	CXM1P  uint8
	CXP0FB uint8
	CXP1FB uint8
	CXM0FB uint8
	CXM1FB uint8
	CXBLPF uint8
	CXPPMM uint8
}

func newLazyCollisions(val *Lazy) *LazyCollisions {
	return &LazyCollisions{val: val}
}

func (lz *LazyCollisions) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicCXM0P.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXM0P)
		lz.atomicCXM1P.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXM1P)
		lz.atomicCXP0FB.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXP0FB)
		lz.atomicCXP1FB.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXP1FB)
		lz.atomicCXM0FB.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXM0FB)
		lz.atomicCXM1FB.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXM1FB)
		lz.atomicCXBLPF.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXBLPF)
		lz.atomicCXPPMM.Store(lz.val.Dbg.VCS.TIA.Video.Collisions.CXPPMM)
	})
	lz.CXM0P, _ = lz.atomicCXM0P.Load().(uint8)
	lz.CXM1P, _ = lz.atomicCXM1P.Load().(uint8)
	lz.CXP0FB, _ = lz.atomicCXP0FB.Load().(uint8)
	lz.CXP1FB, _ = lz.atomicCXP1FB.Load().(uint8)
	lz.CXM0FB, _ = lz.atomicCXM0FB.Load().(uint8)
	lz.CXM1FB, _ = lz.atomicCXM1FB.Load().(uint8)
	lz.CXBLPF, _ = lz.atomicCXBLPF.Load().(uint8)
	lz.CXPPMM, _ = lz.atomicCXPPMM.Load().(uint8)
}
