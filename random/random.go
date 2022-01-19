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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// the base seed for all random numbers
var baseSeed int64

// initialise base seed
func init() {
	baseSeed = int64(time.Now().Nanosecond())
}

// TV defines the television functions required by the Random type.
type TV interface {
	GetCoords() coords.TelevisionCoords
}

// Random is a random number generator that is sensitive to time within the
// emulation. Required for the rewind package and parallel emulations.
type Random struct {
	tv TV

	// use zero seed rather than the random base seed. this is only really
	// useful for normalised instances where random numbers must be predictable
	ZeroSeed bool

	// standard Go random number generator for NoRewind()
	nonRewindable rand.Source
}

// NewRandom is the preferred method of initialisation for the Random type.
func NewRandom(tv TV) *Random {
	return &Random{
		tv:            tv,
		nonRewindable: rand.NewSource(baseSeed),
	}
}

// translate television coordinates into a single value
func coordsSum(c coords.TelevisionCoords) int64 {
	return int64(c.Frame*specification.AbsoluteMaxClks + c.Scanline*specification.ClksScanline + c.Clock)
}

// Rewindable generates a random number very quickly and based on the current
// television coordinates. It's only really suitable for use in a running
// emulation.
//
// It does however have the property of being predictable during a sesssion and
// so is compatible with the rewind system.
//
// The returned number will between zero and value given in the n parameter.
func (rnd *Random) Rewindable(n int) int {
	if n == 0 {
		return 0
	}

	seed := coordsSum(rnd.tv.GetCoords())
	if !rnd.ZeroSeed {
		seed += baseSeed
	}
	seed *= seed
	b := seed >> 32
	if b != 0 {
		seed %= b
	}

	return int(seed % int64(n))
}

// NoRewind uses the standard Go library for generating random numbers. It can
// be used to generate random numbers on a non-running emulation but it is not
// therefore compatible with the rewind system.
//
// It is useful for generating a random state on startup.
//
// The returned number will between zero and value given in the n parameter.
func (rnd *Random) NoRewind(n int) int {
	if n == 0 {
		return 0
	}

	return int(rnd.nonRewindable.Int63() % int64(n))
}
