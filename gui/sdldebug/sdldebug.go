package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// SdlDebug is a simple SDL implementation of the television.Renderer interface
type SdlDebug struct {
	television.Television

	window *sdl.Window

	// much of the sdl magic happens in the screen object
	pxl *pixels

	// connects SDL guiLoop with the parent process
	eventChannel chan gui.Event

	// whether the emulation is currently paused. if paused is true then
	// as much of the current frame is displayed as possible; the previous
	// frame will take up the remainder of the screen.
	paused bool
}

// NewSdlDebug is the preferred method for creating a new instance of SdlDebug
func NewSdlDebug(tv television.Television, scale float32) (gui.GUI, error) {
	var err error

	// set up gui
	scr := &SdlDebug{Television: tv}

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// SDL window - the correct size for the window will be determined below
	scr.window, err = sdl.CreateWindow("Gopher2600", int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED), 0, 0, uint32(sdl.WINDOW_HIDDEN))
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// initialise the screens we'll be using
	scr.pxl, err = newScreen(scr)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// set attributes that depend on the television specification
	err = scr.Resize(scr.GetSpec().ScanlineTop, scr.GetSpec().ScanlinesVisible)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// set window size and scaling
	err = scr.pxl.setScaling(scale)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// register ourselves as a television.Renderer
	scr.AddPixelRenderer(scr)

	// update tv (with a black image)
	err = scr.pxl.update()
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
func (scr *SdlDebug) Resize(topScanline, numScanlines int) error {
	return scr.pxl.resize(topScanline, numScanlines)
}

// NewFrame implements television.Renderer interface
func (scr *SdlDebug) NewFrame(frameNum int) error {
	err := scr.pxl.update()
	if err != nil {
		return err
	}

	scr.pxl.newFrame()

	return nil
}

// NewScanline implements television.Renderer interface
func (scr *SdlDebug) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.Renderer interface
func (scr *SdlDebug) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	return scr.pxl.setRegPixel(int32(x), int32(y), red, green, blue, vblank)
}

// SetAltPixel implements television.Renderer interface
func (scr *SdlDebug) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	return scr.pxl.setAltPixel(int32(x), int32(y), red, green, blue, vblank)
}

// Reset implements television.Renderer interface
func (scr *SdlDebug) Reset() error {
	err := scr.Television.Reset()
	if err != nil {
		return err
	}
	return scr.pxl.reset()
}

// IsVisible implements gui.GUI interface
func (scr SdlDebug) IsVisible() bool {
	flgs := scr.window.GetFlags()
	return flgs&sdl.WINDOW_SHOWN == sdl.WINDOW_SHOWN
}
