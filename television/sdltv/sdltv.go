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

	// whether the emulation is currently paused - affects how we render the
	// screen
	paused bool

	// last mouse selection
	mouseX int // expressed as horizontal position
	mouseY int // expressed as scanlines

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
	tv.scr, err = newScreen(&tv.HeadlessTV)

	// set window size and scaling
	err = tv.scr.setScaling(scale)
	if err != nil {
		return nil, err
	}

	// register callbacks from HeadlessTV to SDLTV
	tv.SignalNewFrameHook = tv.newFrame

	// update tv (with a black image)
	err = tv.update()
	if err != nil {
		return nil, err
	}

	// begin gui loop
	go tv.guiLoop()

	// note that we've elected not to show the window on startup

	return tv, nil
}

// Signal is the principle method of communication between the VCS and
// televsion. note that most of the work is done in the embedded HeadlessTV
// instance
func (tv *SDLTV) Signal(attr television.SignalAttributes) {
	tv.HeadlessTV.Signal(attr)

	tv.guiLoopLock.Lock()
	// decode color
	r, g, b := byte(0), byte(0), byte(0)
	if attr.Pixel <= 256 {
		col := tv.Spec.Colors[attr.Pixel]
		r, g, b = byte((col&0xff0000)>>16), byte((col&0xff00)>>8), byte(col&0xff)
	}

	x := int32(tv.HorizPos.Value().(int)) + int32(tv.Spec.ClocksPerHblank)
	y := int32(tv.Scanline.Value().(int))

	tv.scr.setPixel(x, y, r, g, b)
	tv.guiLoopLock.Unlock()
}

func (tv *SDLTV) newFrame() error {
	defer tv.scr.swapPixels()
	return tv.update()
}

// update the gui so that it reflects changes to buffered data in the tv struct
func (tv *SDLTV) update() error {
	tv.guiLoopLock.Lock()
	defer tv.guiLoopLock.Unlock()

	// abbrogate mot of the updating to the screem
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
