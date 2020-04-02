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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// +build js
// +build wasm

package main

import (
	"encoding/base64"
	"image"
	"image/color"
	"syscall/js"
	"time"

	"github.com/jetsetilly/gopher2600/television"
)

const pixelWidth = 2
const horizScale = 2
const vertScale = 2

// Canvas implements television.PixelRenderer
type Canvas struct {
	// the worker in which our WASM application is running
	worker js.Value

	television.Television
	width  int
	height int
	top    int

	image *image.RGBA
}

// NewCanvas is the preferred method of initialisation for the Canvas type
func NewCanvas(worker js.Value) *Canvas {
	var err error

	scr := &Canvas{worker: worker}

	scr.Television, err = television.NewTelevision("NTSC")
	if err != nil {
		return nil
	}
	defer scr.Television.End()

	scr.Television.AddPixelRenderer(scr)

	// change tv spec after window creation (so we can set the window size)
	err = scr.Resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesVisible)
	if err != nil {
		return nil
	}

	return scr
}

// Resize implements telvision.PixelRenderer
func (scr *Canvas) Resize(topScanline, numScanlines int) error {
	scr.top = topScanline
	scr.height = numScanlines * vertScale

	// strictly, only the height will ever change on a specification change but
	// it's convenient to set the width too
	scr.width = television.HorizClksVisible * pixelWidth * horizScale

	scr.image = image.NewRGBA(image.Rect(0, 0, scr.width, scr.height))

	// resize HTML canvas
	scr.worker.Call("updateCanvasSize", scr.width, scr.height)

	return nil
}

// NewFrame implements telvision.PixelRenderer
func (scr *Canvas) NewFrame(frameNum int) error {
	scr.worker.Call("updateDebug", "frameNum", frameNum)
	encodedImage := base64.StdEncoding.EncodeToString(scr.image.Pix)
	scr.worker.Call("updateCanvas", encodedImage)

	// give way to messageHandler - there must be a more elegant way of doing this
	time.Sleep(1 * time.Millisecond)

	return nil
}

// NewScanline implements telvision.PixelRenderer
func (scr *Canvas) NewScanline(scanline int) error {
	scr.worker.Call("updateDebug", "scanline", scanline)
	return nil
}

// SetPixel implements telvision.PixelRenderer
func (scr *Canvas) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	if vblank {
		// we could return immediately but if vblank is on inside the visible
		// area we need to the set pixel to black, in case the vblank was off
		// in the previous frame (for efficiency, we're not clearing the pixel
		// array at the end of the frame)
		red = 0
		green = 0
		blue = 0
	}

	// adjust pixels so we're only dealing with the visible range
	x -= television.HorizClksHBlank
	y -= scr.top

	if x < 0 || y < 0 {
		return nil
	}

	rgb := color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)}

	for h := 0; h < vertScale; h++ {
		for w := 0; w < horizScale*pixelWidth; w++ {
			scr.image.SetRGBA(
				(x*horizScale*pixelWidth)+w,
				(y*vertScale)+h,
				rgb)
		}
	}

	return nil
}

// SetAltPixel implements telvision.PixelRenderer
func (scr *Canvas) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return nil
}

// EndRendering implements telvision.PixelRenderer
func (scr *Canvas) EndRendering() error {
	return nil
}
