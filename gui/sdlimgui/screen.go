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

package sdlimgui

import (
	"image"
	"image/color"
	"sync"

	"github.com/jetsetilly/gopher2600/television"
	"github.com/jetsetilly/gopher2600/test"

	"github.com/go-gl/gl/v3.2-core/gl"
)

const (
	pixelWidth = 2
	defScaling = 2.0
)

// screen implements television.PixelRenderer
type screen struct {
	img *SdlImgui

	crit screenCrit

	// which set of pixels to use: cropped or unmasked
	cropped bool

	// the basic amount by which the image should be scaled. image width
	// is also scaled by pixelWidth and aspectBias value
	scaling float32

	// aspect bias is taken from the television specification
	aspectBias float32

	// create texture on the next call of render
	createTextures bool

	// the tv screen texture
	screenTexture uint32

	// whether to use the alternative pixel layer
	useAltPixels bool

	// show pixel perfect image or with crt effect
	pixelPerfect bool
}

// for clarity, variables accessed in the critical section are encapsulated in
// their own subtype
type screenCrit struct {
	// critical sectioning
	section sync.RWMutex

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int

	// pixels and altPixels should be constructed exactly the same. the only
	// difference is the colors
	pixels    *image.RGBA
	altPixels *image.RGBA

	// the cropped view of the screen pixels. note that these instances are
	// created through the SubImage() command and should not be written to
	// directly
	cropPixels    *image.RGBA
	cropAltPixels *image.RGBA

	// the coordinates of the last SetPixel(). used to help set the alpha
	// channel when emulation is paused
	lastX int
	lastY int
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:          img,
		scaling:      defScaling,
		cropped:      true,
		pixelPerfect: true,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &scr.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, scr.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// start off by showing entirity of NTSC screen
	scr.resize(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	scr.crit.lastX = 0
	scr.crit.lastY = 0

	return scr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *screen) resize(topScanline int, visibleScanlines int) error {
	scr.crit.section.RLock()
	// we need to be careful with this lock (so no defer)

	scr.crit.topScanline = topScanline
	scr.crit.scanlines = visibleScanlines

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))
	scr.crit.altPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))

	// create a cropped image from the main
	r := image.Rectangle{
		image.Point{television.HorizClksHBlank,
			scr.crit.topScanline,
		},
		image.Point{television.HorizClksHBlank + television.HorizClksVisible,
			scr.crit.topScanline + scr.crit.scanlines,
		},
	}
	scr.crit.cropPixels = scr.crit.pixels.SubImage(r).(*image.RGBA)
	scr.crit.cropAltPixels = scr.crit.altPixels.SubImage(r).(*image.RGBA)

	// clear pixels. SetPixel() alters the value of lastX and lastY. we don't
	// really want it to do that however, so we note these value and restore
	// them after the clearing loops
	lastX := scr.crit.lastX
	lastY := scr.crit.lastY

	// unlock critical section before calling SetPixel() (or we'll deadlock)
	scr.crit.section.RUnlock()

	for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
			scr.SetPixel(x, y, 0, 0, 0, false)
			scr.SetAltPixel(x, y, 0, 0, 0, false)
		}
	}

	// reapply critical section after calls to SetPixel()
	scr.crit.section.RLock()
	scr.crit.lastX = lastX
	scr.crit.lastY = lastY
	scr.crit.section.RUnlock()

	// update aspect-bias value
	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	// defer re-creation of texture to render(). we have to do it in the
	// #mainthread so we may as wait until that function is called
	scr.createTextures = true

	return nil
}

func (scr *screen) scaledWidth() float32 {
	return float32(scr.crit.pixels.Bounds().Size().X) * scr.horizScaling()
}

func (scr *screen) scaledHeight() float32 {
	return float32(scr.crit.pixels.Bounds().Size().Y) * scr.vertScaling()
}

func (scr *screen) scaledCroppedWidth() float32 {
	return float32(scr.crit.cropPixels.Bounds().Size().X) * scr.horizScaling()
}

func (scr *screen) scaledCroppedHeight() float32 {
	return float32(scr.crit.cropPixels.Bounds().Size().Y) * scr.vertScaling()
}

func (scr *screen) horizScaling() float32 {
	return float32(pixelWidth * scr.aspectBias * scr.scaling)
}

func (scr *screen) vertScaling() float32 {
	return scr.scaling
}

// render is called by service loop
func (scr *screen) render() {
	scr.crit.section.RLock()
	defer scr.crit.section.RUnlock()

	var pixels *image.RGBA
	if scr.useAltPixels {
		if scr.cropped {
			pixels = scr.crit.cropAltPixels
		} else {
			pixels = scr.crit.altPixels
		}
	} else {
		if scr.cropped {
			pixels = scr.crit.cropPixels
		} else {
			pixels = scr.crit.pixels
		}
	}

	gl.ActiveTexture(gl.TEXTURE0)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)

	if scr.createTextures {
		scr.createTextures = false
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
		gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	} else {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
}

func (scr *screen) setCropping(set bool) {
	scr.cropped = set
	scr.createTextures = true
}

// Resize implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) Resize(topScanline int, visibleScanlines int) error {
	scr.img.service <- func() {
		scr.img.serviceErr <- scr.resize(topScanline, visibleScanlines)
	}
	return <-scr.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) NewFrame(frameNum int) error {
	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements the television.PixelRenderer interface
func (scr *screen) SetPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	test.AssertNonMainThread()

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// handle VBLANK by setting pixels to black
	if vblank {
		red = 0
		green = 0
		blue = 0
	}

	rgb := color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)}

	scr.crit.lastX = x
	scr.crit.lastY = y
	scr.crit.pixels.SetRGBA(scr.crit.lastX, scr.crit.lastY, rgb)

	return nil
}

// SetAltPixel implements the television.PixelRenderer interface
func (scr *screen) SetAltPixel(x int, y int, red byte, green byte, blue byte, vblank bool) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	rgb := color.RGBA{uint8(red), uint8(green), uint8(blue), uint8(255)}
	scr.crit.altPixels.SetRGBA(x, y, rgb)

	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}
