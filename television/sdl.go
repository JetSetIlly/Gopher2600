package television

import (
	"fmt"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	HeadlessTV

	width      int32
	height     int32
	pixelDepth int32

	pixelWidth int

	window        *sdl.Window
	renderer      *sdl.Renderer
	screenTexture *sdl.Texture
	fadeTexture   *sdl.Texture

	pixelsScreen []byte // screen buffer
	pixelsFade   []byte
	pixels0      []byte
	pixels1      []byte

	paused bool

	lastFrameRender time.Time
}

// NewSDLTV is the preferred method for initalising an SDL TV
func NewSDLTV(tvType string, scale float32) (*SDLTV, error) {
	var err error

	tv := new(SDLTV)
	if tv == nil {
		return nil, fmt.Errorf("can't allocate memory for sdl tv")
	}

	err = InitHeadlessTV(&tv.HeadlessTV, tvType)
	if err != nil {
		return nil, err
	}

	// register callbacks from HeadlessTV to SDLTV
	tv.newFrame = func() error {
		err := tv.update()
		if err != nil {
			return err
		}

		// swap which pixel buffer we're using
		swp := tv.pixelsScreen
		tv.pixelsScreen = tv.pixelsFade
		tv.pixelsFade = swp

		tv.clearScreenBuffer()

		return nil
	}

	// image dimensions
	tv.width = int32(tv.HeadlessTV.spec.clocksPerScanline)
	tv.height = int32(tv.HeadlessTV.spec.scanlinesTotal)

	// set up sdl
	err = sdl.Init(uint32(0))
	if err != nil {
		return nil, err
	}

	// pixelWidth is the number of tv pixels per color clock. we don't need to
	// worry about this again once we've created the window and set the scaling
	// for the renderer
	tv.pixelWidth = 2

	// SDL initialisation

	// SDL window
	tv.window, err = sdl.CreateWindow("Gopher2600", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(float32(tv.width)*scale*float32(tv.pixelWidth)), int32(float32(tv.height)*scale), sdl.WINDOW_HIDDEN)
	if err != nil {
		return nil, err
	}

	// SDL renderer
	tv.renderer, err = sdl.CreateRenderer(tv.window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return nil, err
	}

	// everything applied to the renderer will be scaled
	err = tv.renderer.SetScale(float32(tv.pixelWidth)*scale, scale)
	if err != nil {
		return nil, err
	}

	// number of bytes per pixel (indicating PIXELFORMAT)
	tv.pixelDepth = 4

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, whether the tv is paused or unpaused
	tv.screenTexture, err = tv.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, tv.width, tv.height)
	if err != nil {
		return nil, err
	}
	tv.screenTexture.SetBlendMode(sdl.BLENDMODE_BLEND)

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	tv.fadeTexture, err = tv.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, tv.width, tv.height)
	if err != nil {
		return nil, err
	}
	tv.fadeTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	tv.fadeTexture.SetAlphaMod(50)

	// our acutal screen data
	tv.pixels0 = make([]byte, tv.width*tv.height*tv.pixelDepth)
	tv.pixels1 = make([]byte, tv.width*tv.height*tv.pixelDepth)
	tv.pixelsScreen = tv.pixels0
	tv.pixelsFade = tv.pixels1

	// finish up by updating the tv with a black image
	// -- note that we've elected not to show the window on startup
	tv.clearScreenBuffer()
	err = tv.update()
	if err != nil {
		return nil, err
	}

	return tv, nil
}

// Signal is principle method of communication between the VCS and televsion
// -- note that most of the work is done in the embedded HeadlessTV instance
func (tv *SDLTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, pixel PixelSignal) {
	tv.HeadlessTV.Signal(vsync, vblank, frontPorch, hsync, cburst, pixel)

	// decode color
	r, g, b := byte(0), byte(0), byte(0)
	col, present := tv.spec.colors[pixel]
	if present {
		r, g, b = byte((col&0xff0000)>>16), byte((col&0xff00)>>8), byte(col&0xff)
	}

	tv.setPixel(int32(tv.pixelX()), int32(tv.pixelY()), r, g, b, tv.pixelsScreen)
}

func (tv *SDLTV) clearScreenBuffer() {
	for y := int32(0); y < tv.height; y++ {
		for x := int32(0); x < tv.width; x++ {
			i := (y*tv.width + x) * tv.pixelDepth
			tv.pixelsScreen[i] = 0
			tv.pixelsScreen[i+1] = 0
			tv.pixelsScreen[i+2] = 0
			tv.pixelsScreen[i+3] = 0
		}
	}
}

func (tv *SDLTV) setPixel(x, y int32, red, green, blue byte, pixels []byte) {
	i := (y*tv.width + x) * tv.pixelDepth
	if i < int32(len(pixels))-tv.pixelDepth && i >= 0 {
		pixels[i] = red
		pixels[i+1] = green
		pixels[i+2] = blue
		pixels[i+3] = 255
	}
}

func (tv *SDLTV) update() error {
	var err error

	// clear image from rendered
	tv.renderer.SetDrawColor(5, 5, 5, 255)
	tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
	err = tv.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if tv.paused {
		err := tv.fadeTexture.Update(nil, tv.pixelsFade, int(tv.width*tv.pixelDepth))
		if err != nil {
			return err
		}
		err = tv.renderer.Copy(tv.fadeTexture, nil, nil)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	err = tv.screenTexture.Update(nil, tv.pixelsScreen, int(tv.width*tv.pixelDepth))
	if err != nil {
		return err
	}
	err = tv.renderer.Copy(tv.screenTexture, nil, nil)
	if err != nil {
		return err
	}

	// add screen boundary overlay
	tv.renderer.SetDrawColor(100, 100, 100, 25)
	tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	tv.renderer.FillRect(&sdl.Rect{0, 0, int32(tv.spec.clocksPerHblank), int32(tv.spec.scanlinesTotal)})
	tv.renderer.FillRect(&sdl.Rect{0, 0, int32(tv.spec.clocksPerScanline), int32(tv.spec.scanlinesPerVBlank)})
	tv.renderer.FillRect(&sdl.Rect{0, int32(tv.spec.scanlinesTotal - tv.spec.scanlinesPerOverscan), int32(tv.spec.clocksPerScanline), int32(tv.spec.scanlinesPerOverscan)})

	// add cursor overlay only if tv is paused
	if tv.paused {
		tv.renderer.SetDrawColor(255, 255, 255, 100)
		tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
		cursorX := tv.pixelX()
		cursorY := tv.pixelY()
		if cursorX >= tv.spec.clocksPerScanline+tv.spec.clocksPerHblank {
			cursorX = 0
			cursorY++
		}
		tv.renderer.DrawRect(&sdl.Rect{int32(cursorX), int32(cursorY), 2, 2})
	}

	// finalise updating of screen

	// for windowed SDL, attempt to synchronise to 60fps (VSYNC hint only seems
	// to work if window is in full screen mode)
	time.Sleep(16666*time.Microsecond - time.Since(tv.lastFrameRender))
	tv.renderer.Present()
	tv.lastFrameRender = time.Now()

	return nil
}

// SetVisibility toggles the visiblity of the SDLTV window
func (tv *SDLTV) SetVisibility(visible bool) error {
	if visible {
		tv.window.Show()
	} else {
		tv.window.Hide()
	}
	return nil
}

// SetPause toggles whether the tv is currently being updated. we can use
// this when we pause the emulation to make sure aren't left with a blank
// screen
func (tv *SDLTV) SetPause(pause bool) error {
	if pause {
		tv.paused = true
		tv.update()
	} else {
		tv.paused = false
	}
	return nil
}
