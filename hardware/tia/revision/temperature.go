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

package revision

import (
	"math/rand"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const hi = 50000
const lo = 2

var temp = hi
var tempCycle = 0

// HeatThreshold returns true if heat threshold is met. This is in no way an
// accurate simulation of heat. It simply emulates an increasing probability
// that the function will return true every time it is called.
func HeatThreshold(scanline int) bool {
	defer func() {
		if temp > lo {
			tempCycle++
			if tempCycle == hi/temp {
				tempCycle = 0
				if temp < 20 {
					temp--
				} else {
					temp -= temp / 20
				}
				if temp < lo {
					temp = lo
				}
			}
		}
	}()

	sl := specification.AbsoluteMaxScanlines - scanline
	if rand.Int()%50 < sl {
		return rand.Int()%temp != 0
	}
	return true
}
