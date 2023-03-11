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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
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

const maxFrameQueue = 10
const frameQueueIncDelta = 20
const frameQueueDecDelta = 1
const frameQueueIncVal = 40

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

	// the presentationPixels array is used in the presentation texture of the play and debug screen.
	presentationPixels *image.RGBA

	// frameQueue are what we plot pixels to while we wait for a frame to complete.
	// - the larger the queue the greater the input lag
	// - the smaller the queue the more the emulation will have to wait the
	//		screen to catch up (see emuWait and emuWaitAck channels)
	// - a three to five frame queue seems good. ten frames can feel laggy
	frameQueue     [maxFrameQueue]*image.RGBA
	frameQueueLen  int
	frameQueueAuto bool

	// frame queue count is increased on every dropped frame and decreased on
	// every on time frame. when it reaches a predetermined threshold the
	// length of the frame queue is increased (up to a maximum value)
	frameQueueIncCt int

	// number of pixels (multiplied by the pixel depth) in each entry of the
	// pixels queue. saves calling len() multiple times needlessly
	pixelsCount int

	// count of how many of entries in the pixel queue have been used. reset to
	// length of pixels queue when emulation is paused and reduced every time a
	// new entry is used
	//
	// it is used to prevent the renderIdx using queue entries that haven't
	// been used recently. this is important after a series of rewind and pause
	// states
	//
	// * playmode only
	queueUsed int

	// which entry in the queue we'll be plotting to and which one we'll be
	// rendering from. in playmode we make sure these two indexes never meet.
	// in debugmode we plot and render from the same index, it doesn't matter.
	//
	// * in debug mode these values never change
	plotIdx       int
	renderIdx     int
	plotIdxNext   int
	renderIdxNext int

	//  the previous render index value is used to help smooth frame queue collisions
	prevRenderIdx int

	// element colors and overlay colors are only used in the debugger
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

	scr.crit.presentationPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// allocate reflection
	scr.crit.reflection = make([]reflection.ReflectedVideoStep, specification.AbsoluteMaxClks)

	// allocate frame queue images
	for i := range scr.crit.frameQueue {
		scr.crit.frameQueue[i] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	}

	// note number of pixels in each entry in the pixel queue
	scr.crit.pixelsCount = len(scr.crit.frameQueue[0].Pix)

	scr.crit.section.Unlock()

	// default to NTSC. this will change on the first instance of
	scr.resize(television.NewFrameInfo(specification.SpecNTSC))
	scr.Reset()

	return scr
}

// setRefreshRate decides on the buffering and syncing policy of the
// screen based on the reported TV refresh rate.
//
// must be called from inside a critical section.
func (scr *screen) setRefreshRate(tvRefreshRate float32) {
	// check whether to apply monitorsync and decide on the length of the pixel queue
	scr.crit.monitorSyncInRange = float32(scr.img.plt.mode.RefreshRate)*1.01 >= tvRefreshRate
	scr.setFrameQueue()
}

// must be called from inside a critical section.
func (scr *screen) setFrameQueue() {
	scr.crit.frameQueueAuto = scr.img.prefs.frameQueueAuto.Get().(bool)
	scr.crit.frameQueueLen = scr.img.prefs.frameQueue.Get().(int)

	scr.resetFrameQueue()

	// restore previous render frame to all entries in the queue to ensure we
	// never get an empty frame by accident
	old := scr.crit.frameQueue[scr.crit.renderIdx]
	if old != nil {
		for i := range scr.crit.frameQueue {
			copy(scr.crit.frameQueue[i].Pix, old.Pix)
		}
	}
}

// must be called from inside a critical section.
func (scr *screen) resetFrameQueue() {
	scr.crit.queueUsed = 0
	scr.crit.plotIdx = 0
	scr.crit.renderIdx = scr.crit.frameQueueLen / 2

	scr.crit.plotIdxNext = scr.crit.plotIdx + 1
	if scr.crit.plotIdxNext >= scr.crit.frameQueueLen {
		scr.crit.plotIdx = 0
	}

	scr.crit.renderIdxNext = scr.crit.renderIdx + 1
	if scr.crit.renderIdxNext >= scr.crit.frameQueueLen {
		scr.crit.renderIdxNext = 0
	}
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
	scr.resetFrameQueue()
}

