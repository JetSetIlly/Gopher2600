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
)

// LazyPrefs lazily accesses the debugger/emulator's preference states
type LazyPrefs struct {
	val *Lazy

	atomicRandomState atomic.Value // bool (from prefs.Bool.Get())
	atomicRandomPins  atomic.Value // bool (from prefs.Bool.Get())
	atomicFxxxMirror  atomic.Value // bool (from prefs.Bool.Get())
	atomicSymbols     atomic.Value // bool (from prefs.Bool.Get())

	RandomState bool
	RandomPins  bool
	FxxxMirror  bool
	Symbols     bool
}

func newLazyPrefs(val *Lazy) *LazyPrefs {
	lz := &LazyPrefs{val: val}
	return lz
}

func (lz *LazyPrefs) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicRandomState.Store(lz.val.Dbg.Prefs.RandomState.Get())
		lz.atomicRandomPins.Store(lz.val.Dbg.Prefs.RandomPins.Get())
		lz.atomicFxxxMirror.Store(lz.val.Dbg.Disasm.Prefs.FxxxMirror.Get())
		lz.atomicSymbols.Store(lz.val.Dbg.Disasm.Prefs.Symbols.Get())
	})
	lz.RandomState, _ = lz.atomicRandomState.Load().(bool)
	lz.RandomPins, _ = lz.atomicRandomPins.Load().(bool)
	lz.FxxxMirror, _ = lz.atomicFxxxMirror.Load().(bool)
	lz.Symbols, _ = lz.atomicSymbols.Load().(bool)
}
