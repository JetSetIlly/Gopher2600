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

package colourgen

import (
	"math"
)

func (c *ColourGen) adjustYIQ(Y, I, Q float64) (float64, float64, float64) {
	brightness := c.Brightness.Get().(float64)
	contrast := c.Contrast.Get().(float64)
	saturation := c.Saturation.Get().(float64)
	hue := c.Hue.Get().(float64)

	// C = contrast
	// YIQ * |  C   0   0  |
	//       |  0   1   0  |
	//       |  0   0   1  |
	Y *= contrast

	// B = brightness
	// YIQ + |  B   0   0  |
	//       |  0   0   0  |
	//       |  0   0   0  |
	Y += brightness

	// S = saturation
	// YIQ * |  1   0   0  |
	//       |  0   S   0  |
	//       |  0   0   S  |
	I *= saturation
	Q *= saturation

	// hue is stored in degrees but we need radians for the math functions
	hue *= math.Pi / 180.0

	// the hue rotation of I and Q should happen on the unrotated values. for
	// this reason, we store the rotated Q value in a temporary variable
	//
	// H = hue
	// YIQ * |  1     0       0     |
	//       |  0  cos(H)  -sin(H)  |
	//       |  0  sin(H)   cos(H)  |
	var q float64
	q = (math.Sin(hue) * I) + (math.Cos(hue) * Q)
	I = (math.Cos(hue) * I) - (math.Sin(hue) * Q)
	Q = q

	return Y, I, Q
}

func (c *ColourGen) adjustYUV(Y, U, V float64) (float64, float64, float64) {
	// adjustYIQ() works just as well on YUV values as YIQ
	return c.adjustYIQ(Y, U, V)
}
