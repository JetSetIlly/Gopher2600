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
	"image/color"
	"math"
)

func clampRange(v float64, mn float64, mx float64) float64 {
	if v < mn {
		return mn
	}
	if v > mx {
		return mx
	}
	return v
}

func adjustYIQ(Y, I, Q float64, brightness, contrast, saturation, hue float64) (float64, float64, float64) {
	// C = contrast
	// YIQ * |  C   0   0  |
	//       |  0   1   0  |
	//       |  0   0   1  |
	Y = 0.5 + (Y-0.5)*contrast

	// B = brightness
	// YIQ * |  B   0   0  |
	//       |  0   1   0  |
	//       |  0   0   1  |
	Y *= brightness

	// clamp Y after contrast and brightness transforms
	Y = clampRange(Y, 0.0, 0.90)

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

func adjustRGB(col color.RGBA, brightness, contrast, saturation, hue float64) color.RGBA {
	// clamp black value at zero. if we don't do this then the black will be
	// affected by the contrast setting, which we don't want
	if col.R == 0 && col.G == 0 && col.B == 0 {
		return col
	}

	var R, G, B float64
	R = float64(col.R) / 255
	G = float64(col.G) / 255
	B = float64(col.B) / 255

	var Y, I, Q float64

	Y = 0.299*R + 0.587*G + 0.114*B
	I = 0.5959*R - 0.2746*G - 0.3213*B
	Q = 0.2115*R - 0.5227*G + 0.31122*B

	Y, I, Q = adjustYIQ(Y, I, Q, brightness, contrast, saturation, hue)

	col.R = uint8(clamp(Y+(0.956*I)+(0.619*Q)) * 255)
	col.G = uint8(clamp(Y-(0.272*I)-(0.647*Q)) * 255)
	col.B = uint8(clamp(Y-(1.106*I)+(1.703*Q)) * 255)

	return col
}

func (c *ColourGen) adjustRGB(col color.RGBA) color.RGBA {
	brightness := c.Brightness.Get().(float64)
	contrast := c.Contrast.Get().(float64)
	saturation := c.Saturation.Get().(float64)
	hue := c.Hue.Get().(float64)
	return adjustRGB(col, brightness, contrast, saturation, hue)
}

func (c *ColourGen) adjustYIQ(Y, I, Q float64) (float64, float64, float64) {
	brightness := c.Brightness.Get().(float64)
	contrast := c.Contrast.Get().(float64)
	saturation := c.Saturation.Get().(float64)
	hue := c.Hue.Get().(float64)
	return adjustYIQ(Y, I, Q, brightness, contrast, saturation, hue)
}

func (c *ColourGen) adjustYUV(Y, U, V float64) (float64, float64, float64) {
	// adjustYIQ() works just as well on YUV values as YIQ
	return c.adjustYIQ(Y, U, V)
}
