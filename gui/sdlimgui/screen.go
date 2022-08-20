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

	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/television"
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
	img *SdlImgui

	crit screenCrit

	// list of renderers to call from render. renderers are added with
	// addTextureRenderer()
	renderers  []textureRenderer
	emuWait    chan bool
	emuWaitAck chan bool

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

	// the most recent frameInfo information from the television. updated via
	// Resize() and NewFrame()
	frameInfo television.FrameInfo

	// whether or not to sync with monitor refresh rate
	monitorSync bool

	// whether monitorSync is "similar" to emulated TV's refresh rate
	monitorSyncInRange bool

	// the scanline currenty used to emulate a screenroll effect. if the value
	// is zero then no screenroll is currently taking place
	// * playmode only
	screenrollScanline int

	// the number of consecutive vsyncs seen
	vsyncCount int

	// the pixels array is used in the presentation texture of the play and debug screen.
	pixels *image.RGBA

	// bufferPixels are what we plot pixels to while we wait for a frame to complete.
	// - the larger the buffer the greater the input lag
	// - the smaller the buffer the more the emulation will have to wait the
	//		screen to catch up (see emuWait and emuWaitAck channels)
	// - a five frame buffer seems good. ten frames can feel laggy
	bufferPixels []*image.RGBA

	// the length of each pixel buffer in the bufferPixels array. saves calling
	// len() multiple times needlessly
	bufferPixelsCount int

	// count of how many of the bufferPixel entries have been used. reset to
	// numBufferPixels when emulation is paused and reduced every time a new
	// bufferPixel position is used.
	//
	// it is used to prevent the renderIdx using a buffer that hasn't been used
	// recently. this is important after a series of rewind and pause states
	//
	// * playmode only
	bufferUsed int

	// which buffer we'll be plotting to and which bufffer we'll be rendering
	// from. in playmode we make sure these two indexes never meet. in
	// debugmode we plot and render from the same index, it doesn't matter.
	//
	// * in debug mode these values never change
	plotIdx       int
	renderIdx     int
	prevRenderIdx int

	// element colors and overlay colors are only used in the debugger so we
	// don't need to replicate the "backing pixels" idea.
	elementPixels *image.RGBA
	overlayPixels *image.RGBA

	// reflection is a 2d array for easier access from winDbgScr. being able to
	// index by x and y is more convenient
	reflection []reflection.ReflectedVideoStep

	// the cropped view of the screen pixels. note that these instances are
	// created through the SubImage() command and should not be written to
	// directly
	cropPixels        *image.RGBA
	cropElementPixels *image.RGBA
	cropOverlayPixels *image.RGBA

	// the selected overlay
	overlay string

	// when paused we show two adjacent frames over-and-over. this flag tracks
	// which of those frames to show
	pauseFrame bool

	// the coordinates of the most recent pixel to be set by the television
	//
	// we experimented with not having these fields and using the Scanline and
	// Clock values from LazyTV. however, for one of the purposes we want to
	// use it for, waiting for the lazy system to update causes a visible
	// artefact in the rendering of the debug screen. it's unfortunate and it
	// means PixelRenderer.SetPixels() needs an extra argument but it's worth
	// it IMO
	lastClock    int
	lastScanline int
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:        img,
		emuWait:    make(chan bool),
		emuWaitAck: make(chan bool),
	}

	scr.crit.section.Lock()

	scr.crit.overlay = reflection.OverlayLabels[reflection.OverlayNone]
	scr.crit.monitorSync = true

	// allocate memory for pixel buffers etc.
	scr.allocateBufferPixelsForTV(60)

	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// allocate reflection
	scr.crit.reflection = make([]reflection.ReflectedVideoStep, specification.AbsoluteMaxClks)

	scr.crit.section.Unlock()

	// default to NTSC. this will change on the first instance of
	scr.resize(television.NewFrameInfo(specification.SpecNTSC))
	scr.Reset()

	return scr
}

