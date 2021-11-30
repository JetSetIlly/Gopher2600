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

package random

import (
	"math/rand"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// the base seed for all random numbers
var baseSeed int64

// initialise base seed
func init() {
	baseSeed = int64(time.Now().Nanosecond())
}

// Random is a random number generator that is sensitive to time within the
// emulation. Required for the rewind package and parallel emulations.
type Random struct {
	coords signal.TelevisionCoords

	// use zero seed rather than the random base seed. this is only really
	// useful for normalised instances where random numbers must be predictable
	ZeroSeed bool
}

// NewRandom is the preferred method of initialisation for the Random type.
func NewRandom(coords signal.TelevisionCoords) *Random {
	return &Random{
		coords: coords,
	}
}

// translate television coordinates into a single value
func coordsSum(c coords.TelevisionCoords) int64 {
	return int64(c.Frame*specification.AbsoluteMaxClks + c.Scanline*specification.ClksScanline + c.Clock)
}

// new RNG from the standard library
func (rnd *Random) rand() *rand.Rand {
	if rnd.ZeroSeed {
		return rand.New(rand.NewSource(coordsSum(rnd.coords.GetCoords())))
	}
	return rand.New(rand.NewSource(baseSeed + coordsSum(rnd.coords.GetCoords())))
}

func (rnd *Random) Intn(n int) int {
	return rnd.rand().Intn(n)
}
