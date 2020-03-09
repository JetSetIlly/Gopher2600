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

package lazyvalues

import (
	"gopher2600/television"
	"sync/atomic"
)

// LazyTV lazily accesses tv information from the emulator.
type LazyTV struct {
	val *Values

	atomicSpec      atomic.Value // television.Specification
	Spec            television.Specification
	atomicTVStr     atomic.Value // string
	TVstr           string
	atomicReqFPS    atomic.Value // float32
	ReqFPS          float32
	atomicActualFPS atomic.Value // float32
	AcutalFPS       float32
}

func newLazyTV(val *Values) *LazyTV {
	return &LazyTV{val: val}
}

func (lz *LazyTV) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicSpec.Store(*lz.val.VCS.TV.GetSpec())
		lz.atomicTVStr.Store(lz.val.VCS.TV.String())
		lz.atomicReqFPS.Store(lz.val.VCS.TV.GetReqFPS())
		lz.atomicActualFPS.Store(lz.val.VCS.TV.GetActualFPS())
	})
	lz.Spec, _ = lz.atomicSpec.Load().(television.Specification)
	lz.TVstr, _ = lz.atomicTVStr.Load().(string)
	lz.ReqFPS, _ = lz.atomicReqFPS.Load().(float32)
	lz.AcutalFPS, _ = lz.atomicActualFPS.Load().(float32)
}
