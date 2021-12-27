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

	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

// LazyTimer lazily accesses RIOT timer information from the emulator.
type LazyTimer struct {
	val *LazyValues

	divider        atomic.Value // timer.Divider
	intim          atomic.Value // uint8
	ticksRemaining atomic.Value // int
	timint         atomic.Value // uint8

	Divider        timer.Divider
	INTIM          uint8
	TicksRemaining int
	TIMINT         uint8
}

func newLazyTimer(val *LazyValues) *LazyTimer {
	return &LazyTimer{val: val}
}

func (lz *LazyTimer) push() {
	lz.divider.Store(lz.val.vcs.RIOT.Timer.PeekField("divider"))
	lz.intim.Store(lz.val.vcs.RIOT.Timer.PeekField("intim"))
	lz.ticksRemaining.Store(lz.val.vcs.RIOT.Timer.PeekField("ticksRemaining"))
	lz.timint.Store(lz.val.vcs.RIOT.Timer.PeekField("timint"))
}

func (lz *LazyTimer) update() {
	lz.Divider, _ = lz.divider.Load().(timer.Divider)
	lz.INTIM, _ = lz.intim.Load().(uint8)
	lz.TicksRemaining, _ = lz.ticksRemaining.Load().(int)
	lz.TIMINT, _ = lz.timint.Load().(uint8)
}
