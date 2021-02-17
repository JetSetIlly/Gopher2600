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

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/reflection"
)

// textureRenderers should consider that the timing of the VCS produces
// "pixels" of two pixels across.
const pixelWidth = 2

// textureRenderers can share the underlying pixels of the screen type instance. both of these functions
// should be called inside the screen critical section.
type textureRenderer interface {
	render()
	resize()
}

// screen implements television.PixelRenderer.
type screen struct {
	img  *SdlImgui
	crit screenCrit

	// list of renderers to call from render. renderers are added with
	// addTextureRenderer()
	renderers  []textureRenderer
	emuWait    chan bool
	emuWaitAck chan bool

	// aspect bias is taken from the television specification
	aspectBias float32

	// the mouse coords used in the most recent call to PushGotoCoords(). only
	// read/write by the GUI thread so doesn't need to be in critical section.
	gotoCoordsX int
	gotoCoordsY int
}

// for clarity, variables accessed in the critical section are encapsulated in
// their own subtype.
type screenCrit struct {
	// critical sectioning
	section sync.Mutex

	// whether to follow vsync rules or not
	vsync bool

	// copy of the spec being used by the TV. the TV notifies us through the
	// Resize() function
	spec specification.Spec

	// whether the current frame was generated from a stable television state
	isStable bool

	// current values for *playable* area of the screen
	topScanline    int
	bottomScanline int

	// the pixels array is used in the presentation texture of the play and debug screen.
	pixels *image.RGBA

	// phosphor pixels
	phosphor       *image.RGBA
	updatePhosphor bool

	// bufferPixels are what we plot pixels to while we wait for a frame to complete.
	bufferPixels [5]*image.RGBA

	// which buffer we'll be plotting to and which bufffer we'll be rendering
	// from. in playmode we make sure these two indexes never meet. in
	// debugmode we plot and render from the same index, it doesn't matter.
	plotIdx   int
	renderIdx int

	// element colors and overlay colors are only used in the debugger so we
	// don't need to replicate the "backing pixels" idea.
	elementPixels *image.RGBA
	overlayPixels *image.RGBA

	// the selected overlay
	overlay string

	// 2d array of disasm entries. resized at the same time as overlayPixels resize
	reflection [][]reflection.VideoStep

	// the cropped view of the screen pixels. note that these instances are
	// created through the SubImage() command and should not be written to
	// directly
	cropPixels        *image.RGBA
	cropPhosphor      *image.RGBA
	cropElementPixels *image.RGBA
	cropOverlayPixels *image.RGBA

	// the coordinates of the last SetPixel(). used to help set the alpha
	// channel when emulation is paused
	lastX int
	lastY int
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:        img,
		emuWait:    make(chan bool),
		emuWaitAck: make(chan bool),
	}

	scr.crit.overlay = reflection.WSYNC
	scr.Reset()

	return scr
}

// Reset implements the television.PixelRenderer interface.
//
// called on startup and also whenever the VCS is reset, including when a new
// cartridge is inserted.
func (scr *screen) Reset() {
	// start off by showing entirity of NTSC screen
	scr.resize(specification.SpecNTSC, specification.SpecNTSC.AtariSafeTop, specification.SpecNTSC.AtariSafeBottom)

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	scr.clearPixels()
	scr.crit.plotIdx = 0
	scr.crit.renderIdx = 0

	scr.crit.lastX = 0
	scr.crit.lastY = 0
}

