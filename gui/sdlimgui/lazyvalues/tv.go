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

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// LazyTV lazily accesses tv information from the emulator.
type LazyTV struct {
	val *LazyValues

	spec       atomic.Value // television.Spec
	tvStr      atomic.Value // string
	lastSignal atomic.Value // television.SignalAttributes
	frame      atomic.Value // int
	scanline   atomic.Value // int
	hP         atomic.Value // int
	isStable   atomic.Value // float32
	actualFPS  atomic.Value // float32
	reqFPS     atomic.Value // float32

	Spec       specification.Spec
	TVstr      string
	LastSignal signal.SignalAttributes
	Frame      int
	Scanline   int
	HP         int
	IsStable   bool
	AcutalFPS  float32
	ReqFPS     float32
}

func newLazyTV(val *LazyValues) *LazyTV {
	return &LazyTV{val: val}
}

func (lz *LazyTV) push() {
	lz.spec.Store(lz.val.Dbg.VCS.TV.GetSpec())
	lz.tvStr.Store(lz.val.Dbg.VCS.TV.String())
	lz.lastSignal.Store(lz.val.Dbg.VCS.TV.GetLastSignal())

	frame := lz.val.Dbg.VCS.TV.GetState(signal.ReqFramenum)
	lz.frame.Store(frame)

	scanline := lz.val.Dbg.VCS.TV.GetState(signal.ReqScanline)
	lz.scanline.Store(scanline)

	hp := lz.val.Dbg.VCS.TV.GetState(signal.ReqHorizPos)
	lz.hP.Store(hp)

	lz.isStable.Store(lz.val.Dbg.VCS.TV.IsStable())
	lz.actualFPS.Store(lz.val.Dbg.VCS.TV.GetActualFPS())

	lz.reqFPS.Store(lz.val.Dbg.VCS.TV.GetReqFPS())
}

func (lz *LazyTV) update() {
	lz.Spec, _ = lz.spec.Load().(specification.Spec)
	lz.TVstr, _ = lz.tvStr.Load().(string)
	lz.LastSignal, _ = lz.lastSignal.Load().(signal.SignalAttributes)
	lz.Frame, _ = lz.frame.Load().(int)
	lz.Scanline, _ = lz.scanline.Load().(int)
	lz.HP, _ = lz.hP.Load().(int)
	lz.IsStable, _ = lz.isStable.Load().(bool)
	lz.AcutalFPS, _ = lz.actualFPS.Load().(float32)
	lz.ReqFPS, _ = lz.reqFPS.Load().(float32)
}
