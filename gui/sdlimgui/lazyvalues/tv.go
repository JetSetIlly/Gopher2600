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
	atomicTVStr     atomic.Value // string
	atomicFrame     atomic.Value // int
	atomicScanline  atomic.Value // int
	atomicHP        atomic.Value // int
	atomicReqFPS    atomic.Value // float32
	atomicActualFPS atomic.Value // float32

	Spec      television.Specification
	TVstr     string
	Frame     int
	Scanline  int
	HP        int
	AcutalFPS float32

	// taken from debugger rather than tv
	ReqFPS float32
}

func newLazyTV(val *Values) *LazyTV {
	return &LazyTV{val: val}
}

func (lz *LazyTV) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicSpec.Store(*lz.val.VCS.TV.GetSpec())
		lz.atomicTVStr.Store(lz.val.VCS.TV.String())

		frame, _ := lz.val.VCS.TV.GetState(television.ReqFramenum)
		lz.atomicFrame.Store(frame)
		scanline, _ := lz.val.VCS.TV.GetState(television.ReqScanline)
		lz.atomicScanline.Store(scanline)
		hp, _ := lz.val.VCS.TV.GetState(television.ReqHorizPos)
		lz.atomicHP.Store(hp)

		lz.atomicReqFPS.Store(lz.val.Dbg.GetReqFPS())
		lz.atomicActualFPS.Store(lz.val.VCS.TV.GetActualFPS())
	})
	lz.Spec, _ = lz.atomicSpec.Load().(television.Specification)
	lz.TVstr, _ = lz.atomicTVStr.Load().(string)
	lz.Frame, _ = lz.atomicFrame.Load().(int)
	lz.Scanline, _ = lz.atomicScanline.Load().(int)
	lz.HP, _ = lz.atomicHP.Load().(int)
	lz.ReqFPS, _ = lz.atomicReqFPS.Load().(float32)
	lz.AcutalFPS, _ = lz.atomicActualFPS.Load().(float32)
}
