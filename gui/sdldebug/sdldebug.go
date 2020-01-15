// This file is part of Gopher2600
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

package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/performance/limiter"
	"gopher2600/television"
	"io"

	"github.com/veandco/go-sdl2/sdl"
)

// SdlDebug is a simple SDL implementation of the television.Renderer interfac
type SdlDebug struct {
	television.Television

	// functions that need to be performed in the main thread should be queued
	// for service
	service     chan func()
	serviceDone chan error

	// limit number of frames per second
	lmtr *limiter.FpsLimiter

	// connects SDL guiLoop with the parent process
	eventChannel chan gui.Event

	// sdl stuff
	window   *sdl.Window
	renderer *sdl.Renderer
	textures *textures
	pixels   *pixels
	overlay  *overlay

	// current values for *playable* area of the screen. horizontal size never
	// changes
	//
	// these values are not the same as the window size. window size is scaled
	// appropriately
	scanlines   int32
	topScanline int

	// the rectangle used to limit which pixels are copied from the pixels
	// array to the rendering texture
	cpyRect *sdl.Rect

	// the number of pixels in the various pixel arrays. this includes the
	// pixel array in the overlay type
	pitch int

	// by how much each pixel should be scaled. note that this value needs to
	// be factored by both pixelWidth and GetSpec().AspectBias when applied to
	// the X axis
	pixelScale float32

	// if the emulation is paused then the image is output slightly differently
	paused bool

	// use alternative color palette
	useAltColors bool

	// use metapixel overlay
	useOverlay bool

	// show the overscan/hblank areas
	masked bool

	// the position of the previous call to Pixel(). used for drawing cursor
	// and plotting meta-pixels
	lastX int
	lastY int
}

// NewSdlDebug is the preferred method of initialisation for SdlDebug.
//
// MUST ONLY be called from the #mainthread
func NewSdlDebug(tv television.Television, scale float32) (*SdlDebug, error) {
	scr := &SdlDebug{
		Television:  tv,
		service:     make(chan func(), 1),
		serviceDone: make(chan error, 1),
		pitch:       television.HorizClksScanline * pixelDepth,
		masked:      true,
		paused:      true,
	}

	var err error

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDLDebug, err)
	}

	// SDL window - window size is set in Resize() function
	scr.window, err = sdl.CreateWindow("Gopher2600",
		int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED),
		0, 0,
		uint32(sdl.WINDOW_HIDDEN))
	if err != nil {
		return nil, errors.New(errors.SDLDebug, err)
	}

	// sdl renderer. we set the scaling amount in the setWindow function later
	// once we know what the tv specification is
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		return nil, errors.New(errors.SDLDebug, err)
	}

	// register ourselves as a television.Renderer
	scr.AddPixelRenderer(scr)

	// resize window
	err = scr.resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesVisible)
	if err != nil {
		return nil, errors.New(errors.SDLDebug, err)
	}

	// set window scaling to default value
	err = scr.setWindow(scale)
	if err != nil {
		return nil, errors.New(errors.SDLDebug, err)
	}

	// start off with fps cap
	scr.lmtr = limiter.NewFPSLimiter(scr.GetSpec().FramesPerSecond)

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	scr.renderer.Clear()
	scr.renderer.Present()

	return scr, nil
}

// Destroy implements gui.GUI interface
//
// MUST ONLY be called from the #mainthread
func (scr *SdlDebug) Destroy(output io.Writer) {
	scr.overlay.destroy(output)
	scr.textures.destroy(output)

	err := scr.renderer.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	err = scr.window.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}
}

// Reset implements television.Television interface
func (scr *SdlDebug) Reset() error {
	err := scr.Television.Reset()
	if err != nil {
		return err
	}
	return nil
}

// IsVisible implements gui.GUI interface
func (scr SdlDebug) IsVisible() bool {
	flgs := scr.window.GetFlags()
	return flgs&sdl.WINDOW_SHOWN == sdl.WINDOW_SHOWN
}

// show or hide window
func (scr SdlDebug) showWindow(show bool) {
	scr.service <- func() {
		if show {
			scr.window.Show()
		} else {
			scr.window.Hide()
		}
		scr.serviceDone <- nil
	}
	<-scr.serviceDone
}

