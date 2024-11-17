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
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/reflection"
)

// alllow nudging of the TV. nudging causes the emulation to uncap itself for a
// short number of frames. the benefit of this is of dubious value so it has
// been disabled for now. maybe removed in the future
const allowNudging = false

// textureRenderers should consider that the timing of the VCS produces
// "pixels" of two pixels across.
//
// this value can be ignored if texture is being drawn in a fixed aperture. for
// example when using specification.WidthTV and specification.HeightTV
const pixelWidth = 2

// textureRenderers can share the underlying pixels of the screen type instance. both of these functions
// should be called inside the screen critical section.
type textureRenderer interface {
	render()
	resize()
	updateRefreshRate()
}

// screen implements television.PixelRenderer.
type screen struct {
	img *SdlImgui

	crit screenCrit

	// atmoic access of rotation value. it's more convenient to be able to
	// access this atomically, rather than via the screenCrit type
	rotation atomic.Value // specification.Rotation

	// list of renderers to call from render. renderers are added with
	// addTextureRenderer()
	renderers  []textureRenderer
	emuWait    chan bool
	emuWaitAck chan bool

	// the mouse coords used in the most recent call to PushGotoCoords(). only
	// read/write by the GUI thread so doesn't need to be in critical section.
	gotoCoordsX int
	gotoCoordsY int

	// the number of frames the emulation currently is ahead
	frameQueueSlack int
}

const (
	// these values indicate that the frame queue should be extended when there
	// are two consecutive frames when the frame queue is full
	frameQueueAutoInc          = 20
	frameQueueAutoIncThreshold = 40

	// a small decay value means that a flurry of slow frames, that are not
	// necessarily consecutive, will be detected and cause the queuen to grow
	frameQueueAutoIncDecay = 1

	// the frame queue should never be longer than this
	maxFrameQueue = 10
)

// for clarity, variables accessed in the critical section are encapsulated in
// their own subtype.
type screenCrit struct {
	// critical sectioning
	section sync.Mutex

	// the most recent frameInfo information from the television. information
	// received via the NewFrame() function
	frameInfo television.FrameInfo

	// screen will resize on next GUI iteration if resize is true. if resizeHold
	// is true however, the resize will be delayed until the current state is no
	// longer govern.Rewinding
	resize bool

	// whether or not emulation is fps capped (to speed of television). the
	// screen implementation uses this to decide whether to sync with monitor
	// refresh rate
	fpsCapped bool

	// whether monitor refresh rate is higher than the emulated TV's refresh rate
	monitorSyncHigher bool

	// whether monitor refresh rate is similar to the emulated TV's refresh rate
	monitorSyncSimilar bool

	// the presentationPixels array is used in the presentation texture of the play and debug screen.
	presentationPixels *image.RGBA

	// frameQueue are what we plot pixels to while we wait for a frame to complete.
	// - the larger the queue the greater the input lag
	// - a three to five frame queue seems good. ten frames can feel very laggy
	frameQueue [maxFrameQueue]*image.RGBA

	// local copies of frame queue preferences values. updated by setFrameQueue()
	// which is called when the underlying preference is changed
	prefsFrameQueueAuto bool
	prefsFrameQueueLen  int

	// the actual frame queue value depends on whether FPS is capped or not
	frameQueueAuto bool
	frameQueueLen  int

	// frame queue count is increased on every dropped frame and decreased on
	// every on-time frame. when it reaches a predetermined threshold the
	// length of the frame queue is increased (up to a maximum value). see
	// frameQueueInc* constant values
	frameQueueIncCt int

	// the number of frames after rewinding/pausing before resuming "normal"
	// queue traversal
	//
	// it is used to prevent the renderIdx using queue entries that haven't
	// been used recently
	//
	// * playmode only
	queueRecovery int

	// which entry in the queue we'll be plotting to and which one we'll be
	// rendering from. in playmode we make sure these two indexes never meet.
	// in debugmode we plot and render from the same index, it doesn't matter.
	//
	// * in debug mode these values never change
	plotIdx   int
	renderIdx int

	//  the previous render index value is used to help smooth frame queue collisions
	prevRenderIdx [2]int

	// element colors and overlay colors are only used in the debugger
	elementPixels *image.RGBA
	overlayPixels *image.RGBA

	// reflection is a 2d array for easier access from winDbgScr. being able to
	// index by x and y is more convenient
	reflection []reflection.ReflectedVideoStep

	// the cropped view of the screen pixels. note that these instances are
	// created through the SubImage() command and should not be written to
	// directly
	cropPixels           *image.RGBA
	cropElementPixels    *image.RGBA
	cropOverlayPixels    *image.RGBA
	cropScreenrollPixels *image.RGBA

	// the selected overlay
	overlay string

	// whether the overaly color should be striped
	overlayStriped bool

	// when paused we show two adjacent frames over-and-over. this flag tracks
	// which of those frames to show
	pauseFrame bool

	// the X, Y coordinates of the most recent pixel to be updated this are not
	// the same values that would be in coords.TelevisionCoords
	lastX int
	lastY int
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:        img,
		emuWait:    make(chan bool),
		emuWaitAck: make(chan bool),
	}

	scr.rotation.Store(specification.NormalRotation)
	scr.crit.section.Lock()

	scr.crit.overlay = reflection.OverlayLabels[reflection.OverlayNone]
	scr.crit.fpsCapped = true

	scr.crit.presentationPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// allocate reflection
	scr.crit.reflection = make([]reflection.ReflectedVideoStep, specification.AbsoluteMaxClks)

	// allocate frame queue images
	for i := range scr.crit.frameQueue {
		scr.crit.frameQueue[i] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	}

	scr.crit.section.Unlock()

	// default to NTSC
	scr.crit.frameInfo = television.NewFrameInfo(specification.SpecNTSC)
	scr.crit.resize = true
	scr.resize()
	scr.Reset()

	return scr
}

