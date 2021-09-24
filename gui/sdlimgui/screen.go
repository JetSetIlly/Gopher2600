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

	// the number of consecutive frames where a screenroll might have happened.
	// once it reaches a threshold then screenroll begins
	// * playmode only
	screenrollCt int

	// the scanline currenty used to emulate a screenroll effect. if the value
	// is zero then no screenroll is currently taking place
	// * playmode only
	screenrollScanline int

	// the pixels array is used in the presentation texture of the play and debug screen.
	pixels *image.RGBA

	// bufferPixels are what we plot pixels to while we wait for a frame to complete.
	// - the larger the buffer the greater the input lag
	// - the smaller the buffer the more the emulation will have to wait the
	//		screen to catch up (see emuWait and emuWaitAck channels)
	// - a five frame buffer seems good. ten frames can feel laggy
	bufferPixels [5]*image.RGBA

	// the number of scanlines represented in the bufferPixels. this is set
	// during a resize operation and used to affect screen roll visualisation
	bufferHeight int

	// which buffer we'll be plotting to and which bufffer we'll be rendering
	// from. in playmode we make sure these two indexes never meet. in
	// debugmode we plot and render from the same index, it doesn't matter.
	plotIdx       int
	renderIdx     int
	prevRenderIdx int

	// element colors and overlay colors are only used in the debugger so we
	// don't need to replicate the "backing pixels" idea.
	elementPixels *image.RGBA
	overlayPixels *image.RGBA

	// 2d array of disasm entries. resized at the same time as overlayPixels resize
	reflection [][]reflection.ReflectedVideoStep

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

	// the selected overlay
	overlay string

	// when paused we show two adjacent frames over-and-over. this flag tracks
	// which of those frames to show
	pauseFrame bool
}

func newScreen(img *SdlImgui) *screen {
	scr := &screen{
		img:        img,
		emuWait:    make(chan bool),
		emuWaitAck: make(chan bool),
	}

	scr.crit.overlay = reflection.OverlayLabels[reflection.OverlayNone]
	scr.crit.monitorSync = true

	// default to NTSC. this will change on the first instance of
	scr.resize(television.NewFrameInfo(specification.SpecNTSC))
	scr.Reset()

	return scr
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
			scr.crit.reflection[i][j] = reflection.ReflectedVideoStep{}
		}
	}
	scr.replotOverlay()
}

