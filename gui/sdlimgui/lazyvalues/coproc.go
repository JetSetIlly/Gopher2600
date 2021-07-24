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

// LazyCoProc lazily accesses coproceddor information from the emulator.
type LazyCoProc struct {
	val *LazyValues

	id           atomic.Value // string
	hasCoProcBus atomic.Value // bool

	ID           string
	HasCoProcBus bool
}

func newLazyCoProc(val *LazyValues) *LazyCoProc {
	lz := &LazyCoProc{val: val}
	lz.id.Store("")
	lz.hasCoProcBus.Store(false)
	return lz
}

func (lz *LazyCoProc) push() {
	cp := lz.val.vcs.Mem.Cart.GetCoProcBus()
	if cp != nil {
		lz.hasCoProcBus.Store(true)
		lz.id.Store(cp.CoProcID())
	} else {
		lz.hasCoProcBus.Store(false)
		lz.id.Store("")
	}
}

func (lz *LazyCoProc) update() {
	lz.ID = lz.id.Load().(string)
	lz.HasCoProcBus = lz.hasCoProcBus.Load().(bool)
}
