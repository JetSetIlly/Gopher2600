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

	"github.com/jetsetilly/gopher2600/logger"
)

// LazyLog lazily accesses chip registere information from the emulator.
type LazyLog struct {
	val *LazyValues

	log atomic.Value // []logger.Entry
	Log []logger.Entry

	// have log contents changed
	dirty atomic.Value // bool
	Dirty bool

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
			lz.log.Store(l)
		}
	} else {
		lz.dirty.Store(false)
	}
}

func (lz *LazyLog) update() {
	if l, ok := lz.log.Load().([]logger.Entry); ok {
		lz.Log = l
		lz.Dirty, _ = lz.dirty.Load().(bool)
	}
}