// clear all pixel information including reflection data.
//
// called when screen is reset and also when it is resize.
//
// must be called from inside a critical section.
func (scr *screen) clearPixels() {
	// clear pixels
	for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
			scr.crit.pixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.elementPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.phosphor.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
		}
	}
	for i := range scr.crit.bufferPixels {
		for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
			for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
				scr.crit.bufferPixels[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// reset reflection information
	for i := range scr.crit.reflection {
		for j := range scr.crit.reflection[i] {
			scr.crit.reflection[i][j] = reflection.VideoStep{}
		}
	}
	scr.replotOverlay()
}

// resize() is called by Resize() or resizeThread() depending on thread context.
//
// it can be called when there is no need to resize the image so steps are
// taken at the beginning of the function to return early before any
// side-effects occur.
func (scr *screen) resize(spec specification.Spec, topScanline int, bottomScanline int) {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// do nothing if resize values are the same as previously
	if scr.crit.spec.ID == spec.ID && scr.crit.topScanline == topScanline && scr.crit.bottomScanline == bottomScanline {
		return
	}

	scr.crit.spec = spec
	scr.crit.topScanline = topScanline
	scr.crit.bottomScanline = bottomScanline

	// the total number scanlines we going to have is the number total number
	// of scanlines in the screen spec +1. the additional one is so that
	// scalines with the VSYNC on show up on the uncropped screen. this is
	// particularly important if we want the cursor to be visible.
	totalScanlines := spec.ScanlinesTotal + 1

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, totalScanlines))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, totalScanlines))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, totalScanlines))
	scr.crit.phosphor = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, totalScanlines))

	for i := range scr.crit.bufferPixels {
		scr.crit.bufferPixels[i] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, totalScanlines))
	}

	// allocate reflection info
	scr.crit.reflection = make([][]reflection.VideoStep, specification.ClksScanline)
	for x := 0; x < specification.ClksScanline; x++ {
		scr.crit.reflection[x] = make([]reflection.VideoStep, totalScanlines)
	}

	// create a cropped image from the main
	crop := image.Rect(
		specification.ClksHBlank, scr.crit.topScanline,
		specification.ClksHBlank+specification.ClksVisible, scr.crit.bottomScanline,
	)
	scr.crit.cropPixels = scr.crit.pixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropPhosphor = scr.crit.phosphor.SubImage(crop).(*image.RGBA)
	scr.crit.cropElementPixels = scr.crit.elementPixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropOverlayPixels = scr.crit.overlayPixels.SubImage(crop).(*image.RGBA)

	// make sure all pixels are clear
	scr.clearPixels()

	// update aspect-bias value
	scr.aspectBias = spec.AspectBias

	// resize texture renderers
	for _, r := range scr.renderers {
		r.resize()
	}
}

// Resize implements the television.PixelRenderer interface
//
// called when the television detects a new TV specification.
//
// it is also called by the television when the rewind system is used, in order
// to make sure that the screen specification is accurate.
//
// MUST NOT be called from the gui thread.
func (scr *screen) Resize(spec specification.Spec, topScanline int, bottomScanline int) error {
	scr.img.polling.service <- func() {
		scr.resize(spec, topScanline, bottomScanline)
		scr.img.polling.serviceErr <- nil
	}
	return <-scr.img.polling.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the gui thread.
func (scr *screen) NewFrame(isStable bool) error {
	// unlocking must be done carefully
	scr.crit.section.Lock()

	scr.crit.isStable = isStable

	scr.crit.updatePhosphor = true

	if scr.img.isPlaymode() {
		scr.crit.plotIdx++
		if scr.crit.plotIdx >= len(scr.crit.bufferPixels) {
			scr.crit.plotIdx = 0
		}

		// if plot index has crashed into the render index then
		if scr.crit.plotIdx == scr.crit.renderIdx && scr.crit.vsync {
			// we must unlock the critical section or the gui thread will not
			// be able to process the channel
			scr.crit.section.Unlock()

			scr.emuWait <- true
			<-scr.emuWaitAck

			return nil
		}
	}

	scr.crit.section.Unlock()

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// UpdatingPixels implements the television PixelRenderer and PixelRefresh interfaces.
func (scr *screen) UpdatingPixels(updating bool) {
	if updating {
		scr.crit.section.Lock()
		return
	}

	scr.crit.section.Unlock()
}

// SetPixel implements the television.PixelRenderer interface.
//
// Must only be called between calls to UpdatingPixels(true) and UpdatingPixels(false).
func (scr *screen) SetPixel(sig signal.SignalAttributes, current bool) error {
	col := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	// handle VBLANK by setting pixels to black
	if !sig.VBlank {
		col = scr.crit.spec.GetColor(sig.Pixel)
	}

	if current {
		scr.crit.lastX = sig.Clock
		scr.crit.lastY = sig.Scanline
	}

	// if sig is outside the bounds of the image then the SetRGBA() will silently fail

	scr.crit.bufferPixels[scr.crit.plotIdx].SetRGBA(sig.Clock, sig.Scanline, col)

	return nil
}

// EndRendering implements the television.PixelRenderer interface.
func (scr *screen) EndRendering() error {
	return nil
}

// Reflect implements reflection.Renderer interface.
//
// Must only be called between calls to UpdatingPixels(true) and UpdatingPixels(false).
func (scr *screen) Reflect(ref reflection.VideoStep) error {
	// array indexes into the reflection 2d array are taken from the reflected
	// TV signal.
	x := ref.TV.Clock
	y := ref.TV.Scanline

	// store Reflection instance
	if x < len(scr.crit.reflection) && y < len(scr.crit.reflection[x]) {
		scr.crit.reflection[x][y] = ref
	}

	// set element pixel according to the video element that ulimately informed
	// the color of the pixel generated by the videocycle
	rgb := reflection.PaletteElements[ref.VideoElement]
	scr.crit.elementPixels.SetRGBA(x, y, rgb)

	// write to overlay. this uses the current overlay settings but it can be
	// changed later with a call to replotOverlay()
	scr.plotOverlay(x, y, ref)

	return nil
}

// replotOverlay should be called from within a scr.crit.section Lock().
func (scr *screen) replotOverlay() {
	for y := 0; y < scr.crit.overlayPixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.overlayPixels.Bounds().Size().X; x++ {
			scr.plotOverlay(x, y, scr.crit.reflection[x][y])
		}
	}
}

// plotOverlay should be called from within a scr.crit.section Lock().
func (scr *screen) plotOverlay(x, y int, ref reflection.VideoStep) {
	scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
	switch scr.crit.overlay {
	case reflection.WSYNC:
		if ref.WSYNC {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["WSYNC"])
		}
	case reflection.COLLISIONS:
		if ref.Collision.LastVideoCycle.IsCXCLR() {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["CXCLR"])
		} else if !ref.Collision.LastVideoCycle.IsNothing() {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["Collision"])
		}
	case reflection.HMOVE:
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
	case reflection.COPROCESSOR:
		if ref.CoprocessorActive {
			scr.crit.overlayPixels.SetRGBA(x, y, reflection.PaletteEvents["Coprocessor active"])
		}
	}
}

