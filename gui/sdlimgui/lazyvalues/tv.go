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
	val *LazyValues

	spec       atomic.Value // television.Specification
	autoSpec   atomic.Value // bool
	tvStr      atomic.Value // string
	lastSignal atomic.Value // television.SignalAttributes
	frame      atomic.Value // int
	scanline   atomic.Value // int
	hP         atomic.Value // int
	isStable   atomic.Value // float32
	actualFPS  atomic.Value // float32
	reqFPS     atomic.Value // float32

	Spec       television.Specification
	AutoSpec   bool
	TVstr      string
	LastSignal television.SignalAttributes
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
	spec, auto := lz.val.Dbg.VCS.TV.GetSpec()
	lz.spec.Store(*spec)
	lz.autoSpec.Store(auto)
	lz.tvStr.Store(lz.val.Dbg.VCS.TV.String())
	lz.lastSignal.Store(lz.val.Dbg.VCS.TV.GetLastSignal())

	frame, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqFramenum)
	lz.frame.Store(frame)

	scanline, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqScanline)
	lz.scanline.Store(scanline)

	hp, _ := lz.val.Dbg.VCS.TV.GetState(television.ReqHorizPos)
	lz.hP.Store(hp)

	lz.isStable.Store(lz.val.Dbg.VCS.TV.IsStable())
	lz.actualFPS.Store(lz.val.Dbg.VCS.TV.GetActualFPS())

	// note that the requested fps value is taken from the debugger and not the TV interface
	lz.reqFPS.Store(lz.val.Dbg.GetReqFPS())
}

func (lz *LazyTV) update() {
	lz.Spec, _ = lz.spec.Load().(television.Specification)
	lz.AutoSpec, _ = lz.autoSpec.Load().(bool)
	lz.TVstr, _ = lz.tvStr.Load().(string)
	lz.LastSignal, _ = lz.lastSignal.Load().(television.SignalAttributes)
	lz.Frame, _ = lz.frame.Load().(int)
	lz.Scanline, _ = lz.scanline.Load().(int)
	lz.HP, _ = lz.hP.Load().(int)
	lz.IsStable, _ = lz.isStable.Load().(bool)
	lz.AcutalFPS, _ = lz.actualFPS.Load().(float32)
	lz.ReqFPS, _ = lz.reqFPS.Load().(float32)
}
