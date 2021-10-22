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

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// LazyTV lazily accesses tv information from the emulator.
type LazyTV struct {
	val *LazyValues

	frameInfo  atomic.Value // television.Actual
	tvStr      atomic.Value // string
	lastSignal atomic.Value // signal.SignalAttributes
	coords     atomic.Value // coords.TelevisionCoords
	hz         atomic.Value // float32
	actualFPS  atomic.Value // float32
	reqFPS     atomic.Value // float32

	FrameInfo  television.FrameInfo
	TVstr      string
	LastSignal signal.SignalAttributes
	Coords     coords.TelevisionCoords
	Hz         float32
	ActualFPS  float32
	ReqFPS     float32
}

func newLazyTV(val *LazyValues) *LazyTV {
	return &LazyTV{val: val}
}

func (lz *LazyTV) push() {
	lz.frameInfo.Store(lz.val.vcs.TV.GetFrameInfo())
	lz.tvStr.Store(lz.val.vcs.TV.String())
	lz.lastSignal.Store(lz.val.vcs.TV.GetLastSignal())

	coords := lz.val.vcs.TV.GetCoords()
	lz.coords.Store(coords)

	actual, hz := lz.val.vcs.TV.GetActualFPS()
	lz.hz.Store(hz)
	lz.actualFPS.Store(actual)

	lz.reqFPS.Store(lz.val.vcs.TV.GetReqFPS())
}

func (lz *LazyTV) update() {
	lz.FrameInfo, _ = lz.frameInfo.Load().(television.FrameInfo)
	lz.TVstr, _ = lz.tvStr.Load().(string)
	lz.LastSignal, _ = lz.lastSignal.Load().(signal.SignalAttributes)
	lz.Coords, _ = lz.coords.Load().(coords.TelevisionCoords)
	lz.Hz, _ = lz.hz.Load().(float32)
	lz.ActualFPS, _ = lz.actualFPS.Load().(float32)
	lz.ReqFPS, _ = lz.reqFPS.Load().(float32)
}
