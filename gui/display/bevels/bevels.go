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

package bevels

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/jpeg"
)

//go:embed "solid_state.jpg"
var tv_jpg []byte

// TV is the decoded image
var TV *image.RGBA

// Transformations for the decoded TV image
const (
	Scale   = 0.85
	OffsetX = -0.124
	OffsetY = -0.061
)

func init() {
	r := bytes.NewReader(tv_jpg)
	img, err := jpeg.Decode(r)
	if err != nil {
		panic(err)
	}
	TV = imageToRGBA(img)
}

func imageToRGBA(src image.Image) *image.RGBA {
	// no conversion needed if image is an *image.RGBA
	if dst, ok := src.(*image.RGBA); ok {
		return dst
	}

	// use the image/draw package to convert to *image.RGBA
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
