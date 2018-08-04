package sdltv

import (
	"gopher2600/television"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// IdealScale is the suggested scaling for the screen
const IdealScale = 2.0

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	television.HeadlessTV

	window   *sdl.Window
	renderer *sdl.Renderer

	// we can flip between two screen types. a regular play screen, which is
	// masked as per a real television. and a debug screen, which has no
	// masking
	playScr *screen
	dbgScr  *screen
	// scr points to the screen currently in use
	scr *screen

	// the width of each VCS colour clock (in SDL pixels)
	pixelWidth int

	// by how much each pixel should be scaled
	pixelScale float32

	// the time the last frame was rendered - used to limit frame rate
	lastFrameRender time.Time

	// function to all when close button is pressed
	onWindowClose func()

	// whether the emulation is currently paused - affects how we render the
	// screen
	paused bool

	// last mouse selection
	mouseX int // expressed as horizontal position
	mouseY int // expressed as scanlines

	// guiLoopLock is used to protect anything that happens inside guiLoop()
	// care must be taken to activate the lock when those assets are accessed
	// outside of the guiLoop(), or for strong commentary to be present when it
	// is not required.
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
	err = sdl.Init(uint32(0))
	if err != nil {
		return nil, err
	}

	// pixelWidth is the number of tv pixels per color clock. we don't need to
	// worry about this again once we've created the window and set the scaling
	// for the renderer
	tv.pixelWidth = 2

	// pixel scale is the number of pixels each VCS "pixel" is to be occupy on
	// the screen
	tv.pixelScale = scale

	// SDL initialisation

	// SDL window - the correct size for the window will be determined below
	tv.window, err = sdl.CreateWindow("Gopher2600", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 0, 0, sdl.WINDOW_HIDDEN|sdl.WINDOW_OPENGL)
	if err != nil {
		return nil, err
	}

	// SDL renderer
	tv.renderer, err = sdl.CreateRenderer(tv.window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return nil, err
	}

	// make sure everything drawn through the renderer is correctly scaled
	err = tv.renderer.SetScale(float32(tv.pixelWidth)*tv.pixelScale, tv.pixelScale)
	if err != nil {
		return nil, err
	}

	// new screens
	playWidth := int32(tv.HeadlessTV.Spec.ClocksPerVisible)
	playHeight := int32(tv.HeadlessTV.Spec.ScanlinesPerVisible)

	tv.playScr, err = newScreen(playWidth, playHeight, tv.renderer)
	if err != nil {
		return nil, err
	}

	debugWidth := int32(tv.HeadlessTV.Spec.ClocksPerScanline)
	debugHeight := int32(tv.HeadlessTV.Spec.ScanlinesTotal)

	tv.dbgScr, err = newScreen(debugWidth, debugHeight, tv.renderer)
	if err != nil {
		return nil, err
	}

	tv.scr = tv.playScr
	tv.setWindowSize(tv.scr.width, tv.scr.height)

	// register callbacks from HeadlessTV to SDLTV
	tv.NewFrame = func() error {
		defer tv.scr.swapBuffer()
		return tv.update()
	}

	// update tv with a black image
	tv.scr.clearBuffer()
	err = tv.update()
	if err != nil {
		return nil, err
	}

	// begin "gui loop"
	go tv.guiLoop()

	// note that we've elected not to show the window on startup

	return tv, nil
}

// set window size scales the width and height correctly so that the VCS image
// is correct
func (tv *SDLTV) setWindowSize(width, height int32) {
	// *CRITICAL SECTION* *NOT REQUIRED*
	// called from NewSDLTV and then guiLoop() but never in parallel

	winWidth := int32(float32(width) * tv.pixelScale * float32(tv.pixelWidth))
	winHeight := int32(float32(height) * tv.pixelScale)
	tv.window.SetSize(winWidth, winHeight)
}

func (tv *SDLTV) setPixel(x, y int32, red, green, blue byte, pixels []byte) {
	i := (y*tv.scr.width + x) * tv.scr.pixelDepth
	if i < int32(len(pixels))-tv.scr.pixelDepth && i >= 0 {
		pixels[i] = red
		pixels[i+1] = green
		pixels[i+2] = blue
		pixels[i+3] = 255
	}
}

// update the gui so that it reflects changes to buffered data in the tv struct
func (tv *SDLTV) update() error {
	// *CRITICAL SECTION*
	// (R) tv.scr
	tv.guiLoopLock.Lock()
	defer tv.guiLoopLock.Unlock()

	var err error

	// clear image from rendered
	if tv.scr == tv.dbgScr {
		tv.renderer.SetDrawColor(5, 5, 5, 255)
	} else {
		tv.renderer.SetDrawColor(0, 0, 0, 255)
	}
	tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
	err = tv.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if tv.paused {
		err := tv.scr.fadeTexture.Update(nil, tv.scr.pixelsFade, int(tv.scr.width*tv.scr.pixelDepth))
		if err != nil {
			return err
		}
		err = tv.renderer.Copy(tv.scr.fadeTexture, nil, nil)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	err = tv.scr.texture.Update(nil, tv.scr.pixels, int(tv.scr.width*tv.scr.pixelDepth))
	if err != nil {
		return err
	}
	err = tv.renderer.Copy(tv.scr.texture, nil, nil)
	if err != nil {
		return err
	}

	if tv.scr == tv.dbgScr {
		// add screen boundary overlay
		tv.renderer.SetDrawColor(100, 100, 100, 25)
		tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
		tv.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(tv.Spec.ClocksPerHblank), H: int32(tv.Spec.ScanlinesTotal)})
		tv.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(tv.Spec.ClocksPerScanline), H: int32(tv.Spec.ScanlinesPerVBlank)})
		tv.renderer.FillRect(&sdl.Rect{X: 0, Y: int32(tv.Spec.ScanlinesTotal - tv.Spec.ScanlinesPerOverscan), W: int32(tv.Spec.ClocksPerScanline), H: int32(tv.Spec.ScanlinesPerOverscan)})

		// add cursor overlay only if tv is paused
		if tv.paused {
			tv.renderer.SetDrawColor(255, 255, 255, 100)
			tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
			cursorX := tv.PixelX(false)
			cursorY := tv.PixelY(false)
			if cursorX >= tv.Spec.ClocksPerScanline+tv.Spec.ClocksPerHblank {
				cursorX = 0
				cursorY++
			}
			tv.renderer.DrawRect(&sdl.Rect{X: int32(cursorX), Y: int32(cursorY), W: 2, H: 2})
		}
	}

	// finalise updating of screen

	// for windowed SDL, attempt to synchronise to 60fps (VSYNC hint only seems
	// to work if window is in full screen mode)
	time.Sleep(16666*time.Microsecond - time.Since(tv.lastFrameRender))
	tv.renderer.Present()
	tv.lastFrameRender = time.Now()

	return nil
}
