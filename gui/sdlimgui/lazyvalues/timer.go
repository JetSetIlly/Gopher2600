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

import "sync/atomic"

// LazyTimer lazily accesses RIOT timer information from the emulator.
type LazyTimer struct {
	val *Values

	atomicRequested      atomic.Value // string
	Requested            string
	atomicINTIMvalue     atomic.Value // uint8
	INTIMvalue           uint8
	atomicTicksRemaining atomic.Value // uint16
	TicksRemaining       uint16
}

func newLazyTimer(val *Values) *LazyTimer {
	return &LazyTimer{val: val}
}

func (lz *LazyTimer) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicRequested.Store(lz.val.VCS.RIOT.Timer.Requested.String())
		lz.atomicINTIMvalue.Store(lz.val.VCS.RIOT.Timer.INTIMvalue)
		lz.atomicTicksRemaining.Store(lz.val.VCS.RIOT.Timer.TicksRemaining)
	})
	lz.Requested, _ = lz.atomicRequested.Load().(string)
	lz.INTIMvalue, _ = lz.atomicINTIMvalue.Load().(uint8)
	lz.TicksRemaining, _ = lz.atomicTicksRemaining.Load().(uint16)
}
