// +build js
// +build wasm

package main

import (
	"encoding/base64"
	"gopher2600/television"
	"strconv"
	"syscall/js"
	"time"
)

const screenDepth = 4
const pixelWidth = 2
const horizScale = 2
const vertScale = 2

// CanvasTV implements television.PixelRenderer
type CanvasTV struct {
	// the worker in which our WASM application is running
	worker js.Value

	television.Television
	spec   *television.Specification
	width  int
	height int

	screenTop int

	image []byte
}

// NewCanvasTV is the preferred method of initialisation for the CanvasTV type
func NewCanvasTV(worker js.Value) *CanvasTV {
	var err error

	ctv := CanvasTV{worker: worker}

	ctv.Television, err = television.NewStellaTelevision("NTSC")
	if err != nil {
		return nil
	}
	ctv.Television.AddPixelRenderer(&ctv)
	ctv.ChangeTVSpec()

	return &ctv
}

// NewFrame implements telvision.PixelRenderer
func (ctv *CanvasTV) NewFrame(frameNum int) error {
	ctv.worker.Call("updateDebug", "frameNum", frameNum)
	encodedImage := base64.StdEncoding.EncodeToString(ctv.image)
	ctv.worker.Call("updateCanvas", encodedImage)
	ctv.screenTop = -1

	// give way to messageHandler - there must be a more elegant way of doing this
	time.Sleep(1 * time.Millisecond)

	return nil
}

// NewScanline implements telvision.PixelRenderer
func (ctv *CanvasTV) NewScanline(scanline int) error {
	ctv.worker.Call("updateDebug", "scanline", scanline)
	return nil
}

// SetPixel implements telvision.PixelRenderer
func (ctv *CanvasTV) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	// adjust pixels so we're only dealing with the visible range
	x -= television.ClocksPerHblank
	if x < 0 {
		return nil
	}

	// we need to be careful how we treat VBLANK signals. some ROMs use VBLANK
	// as a cheap way of showing a black pixel. so, at the start of every new
	// frame we set the following to -1 and then to the current scanline at the
	// moment VBLANK is turned of for the first time that frame.
	if !vblank {
		if ctv.screenTop == -1 {
			ctv.screenTop = y
		}
	} else {
		if ctv.screenTop == -1 {
			return nil
		} else {
			red = 0
			green = 0
			blue = 0
		}
	}

	y -= ctv.screenTop

	baseIdx := screenDepth * (y*vertScale*ctv.width + x*pixelWidth*horizScale)
	if baseIdx < len(ctv.image)-screenDepth && baseIdx >= 0 {
		for h := 0; h < vertScale; h++ {
			vertAdj := h * (ctv.width * pixelWidth * horizScale)
			for w := 0; w < pixelWidth*horizScale; w++ {
				horizAdj := baseIdx + (w * screenDepth) + vertAdj
				ctv.image[horizAdj] = red
				ctv.image[horizAdj+1] = green
				ctv.image[horizAdj+2] = blue
				ctv.image[horizAdj+3] = 255
			}
		}
	}

	return nil
}

// SetAltPixel implements telvision.PixelRenderer
func (ctv *CanvasTV) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return nil
}

// ChangeTVSpec implements telvision.PixelRenderer
func (ctv *CanvasTV) ChangeTVSpec() error {
	ctv.spec = ctv.Television.GetSpec()
	ctv.height = ctv.spec.ScanlinesPerVisible * vertScale

	// strictly, only the height will ever change on a specification change but
	// it's convenient to set the width too
	ctv.width = television.ClocksPerVisible * pixelWidth * horizScale

	// recreate image buffer of correct length
	ctv.image = make([]byte, ctv.width*ctv.height*screenDepth)

	// resize HTML canvas
	ctv.worker.Call("updateCanvasSize", strconv.Itoa(ctv.width), strconv.Itoa(ctv.height))

	return nil
}
