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
	"image/png"
)

type Style struct {
	TV      *image.RGBA
	Scale   float32
	OffsetX float32
	OffsetY float32
	BiasY   float32
}

var Selected Style

var SolidState Style
var Telefunken Style

//go:embed "solid_state.png"
var solidState []byte

//go:embed "telefunken.png"
var telefunken []byte

func init() {
	SolidState.TV = loadImage(solidState)
	SolidState.Scale = 0.85
	SolidState.OffsetX = -0.139
	SolidState.OffsetY = -0.085
	SolidState.BiasY = 1.05

	Telefunken.TV = loadImage(telefunken)
	Telefunken.Scale = 0.93
	Telefunken.OffsetX = -0.075
	Telefunken.OffsetY = -0.014
	Telefunken.BiasY = 1.05

	Selected = SolidState
}

func loadImage(d []byte) *image.RGBA {
	r := bytes.NewReader(d)

	img, err := png.Decode(r)
	if err != nil {
		panic(err)
	}

	// no conversion needed if image is an *image.RGBA
	if dst, ok := img.(*image.RGBA); ok {
		return dst
	}

	// use the image/draw package to convert to *image.RGBA
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)

	return dst
}