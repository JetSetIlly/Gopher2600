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

	// whether the current frame was generated from a stable television state
	isStable bool

	// current values for *playable* area of the screen
	topScanline int
	scanlines   int

	// the pixels array is used in the presentation texture of the play and
	// debug screen.
	pixels *image.RGBA

	// backingPixels are what we plot pixels to while we wait for a frame to
	// complete. see NewFrame() and render() functions below for how we achieve
	// this.
	backingPixels         [2]*image.RGBA
	backingPixelsCurrent  int
	backingPixelsToRender int
	backingPixelsUpdate   bool

	// debug colors and overlay colors are only used in the debugger. we're not
	// worried about drawing to them directly
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
	scr.resize(television.SpecNTSC, television.SpecNTSC.ScanlineTop, television.SpecNTSC.ScanlinesVisible)

	scr.crit.lastX = 0
	scr.crit.lastY = 0
	scr.crit.overlay = reflection.OverlayList[0]

	return scr
}

// resize() is called by Resize() or resizeThread() depending on thread context
func (scr *screen) resize(spec *television.Specification, topScanline int, visibleScanlines int) {
	scr.crit.section.Lock()
	// we need to be careful with this lock (so no defer)

	scr.crit.topScanline = topScanline
	scr.crit.scanlines = visibleScanlines

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, spec.ScanlinesTotal))
	for i := range scr.crit.backingPixels {
		scr.crit.backingPixels[i] = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, spec.ScanlinesTotal))
	}
	scr.crit.debugPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, spec.ScanlinesTotal))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, television.HorizClksScanline, spec.ScanlinesTotal))

	// allocate disasm info
	scr.crit.reflection = make([][]reflection.LastResult, television.HorizClksScanline)
	for x := 0; x < television.HorizClksScanline; x++ {
		scr.crit.reflection[x] = make([]reflection.LastResult, spec.ScanlinesTotal)
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

			// backing pixels too
			for i := range scr.crit.backingPixels {
				scr.crit.backingPixels[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	scr.crit.lastX = lastX
	scr.crit.lastY = lastY

	// end critical section
	scr.crit.section.Unlock()

	// update aspect-bias value
	scr.aspectBias = spec.AspectBias

	// resize texture renderers
	for _, r := range scr.renderers {
		r.resize()
	}
}

// Resize implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) Resize(spec *television.Specification, topScanline int, visibleScanlines int) error {
	scr.img.service <- func() {
		scr.resize(spec, topScanline, visibleScanlines)
		scr.img.serviceErr <- nil
	}
	return <-scr.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the #mainthread
func (scr *screen) NewFrame(frameNum int, isStable bool) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	scr.crit.isStable = isStable

	scr.crit.backingPixelsUpdate = true
	if scr.crit.backingPixelsCurrent < len(scr.crit.backingPixels)-1 {
		copy(scr.crit.backingPixels[scr.crit.backingPixelsCurrent+1].Pix, scr.crit.backingPixels[scr.crit.backingPixelsCurrent].Pix)
		scr.crit.backingPixelsCurrent++
	} else {
		copy(scr.crit.backingPixels[0].Pix, scr.crit.backingPixels[scr.crit.backingPixelsCurrent].Pix)
		scr.crit.backingPixelsCurrent = 0
	}

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

	rgb := color.RGBA{red, green, blue, 255}

	scr.crit.lastX = x
	scr.crit.lastY = y
	scr.crit.backingPixels[scr.crit.backingPixelsCurrent].SetRGBA(x, y, rgb)

	return nil
}

// EndRendering implements the television.PixelRenderer interface
func (scr *screen) EndRendering() error {
	return nil
}

// Reflect implements reflection.Renderer interface
func (scr *screen) Reflect(ref reflection.LastResult) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// store LastResult instance
	if scr.crit.lastX < len(scr.crit.reflection) && scr.crit.lastY < len(scr.crit.reflection[scr.crit.lastX]) {
		scr.crit.reflection[scr.crit.lastX][scr.crit.lastY] = ref
	}

	// set debug pixel
	rgb := reflection.PaletteElements[ref.VideoElement]
	scr.crit.debugPixels.SetRGBA(scr.crit.lastX, scr.crit.lastY, rgb)

	// write to overlay
	scr.plotOverlay(scr.crit.lastX, scr.crit.lastY, ref)

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
func (scr *screen) plotOverlay(x, y int, ref reflection.LastResult) {
	scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
	switch scr.crit.overlay {
	case "WSYNC":
		if ref.WSYNC {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["WSYNC"])
		}
	case "Collisions":
		if ref.Collision != "" {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["Collisions"])
		}
	case "HMOVE":
		// HmoveCt counts to -1 (or 255 for a uint8)
		if ref.Hmove.Delay {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["HMOVE delay"])
		} else if ref.Hmove.Latch {
			if ref.Hmove.RippleCt != 255 {
				scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["HMOVE"])
			} else {
				scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["HMOVE latched"])
			}
		}
	case "Unchanged":
		if ref.Unchanged {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["Unchanged"])
		}
	}
}

// texture renderers can share the underlying pixels in the screen instance
func (scr *screen) addTextureRenderer(r textureRenderers) {
	scr.renderers = append(scr.renderers, r)
}

func (scr *screen) render() {
	// critical section
	scr.crit.section.Lock()
	if scr.crit.backingPixelsUpdate {
		copy(scr.crit.pixels.Pix, scr.crit.backingPixels[scr.crit.backingPixelsToRender].Pix)
		scr.crit.backingPixelsToRender = scr.crit.backingPixelsCurrent
		scr.crit.backingPixelsUpdate = false
	}
	scr.crit.section.Unlock()
	// end of critical section

	for _, r := range scr.renderers {
		r.render()
	}
}