// allocateBufferPixelsForTV decides on the buffering and syncing policy of the
// screen based on the reported TV refresh rate.
func (scr *screen) allocateBufferPixelsForTV(tvRefreshRate float32) {
	var l int

	// check whether to apply monitorsync and decide on the length of the pixel buffer
	if float32(scr.img.plt.mode.RefreshRate)*1.01 >= tvRefreshRate {
		scr.crit.monitorSyncInRange = true
		l = 5
	} else {
		scr.crit.monitorSyncInRange = false
		l = 1
	}

	scr.crit.bufferPixels = make([]*image.RGBA, l)
	for i := range scr.crit.bufferPixels {
		scr.crit.bufferPixels[i] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	}

	// note length of bufferPixels
	scr.crit.bufferPixelsCount = len(scr.crit.bufferPixels[0].Pix)

	// reset indexes
	scr.crit.bufferUsed = 0
	scr.crit.plotIdx = 0
	scr.crit.renderIdx = 0
	scr.crit.prevRenderIdx = 0
}

// Reset implements the television.PixelRenderer interface. Note that Reset
// *does not* imply a Resize().
//
// called on startup and also whenever the VCS is reset, including when a new
// cartridge is inserted.
func (scr *screen) Reset() {
	// we don't call resize on screen Reset

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	scr.clearPixels()
	scr.crit.plotIdx = 0
	scr.crit.renderIdx = 0
	scr.crit.prevRenderIdx = 0
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
		}
	}
	for i := range scr.crit.bufferPixels {
		for y := 0; y < scr.crit.pixels.Bounds().Size().Y; y++ {
			for x := 0; x < scr.crit.pixels.Bounds().Size().X; x++ {
				scr.crit.bufferPixels[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// clear element pixels
	for y := 0; y < scr.crit.elementPixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.elementPixels.Bounds().Size().X; x++ {
			scr.crit.elementPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// reset reflection information
	for i := range scr.crit.reflection {
		scr.crit.reflection[i] = reflection.ReflectedVideoStep{}
	}

	for y := 0; y < scr.crit.overlayPixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.overlayPixels.Bounds().Size().X; x++ {
			// the alpha channel for the overlay pixels changes depending on
			// what the overlay is
			scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	scr.plotOverlay()
}

// resize() is called by Resize() and by NewScreen().
func (scr *screen) resize(frameInfo television.FrameInfo) {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	scr.allocateBufferPixelsForTV(frameInfo.RefreshRate)

	// do nothing if resize values are the same as previously
	if scr.crit.frameInfo.Spec.ID == frameInfo.Spec.ID &&
		scr.crit.frameInfo.VisibleTop == frameInfo.VisibleTop &&
		scr.crit.frameInfo.VisibleBottom == frameInfo.VisibleBottom {
		return
	}

	scr.crit.frameInfo = frameInfo

	// create a cropped image from the main
	crop := image.Rect(
		specification.ClksHBlank, scr.crit.frameInfo.VisibleTop,
		specification.ClksHBlank+specification.ClksVisible, scr.crit.frameInfo.VisibleBottom,
	)
	scr.crit.cropPixels = scr.crit.pixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropElementPixels = scr.crit.elementPixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropOverlayPixels = scr.crit.overlayPixels.SubImage(crop).(*image.RGBA)

	// make sure all pixels are clear
	scr.clearPixels()

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
func (scr *screen) Resize(frameInfo television.FrameInfo) error {
	scr.img.polling.service <- func() {
		scr.resize(frameInfo)
		scr.img.polling.serviceErr <- nil
	}
	return <-scr.img.polling.serviceErr
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the gui thread.
func (scr *screen) NewFrame(frameInfo television.FrameInfo) error {
	// unlocking must be done carefully
	scr.crit.section.Lock()

	// we'll store frameInfo at the same time as unlocking the critical section

	if scr.img.isPlaymode() {
		// check screen rolling if crtprefs are enabled
		if scr.img.crtPrefs.Enabled.Get().(bool) {
			syncSpeed := scr.img.crtPrefs.SyncSpeed.Get().(int)

			if !frameInfo.VSynced {
				scr.crit.vsyncCount = 0
			} else if scr.crit.vsyncCount <= syncSpeed {
				scr.crit.vsyncCount++
			}

			// while we are waiting for VSYNC to settle down apply the screen roll
			if scr.crit.vsyncCount < syncSpeed {
				// without the stable check, the screen can roll during startup
				// of many ROMs. Pitfall for example will do this.
				syncPowerOn := scr.img.crtPrefs.SyncPowerOn.Get().(bool)
				if syncPowerOn || (!syncPowerOn && scr.crit.frameInfo.Stable) {
					scr.crit.screenrollScanline += 50
					if scr.crit.screenrollScanline > specification.AbsoluteMaxScanlines {
						scr.crit.screenrollScanline -= specification.AbsoluteMaxScanlines
					}
				}
			} else if scr.crit.screenrollScanline > 0 {
				// recover from screen roll
				scr.crit.screenrollScanline *= 80
				scr.crit.screenrollScanline /= 100
			}
		}

		if scr.crit.monitorSync && scr.crit.monitorSyncInRange {
			switch scr.img.emulation.State() {
			case emulation.Rewinding:
				fallthrough
			case emulation.Paused:
				scr.crit.renderIdx = scr.crit.plotIdx
				scr.crit.prevRenderIdx = scr.crit.plotIdx
				scr.crit.bufferUsed = len(scr.crit.bufferPixels)
			case emulation.Running:
				if scr.crit.bufferUsed > 0 {
					scr.crit.bufferUsed--
				}

				scr.crit.plotIdx++
				if scr.crit.plotIdx >= len(scr.crit.bufferPixels) {
					scr.crit.plotIdx = 0
				}

				// if plot index has crashed into the render index then
				if scr.crit.plotIdx == scr.crit.renderIdx {
					// ** screen update not keeping up with emulation **

					// we must unlock the critical section or the gui thread will not
					// be able to process the channel
					scr.crit.frameInfo = frameInfo
					scr.crit.section.Unlock()

					// pause emulation until screen has caught up
					scr.emuWait <- true
					<-scr.emuWaitAck

					return nil
				}
			}
		}
	}

	scr.crit.frameInfo = frameInfo
	scr.crit.section.Unlock()

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (scr *screen) SetPixels(sig []signal.SignalAttributes, last int) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	var col color.RGBA

	// offset the pixel writes by the amount of screenroll
	offset := scr.crit.screenrollScanline * 4 * specification.ClksScanline

	for i := range sig {
		// end of pixel buffer reached but there are still signals to process.
		//
		// this can happen when screen is rolling and offset started off at a
		// value greater than zero
		if offset >= scr.crit.bufferPixelsCount {
			offset = 0
		}

		// handle VBLANK by setting pixels to black
		if sig[i]&signal.VBlank == signal.VBlank {
			col = color.RGBA{R: 0, G: 0, B: 0}
		} else {
			px := signal.ColorSignal((sig[i] & signal.Color) >> signal.ColorShift)
			col = scr.crit.frameInfo.Spec.GetColor(px)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := scr.crit.bufferPixels[scr.crit.plotIdx].Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		// alpha channel never changes

		offset += 4
	}

	scr.crit.lastScanline = last / specification.ClksScanline
	scr.crit.lastClock = last % specification.ClksScanline

	return nil
}

// EndRendering implements the television.PixelRenderer interface.
func (scr *screen) EndRendering() error {
	return nil
}

// Reflect implements reflection.Renderer interface.
func (scr *screen) Reflect(ref []reflection.ReflectedVideoStep) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	copy(scr.crit.reflection, ref)
	scr.plotOverlay()

	return nil
}

// Reflect implements reflection.Renderer interface.
func (scr *screen) plotOverlay() {
	var col color.RGBA

	// offset the pixel writes by the amount of screenroll
	offset := scr.crit.screenrollScanline * 4 * specification.ClksScanline

	for i := range scr.crit.reflection {
		// end of pixel buffer reached but there are still signals to process.
		//
		// this can happen when screen is rolling and offset started off at a
		// value greater than zero
		if offset >= len(scr.crit.overlayPixels.Pix) {
			offset = 0
		}

		// overlay pixels must set alpha channel
		col = scr.reflectionColor(&scr.crit.reflection[i])
		s := scr.crit.overlayPixels.Pix[offset : offset+4 : offset+4]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B
		s[3] = col.A

		col = altColors[scr.crit.reflection[i].VideoElement]
		s = scr.crit.elementPixels.Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}
}

// reflectionColor should be called from within a scr.crit.section Lock().
func (scr *screen) reflectionColor(ref *reflection.ReflectedVideoStep) color.RGBA {
	switch scr.crit.overlay {
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		if ref.WSYNC {
			return reflectionColors[reflection.WSYNC]
		}
	case reflection.OverlayLabels[reflection.OverlayCollision]:
		if ref.Collision.LastVideoCycle.IsCXCLR() {
			return reflectionColors[reflection.CXCLR]
		} else if !ref.Collision.LastVideoCycle.IsNothing() {
			return reflectionColors[reflection.Collision]
		}
	case reflection.OverlayLabels[reflection.OverlayHMOVE]:
		// HmoveCt counts to -1 (or 255 for a uint8)
		if ref.Hmove.Delay {
			return reflectionColors[reflection.HMOVEdelay]
		} else if ref.Hmove.Latch {
			if ref.Hmove.RippleCt != 255 {
				return reflectionColors[reflection.HMOVEripple]
			} else {
				return reflectionColors[reflection.HMOVElatched]
			}
		}
	case reflection.OverlayLabels[reflection.OverlayRSYNC]:
		if ref.RSYNCalign {
			return reflectionColors[reflection.RSYNCalign]
		} else if ref.RSYNCreset {
			return reflectionColors[reflection.RSYNCreset]
		}
	case reflection.OverlayLabels[reflection.OverlayCoproc]:
		if ref.CoprocessorActive {
			return reflectionColors[reflection.CoprocessorActive]
		}
	}

	return color.RGBA{}
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

// copy pixels from buffer to the pixels.
func (scr *screen) copyPixelsDebugmode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// copy pixels from render buffer to the live copy.
	copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.plotIdx].Pix)
}

// copy pixels from buffer to the pixels.
func (scr *screen) copyPixelsPlaymode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// the bufferUsed check is important for correct operation of the rewinding
	// state. without it, the screen will jump after a rewind event
	if scr.crit.bufferUsed == 0 && scr.crit.monitorSync && scr.crit.monitorSyncInRange {
		// advance render index
		prev := scr.crit.prevRenderIdx
		scr.crit.prevRenderIdx = scr.crit.renderIdx
		scr.crit.renderIdx++
		if scr.crit.renderIdx >= len(scr.crit.bufferPixels) {
			scr.crit.renderIdx = 0
		}

		// render index has bumped into the plotting index. revert render index
		if scr.crit.renderIdx == scr.crit.plotIdx {
			// ** emulation not keeping up with screen update **

			// undo frame advancement
			scr.crit.renderIdx = scr.crit.prevRenderIdx
			scr.crit.prevRenderIdx = prev
		}

		// let the emulator thread know it's okay to continue as soon as possible
		select {
		case <-scr.emuWait:
			scr.emuWaitAck <- true
		default:
		}
	}

	// copy pixels from render buffer to the live copy.
	//
	// if emulation is paused we alternate which set of pixels we copy.
	// currently, we assume that the display kernel is two frames.
	//
	// this assumption works well in most instances but will sometimes cause
	// poor results for one frame kernels (depending on the ROM and what is
	// happening on the screen at the time of the pause ) and will be
	// sub-optimal for three frame kernels in almost all cases.
	if scr.img.emulation.State() == emulation.Paused {
		activePause := scr.img.prefs.activePause.Get().(bool)
		if scr.crit.pauseFrame || !activePause {
			copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
		} else {
			copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.prevRenderIdx].Pix)
		}
		scr.crit.pauseFrame = !scr.crit.pauseFrame
	} else {
		copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
	}
}
