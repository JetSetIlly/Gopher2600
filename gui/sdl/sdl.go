package sdl

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// GUI is the SDL implementation of a the gui/television
type GUI struct {
	television.Television

	// much of the sdl magic happens in the screen object
	scr *screen

	// regulates how often the screen is updated
	fpsLimiter *fpsLimiter

	// connects SDL guiLoop with the parent process
	eventChannel chan gui.Event

	// whether the emulation is currently paused. if paused is true then
	// as much of the current frame is displayed as possible; the previous
	// frame will take up the remainder of the screen.
	paused bool

	// ther's a small bug significant performance boost if we disable certain
	// code paths with this allowDebugging flag
	allowDebugging bool
}

// NewGUI initiliases a new instance of an SDL based display for the VCS
func NewGUI(tvType string, scale float32, tv television.Television) (gui.GUI, error) {
	var err error

	// set up gui
	gtv := new(GUI)

	// create or attach television implementation
	if tv == nil {
		gtv.Television, err = television.NewBasicTelevision(tvType)
		if err != nil {
			return nil, errors.NewFormattedError(errors.SDL, err)
		}
	} else {
		// check that the quoted tvType matches the specification of the
		// supplied BasicTelevision instance. we don't really need this but
		// becuase we're implying that tvType is required, even when an
		// instance of BasicTelevision has been supplied, the caller may be
		// expecting an error
		if tvType != tv.GetSpec().ID {
			return nil, errors.NewFormattedError(errors.SDL, "trying to piggyback a tv of a different spec")
		}
		gtv.Television = tv
	}

	gtv.fpsLimiter, err = newFPSLimiter(50)
	if err != nil {
		return nil, errors.NewFormattedError(errors.SDL, err)
	}

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.NewFormattedError(errors.SDL, err)
	}

	// initialise the screens we'll be using
	gtv.scr, err = newScreen(gtv)

	// set window size and scaling
	err = gtv.scr.setScaling(scale)
	if err != nil {
		return nil, errors.NewFormattedError(errors.SDL, err)
	}

	// register ourselves as a television.Renderer
	gtv.AddRenderer(gtv)

	// update tv (with a black image)
	err = gtv.update()
	if err != nil {
		return nil, errors.NewFormattedError(errors.SDL, err)
	}

	// gui events are serviced by a separate loop
	go gtv.guiLoop()

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	return gtv, nil
}

// update the gui so that it reflects changes to buffered data in the tv struct
func (gtv *GUI) update() error {
	gtv.fpsLimiter.wait()

	// abbrogate most of the updating to the screen instance
	err := gtv.scr.update(gtv.paused)
	if err != nil {
		return err
	}

	gtv.scr.renderer.Present()

	return nil
}

func (gtv *GUI) setDebugging(allow bool) {
	gtv.allowDebugging = allow
}

// NewFrame implements television.Renderer interface
func (gtv *GUI) NewFrame(frameNum int) error {
	defer gtv.scr.clearPixels()
	err := gtv.scr.stb.checkStableFrame()
	if err != nil {
		return err
	}
	return gtv.update()
}

// NewScanline implements television.Renderer interface
func (gtv *GUI) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.Renderer interface
func (gtv *GUI) SetPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return gtv.scr.setRegPixel(x, y, red, green, blue, vblank)
}

// SetAltPixel implements television.Renderer interface
func (gtv *GUI) SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	if !gtv.allowDebugging {
		return nil
	}
	return gtv.scr.setAltPixel(x, y, red, green, blue, vblank)
}
