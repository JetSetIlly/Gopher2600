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

// LazyRewind lazily accesses VCS rewind information.
type LazyRewind struct {
	val *LazyValues

	numStates atomic.Value // int
	currState atomic.Value // int

	NumStates int
	CurrState int
}

func newLazyRewind(val *LazyValues) *LazyRewind {
	return &LazyRewind{val: val}
}

func (lz *LazyRewind) push() {
	n, c := lz.val.Dbg.VCS.Rewind.State()
	lz.numStates.Store(n)
	lz.currState.Store(c)
}

func (lz *LazyRewind) update() {
	lz.NumStates, _ = lz.numStates.Load().(int)
	lz.CurrState, _ = lz.currState.Load().(int)
}
