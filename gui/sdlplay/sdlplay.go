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
	"io"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

const pixelDepth = 4
const pixelWidth = 2.0

// SdlPlay is a simple SDL implementation of the television.PixelRenderer interface
type SdlPlay struct {
	television.Television

	// functions that need to be performed in the main thread should be queued
	// for service.
	service    chan func()
	serviceErr chan error

	// SetFeature() hands off requests to the featureReq channel for servicing
	featureReq chan featureRequest
	featureErr chan error

	// connects SDL guiLoop with the parent process
	events chan gui.Event

	// all audio is handled by the sound type
	aud *sdlaudio.Audio

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

	// mouse coords at last frame
	mx, my int32

	// whether mouse is captured
	isCaptured bool
}

const windowTitle = "Gopher2600"
const windowTitleCaptured = "Gopher2600 [captured]"

// NewSdlPlay is the preferred method of initialisation for SdlPlay.
func NewSdlPlay(tv television.Television, scale float32) (*SdlPlay, error) {
	scr := &SdlPlay{
		Television: tv,
		service:    make(chan func(), 1),
		serviceErr: make(chan error, 1),
		featureReq: make(chan featureRequest, 1),
		featureErr: make(chan error, 1),
	}

	var err error

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	setupService()

	// SDL window - window size is set in Resize() function
	scr.window, err = sdl.CreateWindow(windowTitle,
		int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED),
		0, 0,
		uint32(sdl.WINDOW_HIDDEN))
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// sdl renderer. we set the scaling amount in the setWindow function late
	// once we know what the tv specification is
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// initialise the sound system
	scr.aud, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, errors.New(errors.SDLPlay, err)
	}

	// register ourselves as a television.Renderer
	scr.AddPixelRenderer(scr)

	// register ourselves as a television.AudioMixer
	scr.AddAudioMixer(scr.aud)

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

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	scr.renderer.Clear()
	scr.renderer.Present()

	return scr, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread
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

// show or hide window
func (scr SdlPlay) showWindow(show bool) {
	if show {
		scr.window.Show()
	} else {
		scr.window.Hide()
	}
}

// use scale of -1 to reapply existing scale value
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

// resize is the non-service-wrapped resize function
func (scr *SdlPlay) resize(topScanline, numScanlines int) error {
	// new screen limits
	scr.topScanline = topScanline
	scr.scanlines = int32(numScanlines)

	// pixels arrays and textures are always the maximum size allowed by the
	// specification. we need to remake them here because the specification may
	// have changed as part of the resize() event

	scr.pixels = make([]byte, television.HorizClksVisible*scr.scanlines*pixelDepth)

	// preset alpha channel - we never change the value of this channel
	for i := pixelDepth - 1; i < len(scr.pixels); i += pixelDepth {
		scr.pixels[i] = 255
	}

	var err error
	scr.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888),
		int(sdl.TEXTUREACCESS_STREAMING),
		television.HorizClksVisible, scr.scanlines)
	if err != nil {
		return errors.New(errors.SDLDebug, err)
	}

	// setWindow dimensions. see commentary for Resize() function in
	// PixelRenderer interface definition
	if !scr.IsStable() {
		scr.setWindow(-1)
	}

	return nil
}

// Resize implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlPlay) Resize(topScanline, numScanlines int) error {
	scr.service <- func() {
		scr.serviceErr <- scr.resize(topScanline, numScanlines)
	}
	return <-scr.serviceErr
}

// NewFrame implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
func (scr *SdlPlay) NewFrame(frameNum int) error {
	if scr.showOnNextStable && scr.IsStable() {
		scr.window.Show()
		scr.showOnNextStable = false
	}

	scr.service <- func() {
		err := scr.texture.Update(nil, scr.pixels, int(television.HorizClksVisible*pixelDepth))
		if err != nil {
			return
		}

		err = scr.renderer.Copy(scr.texture, nil, nil)
		if err != nil {
			return
		}

		scr.renderer.Present()
	}

	// unlike sdldebug we don't clear pixels on NewFrame in sdlplay

	// note that we're not returning errors from the service function nor are
	// we waiting for anying signal before continuing. it is too much of a
	// performance hit to stall every frame.
	//
	// of course, it means errors get lost and we might continue in an unsafe
	// state but I don't think it's too important.

	return nil
}

// SetPixel implements television.PixelRenderer interface
//
// MUST NOT be called from #mainthread
//
// interesting that writing to pixel array does not trigger a race condition
// even though we read pixels, when updating texture, in a different thread.
//
// !!TODO: race condition in SetPixel() in sdlplay?
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
