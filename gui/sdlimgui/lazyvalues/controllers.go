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

	"github.com/jetsetilly/gopher2600/hardware/riot/input"
)

// LazyControllers lazily accesses controller information from the emulator
type LazyControllers struct {
	val *Lazy

	atomicHandController0 atomic.Value // input.HandController
	atomicHandController1 atomic.Value // input.HandController
	HandController0       *input.HandController
	HandController1       *input.HandController
}

func newLazyControllers(val *Lazy) *LazyControllers {
	return &LazyControllers{val: val}
}

func (lz *LazyControllers) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicHandController0.Store(lz.val.Dbg.VCS.HandController0)
		lz.atomicHandController1.Store(lz.val.Dbg.VCS.HandController1)
	})
	lz.HandController0, _ = lz.atomicHandController0.Load().(*input.HandController)
	lz.HandController1, _ = lz.atomicHandController1.Load().(*input.HandController)
}
