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

func (c *ColourGen) gammaCorrectRGB(rgb color.RGBA) color.RGBA {
	gamma := c.Gamma.Get().(float64)
	rgb.R = uint8(math.Pow(float64(rgb.R)/255, gamma) * 255)
	rgb.G = uint8(math.Pow(float64(rgb.G)/255, gamma) * 255)
	rgb.B = uint8(math.Pow(float64(rgb.B)/255, gamma) * 255)
	return rgb
}

func (c *ColourGen) gammaCorrect(r, g, b float64) (float64, float64, float64) {
	gamma := c.Gamma.Get().(float64)
	r = math.Pow(r, gamma)
	g = math.Pow(g, gamma)
	b = math.Pow(b, gamma)
	return r, g, b
}
