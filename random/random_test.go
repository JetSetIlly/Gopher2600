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

package random_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/random"
	"github.com/jetsetilly/gopher2600/test"
)

type tv struct {
}

func (m *tv) GetCoords() coords.TelevisionCoords {
	return coords.TelevisionCoords{
		Frame:    100,
		Scanline: 32,
		Clock:    10,
	}
}

func TestRandom(t *testing.T) {
	a := random.NewRandom(&tv{})
	b := random.NewRandom(&tv{})
	a.ZeroSeed = true
	b.ZeroSeed = true

	for i := 1; i < 256; i++ {
		test.ExpectEquality(t, a.Rewindable(i), b.Rewindable(i))
	}
}
