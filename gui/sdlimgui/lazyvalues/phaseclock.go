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

	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
)

// LazyPhaseClock lazily accesses PhaseClock information from the emulator.
type LazyPhaseClock struct {
	val *LazyValues

	lastPClk atomic.Value // phaseclock.PhaseClock
	LastPClk phaseclock.PhaseClock
}

func newLazyPhaseClock(val *LazyValues) *LazyPhaseClock {
	return &LazyPhaseClock{val: val}
}

func (lz *LazyPhaseClock) push() {
	lz.lastPClk.Store(lz.val.vcs.TIA.PClk)
}

func (lz *LazyPhaseClock) update() {
	lz.LastPClk, _ = lz.lastPClk.Load().(phaseclock.PhaseClock)
}
