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

	"github.com/jetsetilly/gopher2600/rewind"
)

// LazyRewind lazily accesses VCS rewind information.
type LazyRewind struct {
	val *LazyValues

	timeline   atomic.Value // rewind.Timeline
	comparison atomic.Value // *rewind.Snapshot

	Timeline   rewind.Timeline
	Comparison *rewind.State
}

func newLazyRewind(val *LazyValues) *LazyRewind {
	return &LazyRewind{
		val: val,
	}
}

func (lz *LazyRewind) push() {
	lz.timeline.Store(lz.val.dbg.Rewind.GetTimeline())
	lz.comparison.Store(lz.val.dbg.Rewind.GetComparison())
}

func (lz *LazyRewind) update() {
	lz.Timeline, _ = lz.timeline.Load().(rewind.Timeline)
	lz.Comparison, _ = lz.comparison.Load().(*rewind.State)
}
