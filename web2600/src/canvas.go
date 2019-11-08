// +build js
// +build wasm

package main

import (
	"encoding/base64"
	"gopher2600/television"
	"syscall/js"
	"time"
)

const pixelDepth = 4
const pixelWidth = 2
const horizScale = 2
const vertScale = 2

// CanvasTV implements television.PixelRenderer
type CanvasTV struct {
	// the worker in which our WASM application is running
	worker js.Value

	television.Television
	width  int
	height int
	top    int

	image []byte
}

// NewCanvasTV is the preferred method of initialisation for the CanvasTV type
func NewCanvasTV(worker js.Value) *CanvasTV {
	var err error

	scr := CanvasTV{worker: worker}

	scr.Television, err = television.NewStellaTelevision("NTSC")
	if err != nil {
		return nil
	}
	scr.Television.AddPixelRenderer(&scr)

	// change tv spec after window creation (so we can set the window size)
	err = scr.Resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesPerVisible)
	if err != nil {
		return nil
	}

	return &scr
}

func (scr *CanvasTV) Resize(topScanline, numScanlines int) error {
	scr.top = topScanline
	scr.height = numScanlines * vertScale

	// strictly, only the height will ever change on a specification change but
	// it's convenient to set the width too
	scr.width = television.ClocksPerVisible * pixelWidth * horizScale

	// recreate image buffer of correct length
	scr.image = make([]byte, scr.width*scr.height*pixelDepth)

	// resize HTML canvas
	scr.worker.Call("updateCanvasSize", scr.width, scr.height)

	return nil
}

// NewFrame implements telvision.PixelRenderer
func (scr *CanvasTV) NewFrame(frameNum int) error {
	scr.worker.Call("updateDebug", "frameNum", frameNum)
	encodedImage := base64.StdEncoding.EncodeToString(scr.image)
	scr.worker.Call("updateCanvas", encodedImage)

	// give way to messageHandler - there must be a more elegant way of doing this
	time.Sleep(1 * time.Millisecond)

	return nil
}

// NewScanline implements telvision.PixelRenderer
func (scr *CanvasTV) NewScanline(scanline int) error {
	scr.worker.Call("updateDebug", "scanline", scanline)
	return nil
}

// SetPixel implements telvision.PixelRenderer
func (scr *CanvasTV) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
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
	x -= television.ClocksPerHblank
	y -= scr.top

	if x < 0 || y < 0 {
		return nil
	}

	baseIdx := pixelDepth * (y*vertScale*scr.width + x*pixelWidth*horizScale)
	if baseIdx <= len(scr.image)-pixelDepth && baseIdx >= 0 {
		for h := 0; h < vertScale; h++ {
			vertAdj := h * (scr.width * pixelWidth * horizScale)
			for w := 0; w < pixelWidth*horizScale; w++ {
				horizAdj := baseIdx + (w * pixelDepth) + vertAdj
				scr.image[horizAdj] = red
				scr.image[horizAdj+1] = green
				scr.image[horizAdj+2] = blue
				scr.image[horizAdj+3] = 255
			}
		}
	}

	return nil
}

// SetAltPixel implements telvision.PixelRenderer
func (scr *CanvasTV) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return nil
}
