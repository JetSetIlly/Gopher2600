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

import "sync/atomic"

// LazyTimer lazily accesses RIOT timer information from the emulator
type LazyTimer struct {
	val *Lazy

	atomicDivider        atomic.Value // string
	atomicINTIMvalue     atomic.Value // uint8
	atomicTicksRemaining atomic.Value // int

	Divider        string
	INTIMvalue     uint8
	TicksRemaining int
}

func newLazyTimer(val *Lazy) *LazyTimer {
	return &LazyTimer{val: val}
}

func (lz *LazyTimer) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicDivider.Store(lz.val.Dbg.VCS.RIOT.Timer.Divider.String())
		lz.atomicINTIMvalue.Store(lz.val.Dbg.VCS.RIOT.Timer.INTIMvalue)
		lz.atomicTicksRemaining.Store(lz.val.Dbg.VCS.RIOT.Timer.TicksRemaining)
	})
	lz.Divider, _ = lz.atomicDivider.Load().(string)
	lz.INTIMvalue, _ = lz.atomicINTIMvalue.Load().(uint8)
	lz.TicksRemaining, _ = lz.atomicTicksRemaining.Load().(int)
}