// SetRotation implements the television.PixelRendererRotation interface
func (scr *screen) SetRotation(rotation specification.Rotation) {
	scr.rotation.Store(rotation)

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// the only other component that needs to be aware of the rotation is the
	// play screen
	for _, r := range scr.renderers {
		if t, ok := r.(television.PixelRendererRotation); ok {
			t.SetRotation(rotation)
		}
	}
}

// SetFPSCap implements the television.PixelRendererFPSCap interface
func (scr *screen) SetFPSCap(limit bool) {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	scr.crit.fpsCapped = limit
	scr.setSyncPolicy(scr.crit.frameInfo.RefreshRate)
	scr.updateFrameQueue()
}

// setSyncPolicy decides on the monitor/tv syncing policy, based on the
// TV refresh rate and the refresh rate of the monitor
//
// must be called from inside a critical section.
func (scr *screen) setSyncPolicy(tvRefreshRate float32) {
	high := float32(scr.img.plt.mode.RefreshRate) * 1.02
	low := float32(scr.img.plt.mode.RefreshRate) * 0.98

	scr.crit.monitorSyncHigher = scr.crit.fpsCapped && high >= tvRefreshRate
	scr.crit.monitorSyncSimilar = scr.crit.fpsCapped && high >= tvRefreshRate && low <= tvRefreshRate
}

// setFrameQueue is called when frame queue preferences are changed. the screen
// type keeps its own copies of these values
//
// must NOT be called from inside a critical section
func (scr *screen) setFrameQueue(auto bool, length int) {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()
	scr.crit.prefsFrameQueueAuto = auto
	scr.crit.prefsFrameQueueLen = length
	scr.updateFrameQueue()
}

// updateFrameQueue() after a reset of a frame queue value is changed
//
// must be called from inside a critical section
func (scr *screen) updateFrameQueue() {
	if scr.img.dbg.State() == govern.Rewinding {
		return
	}

	// make local copies of frame queue preferences if fpsCapped is enabled
	if scr.crit.fpsCapped {
		scr.crit.frameQueueAuto = scr.crit.prefsFrameQueueAuto
		if scr.crit.frameQueueAuto {
			scr.crit.frameQueueLen = 1
		} else {
			scr.crit.frameQueueLen = scr.crit.prefsFrameQueueLen
		}
	} else {
		scr.crit.frameQueueAuto = false
		scr.crit.frameQueueLen = 1
	}

	// set queue recovery
	scr.crit.queueRecovery = scr.crit.frameQueueLen

	// restore previous plot frame to all entries in the queue
	var old *image.RGBA
	switch scr.img.dbg.Mode() {
	case govern.ModePlay:
		old = scr.crit.frameQueue[scr.crit.renderIdx]
	case govern.ModeDebugger:
		old = scr.crit.frameQueue[scr.crit.plotIdx]
	}
	if old != nil {
		for i := range scr.crit.frameQueue {
			copy(scr.crit.frameQueue[i].Pix, old.Pix)
		}
	}

	scr.crit.plotIdx = 0
	scr.crit.renderIdx = 0
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

	scr.setSyncPolicy(scr.crit.frameInfo.RefreshRate)
	scr.clearPixels()
	scr.updateFrameQueue()
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

// resize screen if flag has been set during NewFrame(). called from render()
func (scr *screen) resize() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// do nothing if resize flag is not set
	if !scr.crit.resize {
		return
	}

	// do not resize if emulation is the debugger mode and rewinding state
	//
	// this prevents an ugly flickering of the debug screen when the user is
	// screen-rewinding on a resize boundary
	//
	// the cursor/painting-effect still flickers but it's less annoying than the
	// entire screen flickering
	if scr.img.dbg.Mode() == govern.ModeDebugger && scr.img.dbg.State() == govern.Rewinding {
		return
	}

	scr.crit.resize = false

	// create cropped image(s)
	crop := scr.crit.frameInfo.Crop()
	scr.crit.cropPixels = scr.crit.presentationPixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropElementPixels = scr.crit.elementPixels.SubImage(crop).(*image.RGBA)
	scr.crit.cropOverlayPixels = scr.crit.overlayPixels.SubImage(crop).(*image.RGBA)

	// update frame queue
	if scr.img.dbg.Mode() == govern.ModePlay {
		scr.updateFrameQueue()
	}

	// resize texture renderers
	for _, r := range scr.renderers {
		r.resize()
	}
}