// the desired window width is different depending on whether the frame is
// masked or unmasked
func (scr SdlDebug) windowWidth() (int32, float32) {
	scale := scr.pixelScale * pixelWidth * scr.GetSpec().AspectBias

	if scr.masked {
		return int32(float32(television.HorizClksVisible) * scale), scale
	}

	return int32(float32(television.HorizClksScanline) * scale), scale
}

// the desired window height is different depending on whether the frame is
// masked or unmasked
func (scr SdlDebug) windowHeight() (int32, float32) {
	if scr.masked {
		return int32(float32(scr.scanlines) * scr.pixelScale), scr.pixelScale
	}

	return int32(float32(scr.GetSpec().ScanlinesTotal) * scr.pixelScale), scr.pixelScale
}

// use scale of -1 to reapply existing scale value
//
// MUST ONLY be called from the #mainthread
// use setWindowThread() is not called from render thread
func (scr *SdlDebug) setWindow(scale float32) error {
	if scale >= 0 {
		scr.pixelScale = scale
	}

	w, ws := scr.windowWidth()
	h, hs := scr.windowHeight()
	scr.window.SetSize(w, h)

	// make sure everything drawn through the renderer is correctly scaled
	err := scr.renderer.SetScale(ws, hs)
	if err != nil {
		return err
	}

	// make copy rectangled
	if scr.masked {
		scr.cpyRect = &sdl.Rect{
			television.HorizClksHBlank, int32(scr.topScanline),
			television.HorizClksVisible, scr.scanlines,
		}
	} else {
		scr.cpyRect = &sdl.Rect{
			0, 0,
			television.HorizClksScanline, int32(scr.GetSpec().ScanlinesTotal),
		}
	}

	return nil
}

// wrap call to setWindow() in service call
//
// MUST NOT be called from the #mainthread
func (scr *SdlDebug) setWindowThread(scale float32) error {
	scr.service <- func() {
		scr.serviceDone <- scr.setWindow(scale)
	}
	return <-scr.serviceDone
}

// resize is the non-service-wrapped resize function. if you require a wrapped
// call to resize use Resize()
//
// MUST ONLY be called from #mainthread
func (scr *SdlDebug) resize(topScanline, numScanlines int) error {
	// new screen limits
	scr.topScanline = topScanline
	scr.scanlines = int32(numScanlines)

	var err error

	// ----
	// pixels arrays (including the overlay) and textures are always the
	// maximum size allowed by the specification. we need to remake them here
	// because the specification may have changed as part of the resize() event

	scr.pixels = newPixels(television.HorizClksScanline, scr.GetSpec().ScanlinesTotal)

	scr.textures, err = newTextures(scr.renderer, television.HorizClksScanline, scr.GetSpec().ScanlinesTotal)
	if err != nil {
		return errors.New(errors.SDLDebug, err)
	}

	scr.overlay, err = newOverlay(scr.renderer, television.HorizClksScanline, scr.GetSpec().ScanlinesTotal)
	if err != nil {
		return errors.New(errors.SDLDebug, err)
	}
	// ----

	scr.setWindow(-1)
	scr.lmtr = limiter.NewFPSLimiter(scr.GetSpec().FramesPerSecond)

	return nil
}

// Resize implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) Resize(topScanline, numScanlines int) error {
	scr.service <- func() {
		scr.serviceDone <- scr.resize(topScanline, numScanlines)
	}
	return <-scr.serviceDone
}

