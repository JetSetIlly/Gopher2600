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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// LazyControllers lazily accesses controller information from the emulator.
type LazyControllers struct {
	val *LazyValues

	// unlike the other lazy types we can't use atomic values here because the
	// underlying type of the ports.Periperhal interface might change and
	// atomic.Value can only store consistently typed values
	player0 chan ports.Peripheral
	player1 chan ports.Peripheral

	Player0 ports.Peripheral
	Player1 ports.Peripheral
}

func newLazyControllers(val *LazyValues) *LazyControllers {
	return &LazyControllers{
		val:     val,
		player0: make(chan ports.Peripheral, 1),
		player1: make(chan ports.Peripheral, 1),
	}
}

func (lz *LazyControllers) push() {
	select {
	case lz.player0 <- lz.val.Dbg.VCS.RIOT.Ports.Player0:
	case lz.player1 <- lz.val.Dbg.VCS.RIOT.Ports.Player1:
	default:
	}
}

func (lz *LazyControllers) update() {
	select {
	case lz.Player0 = <-lz.player0:
	case lz.Player1 = <-lz.player1:
	default:
	}
}
