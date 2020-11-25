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

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/reflection"
)

// textureRenderers should consider that the timing of the VCS produces
// "pixels" of two pixels across.
const pixelWidth = 2

// textureRenderers can share the underlying pixels of the screen type instance.
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
	topScanline int
	scanlines   int

	// the pixels array is used in the presentation texture of the play and
	// debug screen.
	pixels *image.RGBA

	// bufferPixels are what we plot pixels to while we wait for a frame to complete.
	bufferPixels [5]*image.RGBA
	bufferUpdate bool

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
	reflection [][]reflection.Reflection

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
	scr := &screen{
		img:        img,
		emuWait:    make(chan bool),
		emuWaitAck: make(chan bool),
	}

	// start off by showing entirity of NTSC screen
	scr.resize(specification.SpecNTSC, specification.SpecNTSC.AtariSafeTop, specification.SpecNTSC.ScanlinesVisible)

	scr.crit.lastX = 0
	scr.crit.lastY = 0
	scr.crit.overlay = reflection.OverlayList[0]

	// start off with a buffer update to make sure the textureRenderer
	// implementations have good information about the pixel data as soon as
	// possible. without this, the visible screen window will jump from its
	// initial scaling value to the correct one.
	scr.crit.bufferUpdate = true

	return scr
}

// resize() is called by Resize() or resizeThread() depending on thread context.
func (scr *screen) resize(spec specification.Spec, topScanline int, visibleScanlines int) {
	// never resize below the visible scanlines according to the specification
	if visibleScanlines < spec.ScanlinesVisible {
		return
	}

	scr.crit.section.Lock()
	// we need to be careful with this lock (so no defer)

	// do nothing if resize values are the same as previously
	if scr.crit.spec.ID == spec.ID && scr.crit.topScanline == topScanline && scr.crit.scanlines == visibleScanlines {
		scr.crit.section.Unlock()
		return
	}

	scr.crit.spec = spec
	scr.crit.topScanline = topScanline
	scr.crit.scanlines = visibleScanlines

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, specification.HorizClksScanline, spec.ScanlinesTotal))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.HorizClksScanline, spec.ScanlinesTotal))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.HorizClksScanline, spec.ScanlinesTotal))

	for i := range scr.crit.bufferPixels {
		scr.crit.bufferPixels[i] = image.NewRGBA(image.Rect(0, 0, specification.HorizClksScanline, spec.ScanlinesTotal))
	}

	// allocate reflection info
	scr.crit.reflection = make([][]reflection.Reflection, specification.HorizClksScanline)
	for x := 0; x < specification.HorizClksScanline; x++ {
		scr.crit.reflection[x] = make([]reflection.Reflection, spec.ScanlinesTotal)
	}

	// create a cropped image from the main
	crop := image.Rect(
		specification.HorizClksHBlank, scr.crit.topScanline,
		specification.HorizClksHBlank+specification.HorizClksVisible, scr.crit.topScanline+scr.crit.scanlines,
	)
	scr.crit.cropPixels = scr.crit.pixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropElementPixels = scr.crit.elementPixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropOverlayPixels = scr.crit.overlayPixels.SubImage(crop).(*image.RGBA)

	// clear pixels
	for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
			scr.crit.pixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.elementPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	for i := range scr.crit.bufferPixels {
		for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
			for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
				scr.crit.bufferPixels[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

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
// MUST NOT be called from the gui thread.
func (scr *screen) Resize(spec specification.Spec, topScanline int, visibleScanlines int) error {
	scr.img.service <- func() {
		scr.resize(spec, topScanline, visibleScanlines)
		scr.img.serviceErr <- nil
	}
	return <-scr.img.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the gui thread.
func (scr *screen) NewFrame(isStable bool) error {
	// unlocking must be done carefully
	scr.crit.section.Lock()

	scr.crit.isStable = isStable

	if scr.img.isPlaymode() {
		scr.crit.plotIdx++
		if scr.crit.plotIdx >= len(scr.crit.bufferPixels) {
			scr.crit.plotIdx = 0
		}

		scr.crit.bufferUpdate = true

		// if plot index has crashed into the render index then
		if scr.crit.plotIdx == scr.crit.renderIdx && scr.crit.vsync {
			// we must unlock the critical section or the gui thread will not
			// be able to process the channel
			scr.crit.section.Unlock()

			scr.emuWait <- true
			<-scr.emuWaitAck
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

	if !scr.img.isPlaymode() {
		scr.crit.bufferUpdate = true
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
		scr.crit.lastX = sig.HorizPos
		scr.crit.lastY = sig.Scanline
	}

	scr.crit.bufferPixels[scr.crit.plotIdx].SetRGBA(sig.HorizPos, sig.Scanline, col)

	return nil
}

// Reset implements the television.PixelRenderer interface.
func (scr *screen) Reset() {
	scr.crit.section.Lock()

	// simplest method of resetting all pixels to black
	for i := range scr.crit.bufferPixels {
		for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
			for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
				scr.crit.bufferPixels[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	scr.crit.section.Unlock()
}

// EndRendering implements the television.PixelRenderer interface.
func (scr *screen) EndRendering() error {
	return nil
}

// Reflect implements reflection.Renderer interface.
//
// Must only be called between calls to UpdatingPixels(true) and UpdatingPixels(false).
func (scr *screen) Reflect(ref reflection.Reflection) error {
	x := ref.TV.HorizPos
	y := ref.TV.Scanline

	// store Reflection instance
	if x < len(scr.crit.reflection) && y < len(scr.crit.reflection[x]) {
		scr.crit.reflection[x][y] = ref
	}

	// set element pixel
	rgb := reflection.PaletteElements[ref.VideoElement]
	scr.crit.elementPixels.SetRGBA(x, y, rgb)

	// write to overlay
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
func (scr *screen) plotOverlay(x, y int, ref reflection.Reflection) {
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

// texture renderers can share the underlying pixels in the screen instance.
func (scr *screen) addTextureRenderer(r textureRenderer) {
	scr.renderers = append(scr.renderers, r)
	r.resize()
}

// called by service loop.
func (scr *screen) render() {
	// not rendering if gui.state is Rewinding or GotoCoords. render will be
	// called automatically when state changes from either of these two states
	// to something else
	if scr.img.state == gui.StateRewinding || scr.img.state == gui.StateGotoCoords {
		return
	}

	// we have to be very particular about how we unlock this
	scr.crit.section.Lock()

	if !scr.crit.bufferUpdate {
		scr.crit.section.Unlock()
		return
	}

	if scr.img.isPlaymode() && scr.crit.vsync {
		// advance render index. keep note of existing index in case we
		// bump into the plotting index.
		v := scr.crit.renderIdx
		scr.crit.renderIdx++
		if scr.crit.renderIdx >= len(scr.crit.bufferPixels) {
			scr.crit.renderIdx = 0
		}

		// if render index has bumped into the plotting index then revert
		// render index
		if scr.crit.renderIdx == scr.crit.plotIdx {
			scr.crit.renderIdx = v
			scr.crit.section.Unlock()
			return
		}

		// copy render pixes to safe copy that we use to copy to the screen
		// textures
		copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
		scr.crit.section.Unlock()

		// let the emulator thread know it's okay to continue
		select {
		case <-scr.emuWait:
			scr.emuWaitAck <- true
		default:
		}
	} else {
		// for non-playmode we use the plotIdx directly, without any buffering
		copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.plotIdx].Pix)
		scr.crit.section.Unlock()
	}

	// update attached renderers
	for _, r := range scr.renderers {
		r.render()
	}
}