// texture renderers can share the underlying pixels in the screen instance.
func (scr *screen) addTextureRenderer(r textureRenderer) {
	scr.renderers = append(scr.renderers, r)
	r.resize()
}

// unset all attached texture renderers.
func (scr *screen) clearTextureRenderers() {
	scr.renderers = make([]textureRenderer, 0)
}

// called by service loop.
func (scr *screen) render() {
	if scr.img.isPlaymode() {
		scr.copyPixelsPlaymode()
	} else {
		scr.copyPixelsDebugmode()
	}

	for _, r := range scr.renderers {
		r.render()
	}
}

// copy pixels from buffer to the pixels and update phosphor pixels.
func (scr *screen) copyPixelsDebugmode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// copy pixels from render buffer to the live copy.
	copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)

	// update phosphor carefully
	for i := 0; i < len(scr.crit.bufferPixels[scr.crit.renderIdx].Pix); i += 4 {
		if scr.crit.pixels.Pix[i] == 0 && scr.crit.pixels.Pix[i+1] == 0 && scr.crit.pixels.Pix[i+2] == 0 {
			if scr.crit.updatePhosphor {
				// alpha channel records the number of frames the phosphor has
				// been active. starting at 255 and counting down to 0
				if scr.crit.phosphor.Pix[i+3] > 0 {
					scr.crit.phosphor.Pix[i+3]--
				}
			}
		} else {
			// copy current render pixels into phosphor
			copy(scr.crit.phosphor.Pix[i:i+2], scr.crit.bufferPixels[scr.crit.renderIdx].Pix[i:i+2])
			scr.crit.phosphor.Pix[i+3] = 0xff
		}
	}

	scr.crit.updatePhosphor = false
}

// copy pixels from buffer to the pixels and update phosphor pixels.
func (scr *screen) copyPixelsPlaymode() {
	// let the emulator thread know it's okay to continue as soon as possible
	select {
	case <-scr.emuWait:
		scr.emuWaitAck <- true
	default:
	}

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	if scr.crit.vsync {
		// advance render index. keep note of existing index in case we
		// bump into the plotting index.
		v := scr.crit.renderIdx
		scr.crit.renderIdx++
		if scr.crit.renderIdx >= len(scr.crit.bufferPixels) {
			scr.crit.renderIdx = 0
		}

		// render index has bumped into the plotting index. revert render index
		if scr.crit.renderIdx == scr.crit.plotIdx {
			scr.crit.renderIdx = v
			return
		}
	}

	// copy pixels from render buffer to the live copy.
	copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)

	// update phosphor carefully
	for i := 0; i < len(scr.crit.bufferPixels[scr.crit.renderIdx].Pix); i += 4 {
		if scr.crit.pixels.Pix[i] == 0 && scr.crit.pixels.Pix[i+1] == 0 && scr.crit.pixels.Pix[i+2] == 0 {
			// alpha channel records the number of frames the phosphor has
			// been active. starting at 255 and counting down to 0
			if scr.crit.phosphor.Pix[i+3] > 0 {
				scr.crit.phosphor.Pix[i+3]--
			}
		} else {
			// copy current render pixels into phosphor
			copy(scr.crit.phosphor.Pix[i:i+2], scr.crit.bufferPixels[scr.crit.renderIdx].Pix[i:i+2])
			scr.crit.phosphor.Pix[i+3] = 0xff
		}
	}
}
