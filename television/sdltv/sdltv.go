package sdltv

import (
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	television.HeadlessTV

	// much of the sdl magic happens in the screen object
	scr *screen

	fpsLimiter *fpsLimiter

	// callback functions
	onWindowClose      callback
	onMouseButtonLeft  callback
	onMouseButtonRight callback

	// whether the emulation is currently paused. if paused is true then
	// as much of the current frame is displayed as possible; the previous
	// frame will take up the remainder of the screen.
	paused bool

	// last mouse selection
	lastMouseHorizPos int
	lastMouseScanline int
}

// NewSDLTV initiliases a new instance of an SDL based display for the VCS
func NewSDLTV(tvType string, scale float32) (*SDLTV, error) {
	var err error

	tv := new(SDLTV)

	tv.fpsLimiter, err = newFPSLimiter(50)
	if err != nil {
		return nil, err
	}

	err = television.InitHeadlessTV(&tv.HeadlessTV, tvType)
	if err != nil {
		return nil, err
	}

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, err
	}

	// initialise the screens we'll be using
	tv.scr, err = newScreen(tv)

	// set window size and scaling
	err = tv.scr.setScaling(scale)
	if err != nil {
		return nil, err
	}

	// register headlesstv callbacks
	// leave SignalNewScanline() hook at its default
	tv.HookNewFrame = tv.update
	tv.HookSetPixel = tv.scr.setPixel

	// update tv (with a black image)
	err = tv.update()
	if err != nil {
		return nil, err
	}

	// gui events are serviced by a separate loop
	go tv.guiLoop()

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	return tv, nil
}

// update the gui so that it reflects changes to buffered data in the tv struct
func (tv *SDLTV) update() error {
	tv.fpsLimiter.wait()

	// abbrogate most of the updating to the screen instance
	err := tv.scr.update(tv.paused)
	if err != nil {
		return err
	}

	tv.scr.renderer.Present()
	tv.scr.swapPixels()

	return nil
}
