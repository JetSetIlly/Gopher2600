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

	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// PeriphShim allows a possibly mutating interface to be stored in an atomic.Value.
type periphShim struct {
	periph ports.Peripheral
}

// LazyPeripherals lazily accesses controller information from the emulator.
type LazyPeripherals struct {
	val *LazyValues

	// unlike the other lazy types we can't use atomic values here because the
	// underlying type of the ports.Periperhal interface might change and
	// atomic.Value can only store consistently typed values
	left  atomic.Value // periphShim
	right atomic.Value // periphShim

	LeftPlayer  ports.Peripheral
	RightPlayer ports.Peripheral
}

func newLazyPeripherals(val *LazyValues) *LazyPeripherals {
	lz := &LazyPeripherals{
		val: val,
	}
	lz.left.Store(periphShim{})
	lz.right.Store(periphShim{})
	return lz
}

func (lz *LazyPeripherals) push() {
	lz.left.Store(periphShim{periph: lz.val.vcs.RIOT.Ports.LeftPlayer})
	lz.right.Store(periphShim{periph: lz.val.vcs.RIOT.Ports.RightPlayer})
}

func (lz *LazyPeripherals) update() {
	lz.LeftPlayer = lz.left.Load().(periphShim).periph
	lz.RightPlayer = lz.right.Load().(periphShim).periph
}