// resize() is called by Resize() and by NewScreen().
func (scr *screen) resize(frameInfo television.FrameInfo) {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// do nothing if resize values are the same as previously
	if scr.crit.frameInfo.Spec.ID == frameInfo.Spec.ID &&
		scr.crit.frameInfo.VisibleTop == frameInfo.VisibleTop &&
		scr.crit.frameInfo.VisibleBottom == frameInfo.VisibleBottom {
		return
	}

	scr.crit.frameInfo = frameInfo

	// reallocate memory for new spec. the amount of vertical space is the
	// MaxScanlines value. the +1 is so that we can draw the debugging cursor
	// at the limit of the screen.
	scr.crit.bufferHeight = scr.crit.frameInfo.Spec.ScanlinesTotal + 1
	scr.crit.pixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, scr.crit.bufferHeight))
	scr.crit.elementPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, scr.crit.bufferHeight))
	scr.crit.overlayPixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, scr.crit.bufferHeight))

	for i := range scr.crit.bufferPixels {
		scr.crit.bufferPixels[i] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, scr.crit.bufferHeight))
	}

	// allocate reflection info
	scr.crit.reflection = make([][]reflection.ReflectedVideoStep, specification.ClksScanline)
	for x := 0; x < specification.ClksScanline; x++ {
		scr.crit.reflection[x] = make([]reflection.ReflectedVideoStep, scr.crit.bufferHeight)
	}

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

	defer func() {
		scr.crit.frameInfo = frameInfo
	}()

	if scr.img.isPlaymode() {
		// check screen rolling if crtprefs are enabled
		if scr.img.crtPrefs.Enabled.Get().(bool) {
			if frameInfo.RefreshRate == scr.crit.frameInfo.RefreshRate && frameInfo.VSynced {
				// recovery required
				if scr.crit.screenrollScanline > 0 {
					scr.crit.screenrollCt = 0
					scr.crit.screenrollScanline *= 8
					scr.crit.screenrollScanline /= 10
				}
			} else if scr.crit.frameInfo.Stable {
				// without the stable check, the screen can desync and recover
				// from a roll on startup on most ROMs, which looks quite cool but
				// we'll leave it disabled for now.

				scr.crit.screenrollCt++
				if scr.crit.screenrollCt > scr.img.crtPrefs.UnsyncTolerance.Get().(int) {
					scr.crit.screenrollScanline = (scr.crit.screenrollScanline + scr.crit.lastY)
					if scr.crit.screenrollScanline >= specification.AbsoluteMaxScanlines {
						scr.crit.screenrollScanline -= specification.AbsoluteMaxScanlines
					}
				}
			}
		} else {
			scr.crit.screenrollScanline = 0
		}

		if scr.img.emulation.State() == emulation.Running {
			scr.crit.plotIdx++
			if scr.crit.plotIdx >= len(scr.crit.bufferPixels) {
				scr.crit.plotIdx = 0
			}

			// if plot index has crashed into the render index then
			if scr.crit.plotIdx == scr.crit.renderIdx && scr.crit.monitorSync {
				// ** screen update not keeping up with emulation **

				// we must unlock the critical section or the gui thread will not
				// be able to process the channel
				scr.crit.section.Unlock()

				// pause emulation until screen has caught up
				scr.emuWait <- true
				<-scr.emuWaitAck

				return nil
			}
		}
	} else if scr.img.emulation.State() != emulation.Rewinding {
		// clear reflection pixels beyond the last X/Y plot
		//
		// this is hardly ever required but it is important for consistent
		// reflection feedback if a frame is smaller than any previous frame.
		//
		// for example, during ROM initialisation, the limits of a frame might
		// be far beyond normal, meaning reflection data from that phase will
		// remain for the duration of the execution.
		//
		// note that we don't do this if the current gui state is
		// StateRewinding. this is because in the case of GotoCoords() the
		// lastX and lastY values are probably misleading for this purpose (the
		// emulation being paused before the end of the screen)
		if scr.crit.lastY < len(scr.crit.reflection[0]) {
			for x := scr.crit.lastX; x < len(scr.crit.reflection); x++ {
				scr.crit.reflection[x][scr.crit.lastY] = reflection.ReflectedVideoStep{}
			}
			for x := 0; x < len(scr.crit.reflection); x++ {
				for y := scr.crit.lastY + 1; y < len(scr.crit.reflection[x]); y++ {
					scr.crit.reflection[x][y] = reflection.ReflectedVideoStep{}
				}
			}
		}
	}

	scr.crit.section.Unlock()

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (scr *screen) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (scr *screen) SetPixels(sig []signal.SignalAttributes, current bool) error {
	if len(sig) == 0 {
		return nil
	}

	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	cl := int((sig[0] & signal.Clock) >> signal.ClockShift)
	sl := int((sig[0] & signal.Scanline) >> signal.ScanlineShift)

	adjustedScanline := (sl + scr.crit.screenrollScanline)
	if adjustedScanline >= scr.crit.bufferHeight {
		adjustedScanline -= scr.crit.bufferHeight
	}

	offset := cl * 4
	offset += adjustedScanline * scr.crit.bufferPixels[scr.crit.plotIdx].Rect.Size().X * 4

	var col color.RGBA

	for i := range sig {
		// we accept a sig of signal.NoSignal and use it to black out unused
		// pixels.
		//
		// this is only really needed in very rare situations (such as emerging
		// from a rewind state when the screen is "smaller" then the previous
		// one drawn). but the performance hit is negligable so it's not worth
		// bothering with an additional condition.

		// check that we're not going to encounter an index-out-of-range error
		if offset >= len(scr.crit.bufferPixels[scr.crit.plotIdx].Pix)-4 {

			// if we're not currently in a screen roll then just stop drawing pixels
			if scr.crit.screenrollScanline == 0 {
				break
			}

			// otherwise continue drawing or we would be left with a lot of
			// undrawn pixels and a poor screen roll effect
			cl := int((sig[i] & signal.Clock) >> signal.ClockShift)
			offset = cl * 4
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

	if current {
		cl := int((sig[len(sig)-1] & signal.Clock) >> signal.ClockShift)
		sl := int((sig[len(sig)-1] & signal.Scanline) >> signal.ScanlineShift)
		scr.crit.lastX = cl
		scr.crit.lastY = sl
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

	for _, v := range ref {
		// array indexes into the reflection 2d array are taken from the reflected
		// TV signal.
		cl := int((v.Signal & signal.Clock) >> signal.ClockShift)
		sl := int((v.Signal & signal.Scanline) >> signal.ScanlineShift)

		// store Reflection instance
		if cl < len(scr.crit.reflection) && sl < len(scr.crit.reflection[cl]) {
			scr.crit.reflection[cl][sl] = v

			// set element pixel according to the video element that ulimately informed
			// the color of the pixel generated by the videocycle
			rgb := altColors[v.VideoElement]
			scr.crit.elementPixels.SetRGBA(cl, sl, rgb)

			// write to overlay. this uses the current overlay settings but it can be
			// changed later with a call to replotOverlay()
			scr.plotOverlay(cl, sl, v)
		}
	}

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
func (scr *screen) plotOverlay(x, y int, ref reflection.ReflectedVideoStep) {
	scr.crit.overlayPixels.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
	switch scr.crit.overlay {
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		if ref.WSYNC {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.WSYNC])
		}
	case reflection.OverlayLabels[reflection.OverlayCollision]:
		if ref.Collision.LastVideoCycle.IsCXCLR() {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.CXCLR])
		} else if !ref.Collision.LastVideoCycle.IsNothing() {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.Collision])
		}
	case reflection.OverlayLabels[reflection.OverlayHMOVE]:
		// HmoveCt counts to -1 (or 255 for a uint8)
		if ref.Hmove.Delay {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.HMOVEdelay])
		} else if ref.Hmove.Latch {
			if ref.Hmove.RippleCt != 255 {
				scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.HMOVEripple])
			} else {
				scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.HMOVElatched])
			}
		}
	case reflection.OverlayLabels[reflection.OverlayRSYNC]:
		if ref.RSYNCalign {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.RSYNCalign])
		} else if ref.RSYNCreset {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.RSYNCreset])
		}
	case reflection.OverlayLabels[reflection.OverlayCoproc]:
		if ref.CoprocessorActive {
			scr.crit.overlayPixels.SetRGBA(x, y, reflectionColors[reflection.CoprocessorActive])
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

// copy pixels from buffer to the pixels.
func (scr *screen) copyPixelsDebugmode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	// copy pixels from render buffer to the live copy.
	copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
}

