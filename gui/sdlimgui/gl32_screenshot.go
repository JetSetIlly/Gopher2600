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

//go:build !gl21

package sdlimgui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.2-core/gl"
)

const (
	maxFrames = 5
)

type gl32Screenshot struct {
	images [maxFrames]*image.RGBA

	width  int32
	height int32

	idx    int
	frames int

	mode   screenshotMode
	finish chan screenshotResult

	// the finalisation of the screenshot happens in a separate goroutine. the
	// result is passed over the finalise channel before being passed to the
	// finish channel
	//
	// the finalise channel is closed and nilled once the result has been
	// received. checking the finalise field for nil can be used to test if the
	// finalise goroutine is working, so long as the field is given the value
	// nil after a result has been received
	//
	// the goroutine accesses the other fields in this struct so it's important
	// that once the finalise channel has been created no other field is touched
	// outside of the finalise goroutine
	finalise chan screenshotResult
}

func newGl32Screenshot() *gl32Screenshot {
	sht := &gl32Screenshot{}
	return sht
}

func (sht *gl32Screenshot) destroy() {
	close(sht.finalise)
}

func (sht *gl32Screenshot) finished() bool {
	return sht.finalise == nil && sht.idx >= sht.frames
}

func (sht *gl32Screenshot) start(mode screenshotMode, finish chan screenshotResult) {
	// checking sht.finalise first ensures the the idx and frames fields aren't
	// touched (in the finished() function) if the condition fails. in other
	// words, if finalise is not nil then we have successfully avoided accessing
	// other fields in the sht struct
	if sht.finalise != nil && !sht.finished() {
		finish <- screenshotResult{
			err: fmt.Errorf("previous screenshotting still in progress"),
		}
		return
	}

	switch mode {
	case modeSingle:
		sht.frames = 1
	case modeDouble:
		sht.frames = 2
	case modeTriple:
		sht.frames = 3
	case modeMovement:
		sht.frames = 5
	default:
		return
	}
	sht.mode = mode
	sht.finish = finish
	sht.idx = 0
}

func (sht *gl32Screenshot) process(width int32, height int32) {
	// checking for a finalised result before continuing. if the finalise
	// channel is not nil (ie. a finalise goroutine is in progress) we don't
	// want to continue unless a result is ready on the finalise channel
	if sht.finalise != nil {
		select {
		case r := <-sht.finalise:
			sht.finish <- r
			close(sht.finalise)
			sht.finalise = nil
		default:
			return
		}
	}

	// the finished() function checks that a screenshot has recently been started
	if sht.finished() {
		return
	}

	if width != sht.width || height != sht.height {
		sht.width = width
		sht.height = height
		for i := range sht.images {
			sht.images[i] = image.NewRGBA(image.Rect(0, 0, int(sht.width), int(sht.height)))
			if sht.images[i] == nil {
				sht.finish <- screenshotResult{
					err: fmt.Errorf("save failed: cannot allocate image data"),
				}
				sht.idx = sht.frames
				return
			}
		}
		sht.idx = 0
		return
	}

	gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(sht.images[sht.idx].Pix))

	sht.idx++
	if !sht.finished() {
		return
	}

	sht.finalise = make(chan screenshotResult)
	go func() {
		// create final image
		final := image.NewRGBA(image.Rect(0, 0, int(sht.width), int(sht.height)))
		if final == nil {
			sht.finish <- screenshotResult{
				err: fmt.Errorf("save failed: cannot allocate image data"),
			}
			sht.idx = sht.frames
			return
		}

		// blend pixels
		for p := 0; p < len(final.Pix); p++ {
			var a int
			for f := 0; f < sht.idx; f++ {
				a += int(sht.images[f].Pix[p])
			}
			final.Pix[p] = uint8(a / sht.frames)
		}

		// special treatment of movement mode
		if sht.mode == modeMovement {
			luminance := func(rgba color.RGBA) float32 {
				return 0.299*float32(rgba.R) + 0.587*float32(rgba.G) + 0.114*float32(rgba.B)
			}

			for y := range int(sht.height) {
				for x := range int(sht.width) {
					px := sht.images[sht.idx-1].RGBAAt(x, y)
					fpx := final.RGBAAt(x, y)
					if luminance(px) >= luminance(fpx) {
						final.Set(x, y, px)
					}
				}
			}
		}

		// flip pixels
		rowSize := int(sht.width * 4)
		swp := make([]byte, rowSize)
		for y := 0; y < int(sht.height)/2; y++ {
			top := final.Pix[y*rowSize : (y+1)*rowSize]
			bot := final.Pix[(int(height)-y-1)*rowSize : (int(height)-y)*rowSize]
			copy(swp, top)
			copy(top, bot)
			copy(bot, swp)
		}

		var r screenshotResult
		r.description = string(sht.mode)
		r.image = final

		sht.finalise <- r
	}()
}
