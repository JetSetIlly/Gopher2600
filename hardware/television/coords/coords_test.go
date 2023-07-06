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

package coords_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/test"
)

func TestEqual(t *testing.T) {
	A := coords.TelevisionCoords{
		Frame:    0,
		Scanline: 0,
		Clock:    0,
	}
	B := coords.TelevisionCoords{
		Frame:    0,
		Scanline: 0,
		Clock:    1,
	}

	// clock fields are different (other fields equal)
	test.ExpectFailure(t, coords.Equal(A, B))

	// all fields are equal
	B.Clock = 0
	test.ExpectSuccess(t, coords.Equal(A, B))

	// scanline fields are different (other fields equal)
	B.Scanline = 1
	test.ExpectFailure(t, coords.Equal(A, B))

	// all fields are equal
	A.Scanline = 1
	test.ExpectSuccess(t, coords.Equal(A, B))

	// frame fields are different
	A.Frame = 1
	test.ExpectFailure(t, coords.Equal(A, B))

	// frame fields are different but one is undefined
	B.Frame = coords.FrameIsUndefined
	test.ExpectSuccess(t, coords.Equal(A, B))
}
