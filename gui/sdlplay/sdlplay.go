package sdlplay

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/performance/limiter"
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

const pixelDepth = 4
const pixelWidth = 2.0

// SdlPlay is a simple SDL implementation of the television.Renderer interface
type SdlPlay struct {
	television.Television

	// connects SDL guiLoop with the parent process
	eventChannel chan gui.Event

	// limit screen updates to a fixed fps
	lmtr   *limiter.FpsLimiter
	fpsCap bool

	// all audio is handled by the sound type
	snd *sound

	// sdl stuff
	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture

	// horizPixels and scanlines represent the *actual* value for the current
	// ROM. many ROMs go beyond the spec and push the number of scanlines into
	// the overscan area. the horizPixels value never changes. it is included
	// for completeness and clarity
	//
	// these values are not the same as the window size. window size is scaled
	// appropriately
	horizPixels int32
	scanlines   int32
	topScanline int

	// pixels is the byte array that we copy to the texture before applying to
	// the renderer. it is equal to horizPixels * scanlines * pixelDepth.
	pixels []byte

	// the amount of scaling applied to each pixel. X is adjusted by an aspect
	// bias, defined in the television specs
	scaleX float32
	scaleY float32

	showOnNextStable bool
}

// NewSdlPlay is the preferred method of initialisation for SdlPlay
func NewSdlPlay(tv television.Television, scale float32) (gui.GUI, error) {
	// set up gui
	scr := &SdlPlay{Television: tv}

	var err error

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// SDL window - window size is set in Resize() function
	scr.window, err = sdl.CreateWindow("Gopher2600",
		int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED),
		0, 0,
		uint32(sdl.WINDOW_HIDDEN))
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// sdl renderer. we set the scaling amount in the setScaling function later
	// once we know what the tv specification is
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// initialise the sound system
	scr.snd, err = newSound(scr)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// register ourselves as a television.Renderer
	scr.AddPixelRenderer(scr)

	// register ourselves as a television.AudioMixer
	scr.AddAudioMixer(scr)

	// change tv spec after window creation (so we can set the window size)
	err = scr.Resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesVisible)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// set scaling to default value
	err = scr.setScaling(scale)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	scr.lmtr, err = limiter.NewFPSLimiter(scr.GetSpec().FramesPerSecond)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// gui events are serviced by a separate go rountine
	go scr.guiLoop()

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	return scr, nil
}

// Resize implements television.Television interface
func (scr *SdlPlay) Resize(topScanline, numScanlines int) error {
	var err error

	scr.horizPixels = television.HorizClksVisible
	scr.scanlines = int32(numScanlines)
	scr.topScanline = topScanline
	scr.pixels = make([]byte, scr.horizPixels*scr.scanlines*pixelDepth)

	// preset alpha channel - we never change the value of this channel
	for i := pixelDepth - 1; i < len(scr.pixels); i += pixelDepth {
		scr.pixels[i] = 255
	}

	// texture is applied to the renderer to show the image. we copy the pixels
	// to it every NewFrame()
	//
	// texture is the same size as the pixel arry. scaling will be applied to
	// in order to fit it in the window
	scr.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888),
		int(sdl.TEXTUREACCESS_STREAMING),
		scr.horizPixels,
		scr.scanlines)
	if err != nil {
		return nil
	}

	scr.setScaling(-1)

	return nil
}

// use scale of -1 to reapply existing scale value
func (scr *SdlPlay) setScaling(scale float32) error {
	if scale >= 0 {
		scr.scaleY = scale
		scr.scaleX = scale * scr.GetSpec().AspectBias
	}

	w := int32(float32(scr.horizPixels) * scr.scaleX * pixelWidth)
	h := int32(float32(scr.scanlines) * scr.scaleY)
	scr.window.SetSize(w, h)

	// make sure everything drawn through the renderer is correctly scaled
	err := scr.renderer.SetScale(float32(w/scr.horizPixels), float32(h/scr.scanlines))
	if err != nil {
		return err
	}

	return nil
}

// NewFrame implements television.Renderer interface
func (scr *SdlPlay) NewFrame(frameNum int) error {
	if scr.showOnNextStable {
		scr.showWindow(true)
		scr.showOnNextStable = false
	}

	if scr.fpsCap {
		scr.lmtr.Wait()
	}

	err := scr.texture.Update(nil, scr.pixels, int(scr.horizPixels*pixelDepth))
	if err != nil {
		return err
	}

	err = scr.renderer.Copy(scr.texture, nil, nil)
	if err != nil {
		return err
	}

	scr.renderer.Present()

	return nil
}

// NewScanline implements television.Renderer interface
func (scr *SdlPlay) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.Renderer interface
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

	i := (y*int(scr.horizPixels) + x) * pixelDepth
	if i <= len(scr.pixels)-pixelDepth {
		scr.pixels[i] = red
		scr.pixels[i+1] = green
		scr.pixels[i+2] = blue
	}

	return nil
}

// SetAltPixel implements television.Renderer interface
func (scr *SdlPlay) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return nil
}

// SetMetaPixel implements gui.MetPixelRenderer interface
func (scr *SdlPlay) SetMetaPixel(sig gui.MetaPixel) error {
	return nil
}

// Reset implements television.Renderer interface
func (scr *SdlPlay) Reset() error {
	err := scr.Television.Reset()
	if err != nil {
		return err
	}
	return nil
}

// EndRendering implements television.Renderer interface
func (scr *SdlPlay) EndRendering() error {
	return nil
}

// IsVisible implements gui.GUI interface
func (scr SdlPlay) IsVisible() bool {
	flgs := scr.window.GetFlags()
	return flgs&sdl.WINDOW_SHOWN == sdl.WINDOW_SHOWN
}

func (scr SdlPlay) showWindow(show bool) {
	if show {
		scr.window.Show()
	} else {
		scr.window.Hide()
	}
}
