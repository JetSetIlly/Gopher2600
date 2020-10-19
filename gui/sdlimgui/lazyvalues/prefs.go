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

// LazyPrefs lazily accesses the debugger/emulator's preference states.
type LazyPrefs struct {
	val *LazyValues

	randomState atomic.Value // bool (from prefs.Bool.Get())
	randomPins  atomic.Value // bool (from prefs.Bool.Get())
	fxxxMirror  atomic.Value // bool (from prefs.Bool.Get())
	symbols     atomic.Value // bool (from prefs.Bool.Get())

	RandomState bool
	RandomPins  bool
	FxxxMirror  bool
	Symbols     bool
}

func newLazyPrefs(val *LazyValues) *LazyPrefs {
	lz := &LazyPrefs{val: val}
	return lz
}

func (lz *LazyPrefs) push() {
	lz.randomState.Store(lz.val.Dbg.VCS.Prefs.RandomState.Get())
	lz.randomPins.Store(lz.val.Dbg.VCS.Prefs.RandomPins.Get())
	lz.fxxxMirror.Store(lz.val.Dbg.Disasm.Prefs.FxxxMirror.Get())
	lz.symbols.Store(lz.val.Dbg.Disasm.Prefs.Symbols.Get())
}
func (lz *LazyPrefs) update() {
	lz.RandomState, _ = lz.randomState.Load().(bool)
	lz.RandomPins, _ = lz.randomPins.Load().(bool)
	lz.FxxxMirror, _ = lz.fxxxMirror.Load().(bool)
	lz.Symbols, _ = lz.symbols.Load().(bool)
}
