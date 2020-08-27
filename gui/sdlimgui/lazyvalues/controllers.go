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

	"github.com/jetsetilly/gopher2600/hardware/riot/ports/controllers"
)

// LazyControllers lazily accesses controller information from the emulator
type LazyControllers struct {
	val *Lazy

	atomicHandController0 atomic.Value // input.HandController
	atomicHandController1 atomic.Value // input.HandController
	HandController0       *controllers.Multi
	HandController1       *controllers.Multi
}

func newLazyControllers(val *Lazy) *LazyControllers {
	return &LazyControllers{val: val}
}

func (lz *LazyControllers) update() {
	lz.val.Dbg.PushRawEvent(func() {
		if p, ok := lz.val.Dbg.VCS.RIOT.Ports.Player0.(*controllers.Multi); ok {
			lz.atomicHandController0.Store(p)
		}
		if p, ok := lz.val.Dbg.VCS.RIOT.Ports.Player1.(*controllers.Multi); ok {
			lz.atomicHandController1.Store(p)
		}
	})
	lz.HandController0, _ = lz.atomicHandController0.Load().(*controllers.Multi)
	lz.HandController1, _ = lz.atomicHandController1.Load().(*controllers.Multi)
}
