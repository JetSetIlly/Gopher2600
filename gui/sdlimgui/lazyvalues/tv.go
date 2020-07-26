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

	"github.com/jetsetilly/gopher2600/television"
)

// LazyTV lazily accesses tv information from the emulator
type LazyTV struct {
	val *Lazy

	atomicSpec       atomic.Value // television.Specification
	atomicAutoSpec   atomic.Value // bool
	atomicTVStr      atomic.Value // string
	atomicLastSignal atomic.Value // television.SignalAttributes
	atomicFrame      atomic.Value // int
	atomicScanline   atomic.Value // int
	atomicHP         atomic.Value // int
	atomicReqFPS     atomic.Value // float32
	atomicActualFPS  atomic.Value // float32
	atomicIsStable   atomic.Value // float32

	Spec       television.Specification
	AutoSpec   bool
	TVstr      string
	LastSignal television.SignalAttributes
	Frame      int
	Scanline   int
	HP         int
	AcutalFPS  float32
	IsStable   bool

	// taken from debugger rather than tv
	ReqFPS float32
}

func newLazyTV(val *Lazy) *LazyTV {
	return &LazyTV{val: val}
}

func (lz *LazyTV) update() {
	lz.val.Dbg.PushRawEvent(func() {
		spec, auto := lz.val.Dbg.VCS.TV.GetSpec()
		lz.atomicSpec.Store(*spec)
		lz.atomicAutoSpec.Store(auto)
		lz.atomicTVStr.Store(lz.val.Dbg.VCS.TV.String())
		lz.atomicLastSignal.Store(lz.val.Dbg.VCS.TV.GetLastSignal())

		frame, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqFramenum)
		lz.atomicFrame.Store(frame)

		scanline, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqScanline)
		lz.atomicScanline.Store(scanline)

		hp, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqHorizPos)
		lz.atomicHP.Store(hp)

		lz.atomicReqFPS.Store(lz.val.Dbg.GetReqFPS())
		lz.atomicActualFPS.Store(lz.val.Dbg.VCS.TV.GetActualFPS())
		lz.atomicIsStable.Store(lz.val.Dbg.VCS.TV.IsStable())

	})
	lz.Spec, _ = lz.atomicSpec.Load().(television.Specification)
	lz.AutoSpec, _ = lz.atomicAutoSpec.Load().(bool)
	lz.TVstr, _ = lz.atomicTVStr.Load().(string)
	lz.LastSignal, _ = lz.atomicLastSignal.Load().(television.SignalAttributes)
	lz.Frame, _ = lz.atomicFrame.Load().(int)
	lz.Scanline, _ = lz.atomicScanline.Load().(int)
	lz.HP, _ = lz.atomicHP.Load().(int)
	lz.ReqFPS, _ = lz.atomicReqFPS.Load().(float32)
	lz.AcutalFPS, _ = lz.atomicActualFPS.Load().(float32)
	lz.IsStable, _ = lz.atomicIsStable.Load().(bool)
}
