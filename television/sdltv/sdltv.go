package sdltv

import (
	"gopher2600/television"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	television.HeadlessTV

	// much of the sdl magic happens in the screen object
	scr *screen

	// the time the last frame was rendered - used to limit frame rate
	lastFrameRender time.Time

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

	// critical section protection
	guiLoopLock sync.Mutex
}

// NewSDLTV initiliases a new instance of an SDL based display for the VCS
func NewSDLTV(tvType string, scale float32) (*SDLTV, error) {
	var err error

	tv := new(SDLTV)

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

	// register new frame callback from HeadlessTV to SDLTV
	// leaving SignalNewScanline() hook at its default
	tv.HookNewFrame = tv.newFrame
	tv.HookSetPixel = tv.setPixel

	// update tv (with a black image)
	err = tv.update()
	if err != nil {
		return nil, err
	}

	// begin gui loop
	go tv.guiLoop()

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	return tv, nil
}

// Pixel puts the pixel on the tv
func (tv *SDLTV) setPixel(x, y int32, red, green, blue byte) error {
	tv.guiLoopLock.Lock()
	defer tv.guiLoopLock.Unlock()

	return tv.scr.setPixel(x, y, red, green, blue)
}

func (tv *SDLTV) newFrame() error {
	defer tv.scr.swapPixels()
	return tv.update()
}

// update the gui so that it reflects changes to buffered data in the tv struct
func (tv *SDLTV) update() error {
	tv.guiLoopLock.Lock()
	defer tv.guiLoopLock.Unlock()

	// abbrogate most of the updating to the screen instance
	err := tv.scr.update(tv.paused)
	if err != nil {
		return err
	}

	// FPS limiting - for windowed SDL, attempt to synchronise to 60fps (VSYNC
	// hint only seems to work if window is in full screen mode)
	time.Sleep(16666*time.Microsecond - time.Since(tv.lastFrameRender))
	tv.scr.renderer.Present()
	tv.lastFrameRender = time.Now()

	return nil
}