// clear pixel information including reflection data. important to do on
// initialisation because we need to set the alpha channel (because the alpha
// value never changes we then don't need to write it over and over again in
// the SetPixels() function)
//
// must be called from inside a critical section.
func (scr *screen) clearPixels() {
	// clear pixels in frame queue
	for i := range scr.crit.frameQueue {
		for y := 0; y < scr.crit.presentationPixels.Bounds().Size().Y; y++ {
			for x := 0; x < scr.crit.presentationPixels.Bounds().Size().X; x++ {
				scr.crit.frameQueue[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// clear pixels
	for y := 0; y < scr.crit.presentationPixels.Bounds().Size().Y; y++ {
		for x := 0; x < scr.crit.presentationPixels.Bounds().Size().X; x++ {
			scr.crit.presentationPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.elementPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
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

	scr.setRefreshRate(frameInfo.RefreshRate)

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
	scr.crit.cropPixels = scr.crit.presentationPixels.SubImage(crop).(*image.RGBA)
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
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// we'll store frameInfo at the same time as unlocking the critical section

	if scr.img.isPlaymode() {
		// check screen rolling if crtprefs are enabled
		if scr.img.crtPrefs.Enabled.Get().(bool) {
			syncSpeed := scr.img.crtPrefs.SyncSpeed.Get().(int)

			if !frameInfo.VSync || frameInfo.VSyncScanlines < scr.img.crtPrefs.SyncSensitivity.Get().(int) {
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

	}

	scr.crit.frameInfo = frameInfo

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (scr *screen) SetPixels(sig []signal.SignalAttributes, last int) error {
	// wait flag indicates whether to slow down the emulation
	// * will only be set in playmode
	wait := false

	// unlocking must be done carefully
	scr.crit.section.Lock()

	if scr.img.isPlaymode() {
		if scr.crit.monitorSync && scr.crit.monitorSyncInRange {
			switch scr.img.dbg.State() {
			case govern.Rewinding:
				fallthrough
			case govern.Paused:
				scr.crit.renderIdx = scr.crit.plotIdx
				scr.crit.queueUsed = scr.crit.frameQueueLen
			case govern.Running:
				if scr.crit.queueUsed > 0 {
					scr.crit.queueUsed--
				}

				scr.crit.plotIdx = scr.crit.plotIdxNext
				scr.crit.plotIdxNext++
				if scr.crit.plotIdxNext >= scr.crit.frameQueueLen {
					scr.crit.plotIdxNext = 0
				}

				// if plot index has crashed into the render index then set wait flag
				// ** screen update not keeping up with emulation **
				wait = scr.crit.plotIdx == scr.crit.renderIdx && scr.crit.frameQueueLen > 1
			}
		}
	}

	var col color.RGBA

	// offset the pixel writes by the amount of screenroll
	offset := scr.crit.screenrollScanline * 4 * specification.ClksScanline

	for i := range sig {
		// end of pixel queue reached but there are still signals to process.
		//
		// this can happen when screen is rolling and the initial offset value
		// was greater than zero
		if offset >= scr.crit.pixelsCount {
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
		s := scr.crit.frameQueue[scr.crit.plotIdx].Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		// alpha channel never changes

		offset += 4
	}

	scr.crit.lastScanline = last / specification.ClksScanline
	scr.crit.lastClock = last % specification.ClksScanline

	scr.crit.section.Unlock()

	// slow emulation until screen has caught up
	// * wait should only be set to true in playmode
	if wait {
		scr.emuWait <- true
		<-scr.emuWaitAck
	}

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

func (scr *screen) plotOverlay() {
	var col color.RGBA

	// offset the pixel writes by the amount of screenroll
	offset := scr.crit.screenrollScanline * 4 * specification.ClksScanline

	for i := range scr.crit.reflection {
		// end of pixel queue reached but there are still signals to process.
		//
		// this can happen when screen is rolling and the initial offset value
		// was greater than zero
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
	case reflection.OverlayLabels[reflection.OverlayAudio]:
		if ref.AudioChanged {
			return reflectionColors[reflection.AudioChanged]
		} else if ref.AudioPhase0 {
			return reflectionColors[reflection.AudioPhase0]
		} else if ref.AudioPhase1 {
			return reflectionColors[reflection.AudioPhase1]
		}
	case reflection.OverlayLabels[reflection.OverlayCoproc]:
		switch ref.CoProcState {
		case mapper.CoProcIdle:
			return reflectionColors[reflection.CoProcInactive]
		case mapper.CoProcNOPFeed:
			return reflectionColors[reflection.CoProcActive]
		case mapper.CoProcStrongARMFeed:
			return reflectionColors[reflection.CoProcInactive]
		case mapper.CoProcParallel:
			return reflectionColors[reflection.CoProcActive]
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

// copy pixels from queue to the live copy.
func (scr *screen) copyPixelsDebugmode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// copy pixels from pixel queue to the live copy.
	copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.plotIdx].Pix)
}

// copy pixels from queue to the live copy.
func (scr *screen) copyPixelsPlaymode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// let the emulator thread know it's okay to continue as soon as possible
	select {
	case <-scr.emuWait:
		scr.emuWaitAck <- true
	default:
	}

	// show pause frames
	if scr.img.dbg.State() == govern.Paused {
		if scr.img.prefs.activePause.Get().(bool) {
			if scr.crit.pauseFrame {
				copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.prevRenderIdx].Pix)
				scr.crit.pauseFrame = false
			} else {
				copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)
				scr.crit.pauseFrame = true
			}
		} else {
			copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)
		}
		return
	}

	// the bufferUsed check is important for correct operation of the rewinding
	// state. without it, the screen will jump after a rewind event
	if scr.crit.queueUsed == 0 && scr.crit.monitorSync && scr.crit.monitorSyncInRange {
		// advance render index
		prev := scr.crit.prevRenderIdx
		scr.crit.prevRenderIdx = scr.crit.renderIdx
		scr.crit.renderIdx++
		if scr.crit.renderIdx >= scr.crit.frameQueueLen {
			scr.crit.renderIdx = 0
		}

		// render index has bumped into the plotting index. revert render index
		if scr.crit.renderIdx == scr.crit.plotIdx {
			// ** emulation not keeping up with screen update **

			// undo frame advancement
			scr.crit.renderIdx = scr.crit.prevRenderIdx
			scr.crit.prevRenderIdx = prev

			// adjust frame queue increase counter
			if scr.crit.frameQueueAuto && scr.crit.frameInfo.Stable && scr.crit.frameQueueLen < maxFrameQueue {
				scr.crit.frameQueueIncCt += frameQueueIncDelta
				if scr.crit.frameQueueIncCt >= frameQueueIncVal {
					scr.crit.frameQueueLen++
				}
			}
		} else {
			// adjust frame queue increase counter
			if scr.crit.frameQueueIncCt > 0 {
				scr.crit.frameQueueIncCt -= frameQueueDecDelta
			}
		}

	}

	copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)
}
