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

	"github.com/jetsetilly/gopher2600/tracker"
)

// LazyTracker lazily accesses logging entries.
type LazyTracker struct {
	val *LazyValues

	entries atomic.Value // []tracker.Entry
	Entries []tracker.Entry
}

func newLazyTracker(val *LazyValues) *LazyTracker {
	return &LazyTracker{val: val}
}

func (lz *LazyTracker) push() {
	lz.entries.Store(lz.val.dbg.Tracker.Copy())
}

func (lz *LazyTracker) update() {
	lz.Entries = lz.entries.Load().([]tracker.Entry)
}
