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

// LazyControllers lazily accesses controller information from the emulator
type LazyControllers struct {
	val *Lazy

	// we can't use atomic values here because the underlying type of the
	// ports.Periperhal interface might change.

	chanPlayer0 chan ports.Peripheral
	chanPlayer1 chan ports.Peripheral
	Player0     ports.Peripheral
	Player1     ports.Peripheral
}

func newLazyControllers(val *Lazy) *LazyControllers {
	return &LazyControllers{
		val:         val,
		chanPlayer0: make(chan ports.Peripheral, 1),
		chanPlayer1: make(chan ports.Peripheral, 1),
	}
}

func (lz *LazyControllers) update() {
	lz.val.Dbg.PushRawEvent(func() {
		select {
		case lz.chanPlayer0 <- lz.val.Dbg.VCS.RIOT.Ports.Player0:
		default:
		}
		select {
		case lz.chanPlayer1 <- lz.val.Dbg.VCS.RIOT.Ports.Player1:
		default:
		}
	})
	select {
	case lz.Player0 = <-lz.chanPlayer0:
	default:
	}
	select {
	case lz.Player1 = <-lz.chanPlayer1:
	default:
	}
}
