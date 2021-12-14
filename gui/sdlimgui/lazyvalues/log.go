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
	"strings"
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/logger"
)

// LazyLog lazily accesses logging entries.
type LazyLog struct {
	val *LazyValues

	entries atomic.Value // []logger.Entry
	Entries []logger.Entry

	// have log contents changed
	dirty atomic.Value // bool
	Dirty bool

	// the number of lines in the log. not the same as the number of entries -
	// an entry might consist of many lines
	NumLines int

	// used to detect dirty logs
	timeoflast int
}

func newLazyLog(val *LazyValues) *LazyLog {
	return &LazyLog{val: val}
}

func (lz *LazyLog) push() {
	t := logger.TimeOfLast()
	if t != lz.timeoflast {
		lz.timeoflast = t
		lz.dirty.Store(true)
		if l := logger.Copy(); l != nil {
			lz.entries.Store(l)
		}
	}
}

func (lz *LazyLog) update() {
	lz.Dirty, _ = lz.dirty.Load().(bool)
	if lz.Dirty {
		lz.dirty.Store(false)
		if l, ok := lz.entries.Load().([]logger.Entry); ok {
			lz.Entries = l
		}

		// number of lines is equal to the number of entries plus the number of
		// \n characters (works because log strings do not end with a \n)
		lz.NumLines = 0
		for _, l := range lz.Entries {
			lz.NumLines += 1 + strings.Count(l.String(), "\n")
		}
	}
}
