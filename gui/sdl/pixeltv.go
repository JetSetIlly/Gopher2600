package sdl

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/television"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

// PixelTV is a simple SDL implementation of the television.Renderer interface
// with an embedded television for convenience. It treats every SetPixel() call
// as gospel - no refraction or blurring of adjacent pixels. It is imagined
// that other SDL implementations will be more imaginitive with SetPixel() and
// produce a more convincing image.
type PixelTV struct {
	television.Television

	// much of the sdl magic happens in the screen object
	scr *screen

	// audio
	snd *sound

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

// NewPixelTV creates a new instance of PixelTV. For convenience, the
// television argument can be nil, in which case an instance of
// StellaTelevision will be created.
func NewPixelTV(tvType string, scale float32, tv television.Television) (gui.GUI, error) {
	var err error

	// set up gui
	pxtv := new(PixelTV)

	// create or attach television implementation
	if tv == nil {
		pxtv.Television, err = television.NewStellaTelevision(tvType)
		if err != nil {
			return nil, errors.New(errors.SDL, err)
		}
	} else {
		// check that the quoted tvType matches the specification of the
		// supplied BasicTelevision instance. we don't really need this but
		// becuase we're implying that tvType is required, even when an
		// instance of BasicTelevision has been supplied, the caller may be
		// expecting an error
		tvType = strings.ToUpper(tvType)
		if tvType != "AUTO" && tvType != tv.GetSpec().ID {
			return nil, errors.New(errors.SDL, "trying to piggyback a tv of a different spec")
		}
		pxtv.Television = tv
	}

	// set up sdl
	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// initialise the screens we'll be using
	pxtv.scr, err = newScreen(pxtv)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// initialise the sound system
	pxtv.snd, err = newSound(pxtv)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// set window size and scaling
	err = pxtv.scr.setScaling(scale)
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// register ourselves as a television.Renderer
	pxtv.AddPixelRenderer(pxtv)

	// register ourselves as a television.AudioMixer
	pxtv.AddAudioMixer(pxtv)

	// update tv (with a black image)
	err = pxtv.scr.update()
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	// gui events are serviced by a separate go rountine
	go pxtv.guiLoop()

	// note that we've elected not to show the window on startup
	// window is instead opened on a ReqSetVisibility request

	return pxtv, nil
}

// ChangeTVSpec implements television.Television interface
func (pxtv *PixelTV) ChangeTVSpec() error {
	pxtv.scr.stb.restart()
	return pxtv.scr.initialiseScreen()
}

// NewFrame implements television.Renderer interface
func (pxtv *PixelTV) NewFrame(frameNum int) error {
	err := pxtv.scr.stb.stabiliseFrame()
	if err != nil {
		return err
	}

	err = pxtv.scr.update()
	if err != nil {
		return err
	}

	pxtv.scr.newFrame()

	return nil
}

// NewScanline implements television.Renderer interface
func (pxtv *PixelTV) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.Renderer interface
func (pxtv *PixelTV) SetPixel(x, y int, red, green, blue byte, vblank bool) error {
	return pxtv.scr.setRegPixel(int32(x), int32(y), red, green, blue, vblank)
}

// SetAltPixel implements television.Renderer interface
func (pxtv *PixelTV) SetAltPixel(x, y int, red, green, blue byte, vblank bool) error {
	if !pxtv.allowDebugging {
		return nil
	}
	return pxtv.scr.setAltPixel(int32(x), int32(y), red, green, blue, vblank)
}

// Reset implements television.Renderer interface
func (pxtv *PixelTV) Reset() error {
	err := pxtv.Television.Reset()
	if err != nil {
		return err
	}
	pxtv.scr.newFrame()
	pxtv.scr.lastX = 0
	pxtv.scr.lastY = 0
	return nil
}

// IsVisible implements gui.GUI interface
func (pxtv PixelTV) IsVisible() bool {
	flgs := pxtv.scr.window.GetFlags()
	return flgs&sdl.WINDOW_SHOWN == sdl.WINDOW_SHOWN
}