// copy pixels from buffer to the pixels.
func (scr *screen) copyPixelsPlaymode() {
	scr.crit.section.Lock()
	defer scr.crit.section.Unlock()

	if scr.crit.frameInfo.Stable {
		if scr.img.emulation.State() == emulation.Paused {
			// when emulation is paused we alternate which frame to show. this
			// simple technique means that two-frame flicker kernels will show a
			// still image that looks natural.
			//
			// it does mean that for three-frame flicker kernels and single frame
			// kernels are sub-optimal but I think this is better than allowing a
			// flicker kernel to show only half the pixels necessary to show a
			// natural image.
			if scr.crit.pauseFrame {
				copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
			} else {
				copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.prevRenderIdx].Pix)
			}
			scr.crit.pauseFrame = !scr.crit.pauseFrame
			return
		} else {
			// attempt to sync frame generation with monitor refresh rate
			if scr.crit.monitorSync && scr.img.emulation.State() == emulation.Running {
				// advance render index
				scr.crit.prevRenderIdx = scr.crit.renderIdx
				scr.crit.renderIdx++
				if scr.crit.renderIdx >= len(scr.crit.bufferPixels) {
					scr.crit.renderIdx = 0
				}

				// render index has bumped into the plotting index. revert render index
				if scr.crit.renderIdx == scr.crit.plotIdx {
					// ** emulation not keeping up with screen update **

					// undo frame advancement. in earlier versions of the code we
					// reduced the renderIdx by two, having the effect of reusing not
					// the previous frame, but the frame before that.
					//
					// it was thought that this would help out the displaying of
					// two-frame flicker kernels. which it did, but in some cases that
					// could result in stuttering of moving sprites. it was
					// particularly bad if the FPS of the ROM was below the refresh
					// rate of the monitor.
					//
					// a good example of this is the introductory scroller in the demo
					// Ataventure (by KK of DMA). a scroller that updates every frame
					// at 50fps and causes very noticeable side-effects on a 60Hz
					// monitor.

					// by undoing the frame advancement however, we will cause the
					// prevRenderIdx to be the same as the renderIdx, which will cause
					// an ineffective pause screen for flicker kernels. remedy this by
					// simply swapping the current and previous index values
					t := scr.crit.prevRenderIdx
					scr.crit.prevRenderIdx = scr.crit.renderIdx
					scr.crit.renderIdx = t
				}
			}

			// copy pixels from render buffer to the live copy.
			copy(scr.crit.pixels.Pix, scr.crit.bufferPixels[scr.crit.renderIdx].Pix)
		}
	}

	// let the emulator thread know it's okay to continue as soon as possible
	//
	// this is only ever the case if monitorSync is true but there's no
	// performance harm in allowing the select block to run in all instances
	select {
	case <-scr.emuWait:
		scr.emuWaitAck <- true
	default:
	}
}
