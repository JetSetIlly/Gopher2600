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

package sdlplay

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/performance/limiter"
	"gopher2600/television"
	"io"

	"github.com/veandco/go-sdl2/sdl"
)

const pixelDepth = 4
const pixelWidth = 2.0

// SdlPlay is a simple SDL implementation of the television.Renderer interface
type SdlPlay struct {
	television.Television

	// functions that need to be performed in the main thread should be queued
	// for service.
	service     chan func()
	serviceDone chan error

	// limit number of frames per second
	lmtr *limiter.FpsLimiter

	// connects SDL guiLoop with the parent process
	eventChannel chan gui.Event

	// all audio is handled by the sound type
	snd *sound

	// sdl stuff
	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	pixels   []byte

	// current values for *playable* area of the screen. horizontal size never
	// changes
	//
	// these values are not the same as the window size. window size is scaled
	// appropriately
	scanlines   int32
	topScanline int

	// by how much each pixel should be scaled. note that this value needs to
	// be factored by both pixelWidth and GetSpec().AspectBias when applied to
	// the X axis
	pixelScale float32

	// window opening is delayed until television frame is stable
	showOnNextStable bool
}

// NewSdlPlay is the preferred method of initialisation for SdlPlay.
//
// MUST be called from the #mainthread
func NewSdlPlay(tv television.Television, scale float32) (*SdlPlay, error) {
	scr := &SdlPlay{
		Television:  tv,
		service:     make(chan func(), 1),
		serviceDone: make(chan error, 1),
	}

	var err error

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// SDL window - window size is set in Resize() function
	scr.window, err = sdl.CreateWindow("Gopher2600",
		int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED),
		0, 0,
		uint32(sdl.WINDOW_HIDDEN))
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// sdl renderer. we set the scaling amount in the setWindow function later
	// once we know what the tv specification is
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// initialise the sound system
	scr.snd, err = newSound(scr)
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// register ourselves as a television.Renderer
	scr.AddPixelRenderer(scr)

	// register ourselves as a television.AudioMixer
	scr.AddAudioMixer(scr)

	// resize window
	err = scr.resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesVisible)
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// set window scaling to default value
	err = scr.setWindow(scale)
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
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
func (scr *SdlPlay) Destroy(output io.Writer) {
	err := scr.texture.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	err = scr.renderer.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	err = scr.window.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}
}

// Reset implements television.Television interface
func (scr *SdlPlay) Reset() error {
	err := scr.Television.Reset()
	if err != nil {
		return err
	}
	return nil
}

// IsVisible implements gui.GUI interface
func (scr SdlPlay) IsVisible() bool {
	flgs := scr.window.GetFlags()
	return flgs&sdl.WINDOW_SHOWN == sdl.WINDOW_SHOWN
}

// show or hide window
//
// HELPER function can be called from #mainthread or not
func (scr SdlPlay) showWindow(show bool) {
	scr.service <- func() {
		if show {
			scr.window.Show()
		} else {
			scr.window.Hide()
		}
		scr.serviceDone <- nil
	}
}

// use scale of -1 to reapply existing scale value
//
// MUST only be called from the #mainthread
// use setWindowThread() is not called from render thread
func (scr *SdlPlay) setWindow(scale float32) error {
	if scale >= 0 {
		scr.pixelScale = scale
	}

	w := int32(float32(television.HorizClksVisible) * scr.pixelScale * pixelWidth * scr.GetSpec().AspectBias)
	h := int32(float32(scr.scanlines) * scr.pixelScale)
	scr.window.SetSize(w, h)

	// make sure everything drawn through the renderer is correctly scaled
	err := scr.renderer.SetScale(float32(w/television.HorizClksVisible), float32(h/scr.scanlines))
	if err != nil {
		return err
	}

	return nil
}

// wrap call to setWindow() in service call
//
// SHOULD not be called from the #mainthread
func (scr *SdlPlay) setWindowThread(scale float32) error {
	scr.service <- func() {
		scr.serviceDone <- scr.setWindow(scale)
	}
	return <-scr.serviceDone
}

// resize is the non-service-wrapped resize function. if you require a wrapped
// call to resize use Resize()
//
// MUST NOT be called from #mainthread
func (scr *SdlPlay) resize(topScanline, numScanlines int) error {
	// new screen limits
	scr.topScanline = topScanline
	scr.scanlines = int32(numScanlines)

	var err error

	// ----
	// pixels arrays and textures are always the maximum size allowed by the
	// specification. we need to remake them here because the specification may
	// have changed as part of the resize() event

	scr.pixels = make([]byte, television.HorizClksVisible*scr.scanlines*pixelDepth)

	// preset alpha channel - we never change the value of this channel
	for i := pixelDepth - 1; i < len(scr.pixels); i += pixelDepth {
		scr.pixels[i] = 255
	}

	scr.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888),
		int(sdl.TEXTUREACCESS_STREAMING),
		television.HorizClksVisible, scr.scanlines)
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
// SHOULD NOT be called from #mainthread
func (scr *SdlPlay) Resize(topScanline, numScanlines int) error {
	scr.service <- func() {
		scr.serviceDone <- scr.resize(topScanline, numScanlines)
	}
	return <-scr.serviceDone
}

// NewFrame implements television.PixelRenderer interface
//
// SHOULD NOT be called from #mainthread
func (scr *SdlPlay) NewFrame(frameNum int) error {
	scr.service <- func() {
		if scr.showOnNextStable && scr.IsStable() {
			scr.showWindow(true)
			scr.showOnNextStable = false
		}

		err := scr.texture.Update(nil, scr.pixels, int(television.HorizClksVisible*pixelDepth))
		if err != nil {
			scr.serviceDone <- err
			return
		}

		err = scr.renderer.Copy(scr.texture, nil, nil)
		if err != nil {
			scr.serviceDone <- err
			return
		}

		scr.renderer.Present()
		scr.serviceDone <- nil
	}

	// we don't clear pixels from one frame to the next

	return <-scr.serviceDone
}

// SetPixel implements television.PixelRenderer interface
//
// MUST not be called from #mainthread
func (scr *SdlPlay) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	if vblank {
		// we could return immediately but if vblank is on inside the visible
		// area we need to the set pixel to black, in case the vblank was off
		// in the previous frame (for efficiency, we're not clearing the pixel
		// array at the end of the frame)
		red = 0
		green = 0
		blue = 0
	}

	// adjust pixels so we're only dealing with the visible range
	x -= television.HorizClksHBlank
	y -= scr.topScanline

	if x < 0 || y < 0 {
		return nil
	}

	i := (y*int(television.HorizClksVisible) + x) * pixelDepth
	if i <= len(scr.pixels)-pixelDepth {
		scr.pixels[i] = red
		scr.pixels[i+1] = green
		scr.pixels[i+2] = blue

		// alpha value remains unchanged
	}

	return nil
}

// NewScanline implements television.PixelRenderer interface
//
// UNUSED
func (scr *SdlPlay) NewScanline(scanline int) error {
	return nil
}

// SetAltPixel implements television.PixelRenderer interface
//
// UNUSED
func (scr *SdlPlay) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return nil
}

// EndRendering implements television.PixelRenderer interface
//
// UNUSED
func (scr *SdlPlay) EndRendering() error {
	return nil
}

// SetMetaPixel implements gui.MetPixelRenderer interface
//
// UNUSED
func (scr *SdlPlay) SetMetaPixel(sig gui.MetaPixel) error {
	return nil
}