// NewFrame implements the television.PixelRenderer interface
//
// MUST NOT be called from the gui thread.
func (scr *screen) NewFrame(frameInfo television.FrameInfo) error {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// set sync policy if refresh rate has changed
	if scr.crit.frameInfo.RefreshRate != frameInfo.RefreshRate {
		scr.setSyncPolicy(frameInfo.RefreshRate)
		for _, r := range scr.renderers {
			r.updateRefreshRate()
		}
	}

	// check if screen needs to be resized
	//
	// note that we're only signalling that a resize should take place. it will
	// be reset to false in the resize() function
	//
	// we also limit the resize to when the television is stable in order to
	// prevent ugly screen updates while the resizer is finding the correct
	// size. in the case of PAL60 ROMs in particular, the screen can resize
	// dramatically
	scr.crit.resize = scr.crit.resize || scr.crit.frameInfo.IsDifferent(frameInfo)

	// record frame info
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
		switch scr.img.dbg.State() {
		case govern.Rewinding:
			fallthrough
		case govern.Paused:
			scr.crit.queueRecovery = scr.crit.frameQueueLen
			scr.crit.renderIdx = scr.crit.plotIdx
		case govern.Running:
			if scr.crit.queueRecovery > 0 {
				scr.crit.queueRecovery--
			}

			scr.crit.plotIdx++
			if scr.crit.plotIdx >= scr.crit.frameQueueLen {
				scr.crit.plotIdx = 0
			}

			// if plot index has crashed into the render index then set wait flag
			// ** screen update not keeping up with emulation **
			wait = scr.crit.plotIdx == scr.crit.renderIdx && scr.crit.frameQueueLen > 2
		}
	}

	var col color.RGBA
	var offset int

	for i := range sig {
		// end of pixel queue reached but there are still signals to process.
		// return immediately to simplify bounds issues. this shouldn't ever be
		// an issue
		if offset >= len(scr.crit.frameQueue[0].Pix) {
			return nil
		}

		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = scr.crit.frameInfo.Spec.GetColor(signal.VideoBlack)
		} else {
			col = scr.crit.frameInfo.Spec.GetColor(sig[i].Color)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := scr.crit.frameQueue[scr.crit.plotIdx].Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		// alpha channel never changes

		offset += 4
	}

	scr.crit.lastY = last / specification.ClksScanline
	scr.crit.lastX = last % specification.ClksScanline

	scr.crit.section.Unlock()

	// slow emulation until screen has caught up
	// * wait should only be set to true in playmode
	if wait && scr.crit.monitorSyncHigher {
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
	var offset int

	for i := range scr.crit.reflection {
		// end of pixel queue reached but there are still signals to process.
		// return immediately to simplify bounds issues. this shouldn't ever be
		// an issue
		if offset >= len(scr.crit.overlayPixels.Pix) {
			return
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
	case reflection.OverlayLabels[reflection.OverlayVBLANK_VSYNC]:
		if ref.Signal.VBlank {
			return reflectionColors[reflection.VBLANK]
		}
		if ref.Signal.VSync {
			return reflectionColors[reflection.VSYNC]
		}
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		if ref.WSYNC {
			return reflectionColors[reflection.WSYNC]
		}
	case reflection.OverlayLabels[reflection.OverlayCollision]:
		if ref.Collision.LastColorClock.IsCXCLR() {
			return reflectionColors[reflection.CXCLR]
		} else if !ref.Collision.LastColorClock.IsNothing() {
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
		switch ref.CoProcSync {
		case coprocessor.CoProcIdle:
			return reflectionColors[reflection.CoProcInactive]
		case coprocessor.CoProcNOPFeed:
			return reflectionColors[reflection.CoProcActive]
		case coprocessor.CoProcStrongARMFeed:
			return reflectionColors[reflection.CoProcInactive]
		case coprocessor.CoProcParallel:
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

// called by service loop
func (scr *screen) render() {
	scr.resize()

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
	// let the emulator thread know it's okay to continue as soon as possible
	select {
	case <-scr.emuWait:
		scr.emuWaitAck <- true
	default:
	}

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// whether to nudge the TV or not
	var nudge bool

	// whether the frame queue can tolerate nudging. the value can/should be
	// supplemented with the monitorSyncHigher or monitorSyncSlower value
	canNudge := allowNudging && scr.crit.frameInfo.Stable && scr.crit.frameQueueLen > 2

	// measure frame queue slack
	scr.frameQueueSlack = (scr.crit.renderIdx - scr.crit.plotIdx + scr.crit.frameQueueLen) % scr.crit.frameQueueLen

	// set nudge to true if frame slack is too small
	if canNudge && scr.crit.monitorSyncSimilar {
		if scr.frameQueueSlack < scr.crit.frameQueueLen/2 {
			nudge = true
		}
	}

	// show pause frames
	if scr.img.dbg.State() == govern.Paused {
		if scr.crit.queueRecovery == 0 && scr.img.prefs.activePause.Get().(bool) {
			if scr.crit.pauseFrame {
				copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.prevRenderIdx[0]].Pix)
				scr.crit.pauseFrame = false
			} else {
				copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)
				scr.crit.pauseFrame = true
			}
		} else {
			// non-active pausing can end the function immediately
			copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)
			return
		}
	}

	// the queueRecovery check is important for correct operation of the rewinding
	// state. without it, the screen will jump after a rewind event
	if scr.crit.queueRecovery == 0 {
		// keep a small history of prev render indexes
		scr.crit.prevRenderIdx[1] = scr.crit.prevRenderIdx[0]
		scr.crit.prevRenderIdx[0] = scr.crit.renderIdx

		// advance render index
		scr.crit.renderIdx++
		if scr.crit.renderIdx >= scr.crit.frameQueueLen {
			scr.crit.renderIdx = 0
		}

		// render index has bumped into the plotting index. revert render index
		if scr.crit.renderIdx == scr.crit.plotIdx {
			// ** emulation not keeping up with screen update **

			// undo frame advancement by restoring an older index
			//
			// by choosing the frame previous to the frame shown on the last
			// render, we are smoothing out graphical glitches caused by flicker
			// kernels
			//
			// this may mean that there will be visible jump of graphical
			// elements in some situations. but hopefully they are not
			// noticeable
			scr.crit.renderIdx = scr.crit.prevRenderIdx[1]

			// set nudge flag
			nudge = canNudge && scr.crit.monitorSyncSimilar

			// adjust frame queue increase counter
			if scr.crit.frameQueueAuto && scr.crit.frameInfo.Stable && scr.crit.frameQueueLen < maxFrameQueue {
				scr.crit.frameQueueIncCt += frameQueueAutoInc
				if scr.crit.frameQueueIncCt >= frameQueueAutoIncThreshold {
					scr.crit.frameQueueIncCt = 0

					// increase frame queue length depending on whether monitor
					// sync is similar to TV. if it is not we limit the frame
					// queue length to the manual preference value
					//
					// this prevents, for example, queues immediately growing to
					// a maximum size for PAL50 TVs on 60Hz monitors
					if scr.crit.monitorSyncSimilar {
						scr.crit.frameQueueLen++
					} else {
						if scr.crit.frameQueueLen < scr.crit.prefsFrameQueueLen {
							scr.crit.frameQueueLen++
						}
					}
				}
			}
		} else {
			// adjust frame queue increase counter
			if scr.crit.frameQueueIncCt > 0 {
				scr.crit.frameQueueIncCt -= frameQueueAutoIncDecay
			}
		}
	}

	copy(scr.crit.presentationPixels.Pix, scr.crit.frameQueue[scr.crit.renderIdx].Pix)

	// nudge fps cap to try to bring the plot and render indexes back into equilibrium
	if nudge {
		scr.img.dbg.VCS().TV.NudgeFPSCap(1)
	}
}