// update is called automatically on every call to NewFrame() and whenever a
// state change in SetFeature() requires it.
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) update() error {
	scr.service <- func() {
		scr.renderer.SetDrawColor(0, 0, 0, 255)
		err := scr.renderer.Clear()
		if err != nil {
			scr.serviceDone <- err
			return
		}

		// decide whether to use regular or alt pixels
		pixels := scr.pixels.regular
		if scr.useAltColors {
			pixels = scr.pixels.alt
		}

		// render main textures
		err = scr.textures.render(scr.cpyRect, pixels, scr.pitch)
		if err != nil {
			scr.serviceDone <- err
			return
		}

		// render screen guides
		if !scr.masked {
			scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
			scr.renderer.SetDrawColor(100, 100, 100, 50)
			r := &sdl.Rect{0, 0,
				int32(television.HorizClksHBlank), int32(scr.GetSpec().ScanlinesTotal)}
			err = scr.renderer.FillRect(r)
			if err != nil {
				scr.serviceDone <- err
				return
			}

			r = &sdl.Rect{0, 0,
				int32(television.HorizClksScanline), int32(scr.GetSpec().ScanlineTop)}
			err = scr.renderer.FillRect(r)
			if err != nil {
				scr.serviceDone <- err
				return
			}

			r = &sdl.Rect{0, int32(scr.GetSpec().ScanlineBottom),
				int32(television.HorizClksScanline), int32(scr.GetSpec().ScanlinesOverscan)}
			err = scr.renderer.FillRect(r)
			if err != nil {
				scr.serviceDone <- err
				return
			}
		}

		// render overlay
		if scr.useOverlay {
			err = scr.overlay.render(scr.cpyRect, scr.pitch)
			if err != nil {
				scr.serviceDone <- err
				return
			}
		}

		if scr.paused {
			// adjust cursor coordinates
			x := scr.lastX - 1
			y := scr.lastY
			if scr.masked {
				y -= scr.topScanline
				x -= television.HorizClksHBlank - 1
			}

			scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
			scr.renderer.SetDrawColor(100, 100, 255, 255)

			if x < 0 {
				scr.renderer.SetDrawColor(255, 100, 100, 255)
				x = 0
			}

			if y < 0 {
				scr.renderer.SetDrawColor(255, 100, 100, 255)
				y = 0
			}

			// leave the current pixel visible at the top-left corner of the cursor
			scr.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y), W: 1, H: 1})
			scr.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y + 1), W: 1, H: 1})
			scr.renderer.DrawRect(&sdl.Rect{X: int32(x), Y: int32(y + 1), W: 1, H: 1})
		}

		scr.renderer.Present()
		scr.serviceDone <- nil
	}

	return <-scr.serviceDone
}

// NewFrame implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) NewFrame(frameNum int) error {
	err := scr.update()
	if err != nil {
		return err
	}
	scr.pixels.clear()
	scr.overlay.clear()
	scr.textures.flip()
	return nil
}

// SetPixel implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	if vblank {
		// we could return immediately but if vblank is on inside the visible
		// area we need to the set pixel to black, in case the vblank was off
		// in the previous frame (for efficiency, we're not clearing the pixel
		// array at the end of the frame)
		red = 0
		green = 0
		blue = 0
	}

	i := (y*int(television.HorizClksScanline) + x) * pixelDepth
	if i <= scr.pixels.length()-pixelDepth {
		scr.pixels.regular[i] = red
		scr.pixels.regular[i+1] = green
		scr.pixels.regular[i+2] = blue
		scr.pixels.regular[i+3] = 255
	}

	// update cursor position
	scr.lastX = x
	scr.lastY = y

	return nil
}

// SetAltPixel implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	i := (y*int(television.HorizClksScanline) + x) * pixelDepth
	if i <= scr.pixels.length()-pixelDepth {
		scr.pixels.alt[i] = red
		scr.pixels.alt[i+1] = green
		scr.pixels.alt[i+2] = blue
		scr.pixels.alt[i+3] = 255
	}

	return nil
}

// SetMetaPixel implements gui.MetPixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlDebug) SetMetaPixel(sig gui.MetaPixel) error {
	i := (scr.lastY*int(television.HorizClksScanline) + scr.lastX) * pixelDepth
	if i <= scr.overlay.length()-pixelDepth {
		scr.overlay.pixels[i] = sig.Red
		scr.overlay.pixels[i+1] = sig.Green
		scr.overlay.pixels[i+2] = sig.Blue
		scr.overlay.pixels[i+3] = sig.Alpha
	}

	return nil
}

// NewScanline implements television.PixelRenderer interface
//
// UNUSED
func (scr *SdlDebug) NewScanline(scanline int) error {
	return nil
}

// EndRendering implements television.PixelRenderer interface
//
// UNUSED
func (scr *SdlDebug) EndRendering() error {
	return nil
}
