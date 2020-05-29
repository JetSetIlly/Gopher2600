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

	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/television"
)

// textureRenderers should consider that the timing of the VCS produces
// "pixels" of two pixels across
const pixelWidth = 2

// textureRenderers can share the underlying pixels of the screen type instance
type textureRenderers interface {
	render()
	resize()
}

// screen implements television.PixelRenderer
type screen struct {
	img  *SdlImgui
	crit screenCrit

	// list of renderers to call from render. renderers are added with
	// addTextureRenderer()
	renderers []textureRenderers

	// aspect bias is taken from the television specification
	aspectBias float32
}

// for clarity, variables accessed in the critical section are encapsulated in
// their own subtype
type screenCrit struct {
	// critical sectioning
	section sync.Mutex

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int

	// all pixel arrays should be constructed the same
	pixels        *image.RGBA
	debugPixels   *image.RGBA
	overlayPixels *image.RGBA

	// the selected overlay
	overlay string

	// 2d array of disasm entries. resized at the same time as overlayPixels resize
	reflection [][]reflection.LastResult

	// the cropped view of the screen pixels. note that these instances are
	// created through the SubImage() command and should not be written to
	// directly
	cropPixels        *image.RGBA
	cropElementPixels *image.RGBA
	cropOverlayPixels *image.RGBA

	// the coordinates of the last SetPixel(). used to help set the alpha
	// channel when emulation is paused
	lastX int
	lastY int
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{img: img}

	// start off by showing entirity of NTSC screen
	scr.resize(television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	scr.crit.lastX = 0
	scr.crit.lastY = 0
	scr.crit.overlay = reflection.OverlayList[0]

	return scr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *screen) resize(topScanline int, visibleScanlines int) error {
	scr.crit.section.Lock()
	// we need to be careful with this lock (so no defer)

	scr.crit.topScanline = topScanline
	scr.crit.scanlines = visibleScanlines

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))
	scr.crit.debugPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, scr.img.tv.GetSpec().ScanlinesTotal))

	// allocate disasm info
	scr.crit.reflection = make([][]reflection.LastResult, television.HorizClksScanline)
	for x := 0; x < television.HorizClksScanline; x++ {
		scr.crit.reflection[x] = make([]reflection.LastResult, scr.img.tv.GetSpec().ScanlinesTotal)
	}

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
	scr.crit.cropElementPixels = scr.crit.debugPixels.SubImage(r).(*image.RGBA)
	scr.crit.cropOverlayPixels = scr.crit.overlayPixels.SubImage(r).(*image.RGBA)

	// clear pixels. SetPixel() alters the value of lastX and lastY. we don't
	// really want it to do that however, so we note these value and restore
	// them after the clearing loops
	lastX := scr.crit.lastX
	lastY := scr.crit.lastY

	for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
			scr.crit.pixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.debugPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	scr.crit.lastX = lastX
	scr.crit.lastY = lastY
	scr.crit.section.Unlock()

	// update aspect-bias value
	scr.aspectBias = scr.img.tv.GetSpec().AspectBias

	// resize texture renderers
	for _, r := range scr.renderers {
		r.resize()
	}

	return nil
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

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}

// Reflect implements reflection.Renderer interface
func (scr *screen) Reflect(result reflection.LastResult) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// store LastResult instance
	if scr.crit.lastX < len(scr.crit.reflection) && scr.crit.lastY < len(scr.crit.reflection[scr.crit.lastX]) {
		scr.crit.reflection[scr.crit.lastX][scr.crit.lastY] = result
	}

	// set debug pixel
	rgb := reflection.PaletteElements[result.VideoElement]
	scr.crit.debugPixels.SetRGBA(scr.crit.lastX, scr.crit.lastY, rgb)

	// write to overlay
	scr.plotOverlay(scr.crit.lastX, scr.crit.lastY, result)

	return nil
}

// replotOverlay should be called from within a scr.crit.section Lock()
func (scr *screen) replotOverlay() {
	for y := 0; y < scr.crit.overlayPixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.overlayPixels.Bounds().Size().X; x++ {
			scr.plotOverlay(x, y, scr.crit.reflection[x][y])
		}
	}
}

// plotOverlay should be called from within a scr.crit.section Lock()
func (scr *screen) plotOverlay(x, y int, result reflection.LastResult) {
	scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
	switch scr.crit.overlay {
	case "WSYNC":
		if result.WSYNC {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["WSYNC"])
		}
	case "Collisions":
		if result.Collision != "" {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["Collisions"])
		}
	}
}

// texture renderers can share the underlying pixels in the screen instance
func (scr *screen) addTextureRenderer(r textureRenderers) {
	scr.renderers = append(scr.renderers, r)
}

func (scr *screen) render() {
	for _, r := range scr.renderers {
		r.render()
	}
}
