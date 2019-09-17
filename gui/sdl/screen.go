package sdl

import (
	"gopher2600/errors"
	"gopher2600/performance/limiter"
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// the number of bytes required for each screen pixel
// 4 == red + green + blue + alpha
const scrDepth int32 = 4

type screen struct {
	gtv  *GUI
	spec *television.Specification

	// regulates how often the screen is updated
	fpsLimiter *limiter.FpsLimiter

	window   *sdl.Window
	renderer *sdl.Renderer

	// maxWidth and maxHeight are the maximum possible sizes for the current tv
	// specification
	maxWidth  int32
	maxHeight int32
	maxMask   *sdl.Rect

	// textures are used to present the pixels to the renderer
	texture     *sdl.Texture
	textureFade *sdl.Texture

	// the width of each VCS colour clock (in SDL pixels)
	pixelWidth int

	// by how much each pixel should be scaled
	pixelScaleY float32
	pixelScaleX float32

	// play variables differ depending on the ROM
	playWidth   int32
	playHeight  int32
	playSrcMask *sdl.Rect
	playDstMask *sdl.Rect

	// destRect and srcRect change depending on the value of unmasked
	srcRect  *sdl.Rect
	destRect *sdl.Rect

	// stabiliser to make sure image remains solid
	stb *screenStabiliser

	// whether we're using an unmasked screen
	// -- changed by user request
	unmasked bool

	// the remaining attributes change every update

	// last plot coordinates
	lastX int32
	lastY int32

	// pixels arrays are of maximum screen size - actual smaller play screens
	// are masked appropriately
	pixels     []byte
	pixelsFade []byte

	// altPixels mirrors the pixels array with alternative color palette
	// -- useful for switching between regular and debug colors
	// -- allocated but only used if gtv.allowDebugging and useAltPixels is true
	altPixels     []byte
	altPixelsFade []byte
	useAltPixels  bool

	// overlay for screen showing metasignal information
	// -- always allocated but only used when tv.allowDebugging and
	// showMetaVideo are true
	metaVideo     *metaVideoOverlay
	showMetaVideo bool
}

func newScreen(gtv *GUI) (*screen, error) {
	var err error

	scr := new(screen)
	scr.gtv = gtv

	// SDL window - the correct size for the window will be determined below
	scr.window, err = sdl.CreateWindow("Gopher2600", int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED), 0, 0, uint32(sdl.WINDOW_HIDDEN)|uint32(sdl.WINDOW_OPENGL))
	if err != nil {
		return nil, err
	}

	// SDL renderer
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED)|uint32(sdl.RENDERER_PRESENTVSYNC))
	if err != nil {
		return nil, err
	}

	// set attributes that depend on the television specification
	err = scr.changeTVSpec()
	if err != nil {
		return nil, err
	}

	// new stabiliser
	scr.stb = newScreenStabiliser(scr)

	return scr, nil
}

func (scr *screen) changeTVSpec() error {
	var err error

	scr.spec = scr.gtv.GetSpec()

	scr.maxWidth = int32(television.ClocksPerScanline)
	scr.maxHeight = int32(scr.spec.ScanlinesTotal)
	scr.maxMask = &sdl.Rect{X: 0, Y: 0, W: scr.maxWidth, H: scr.maxHeight}

	scr.playWidth = int32(television.ClocksPerVisible)
	scr.setPlayArea(int32(scr.spec.ScanlinesPerVisible), int32(scr.spec.ScanlinesPerVBlank+scr.spec.ScanlinesPerVSync))

	// pixelWidth is the number of tv pixels per color clock. we don't need to
	// worry about this again once we've created the window and set the scaling
	// for the renderer
	scr.pixelWidth = 2

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, regardless of whether the tv is paused
	// or unpaused
	scr.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(scr.maxWidth), int32(scr.maxHeight))
	if err != nil {
		return err
	}
	scr.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	scr.textureFade, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(scr.maxWidth), int32(scr.maxHeight))
	if err != nil {
		return err
	}
	scr.textureFade.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	scr.textureFade.SetAlphaMod(50)

	// our acutal screen data
	scr.pixels = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)
	scr.pixelsFade = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)
	scr.altPixels = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)
	scr.altPixelsFade = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)

	// frame limiter
	scr.fpsLimiter, err = limiter.NewFPSLimiter(int(scr.spec.FramesPerSecond))
	if err != nil {
		return errors.NewFormattedError(errors.SDL, err)
	}

	// new overlay
	scr.metaVideo, err = newMetaVideoOverlay(scr)
	if err != nil {
		return err
	}

	return nil
}

// setPlayArea defines the limits of the "play area"
func (scr *screen) setPlayArea(scanlines int32, top int32) error {
	scr.playHeight = scanlines
	scr.playDstMask = &sdl.Rect{X: 0, Y: 0, W: scr.playWidth, H: scr.playHeight}
	scr.playSrcMask = &sdl.Rect{X: int32(television.ClocksPerHblank), Y: top, W: scr.playWidth, H: scr.playHeight}

	return scr.setMasking(scr.unmasked)
}

// adjustPlayArea is used to move the play area up/down by the specified amount
func (scr *screen) adjustPlayArea(adjust int32) {
	// !!TODO: make screen adjustment optional
	scr.playSrcMask.Y += adjust
}

// setScaling alters how big each pixel is on the physical screen. any change
// in the scale will cause the window size to change (via a call to
// the setMasking() function)
func (scr *screen) setScaling(scale float32) error {
	// pixel scale is the number of pixels each VCS "pixel" is to be occupy on
	// the screen
	scr.pixelScaleY = scale
	scr.pixelScaleX = scale * scr.gtv.GetSpec().AspectBias

	// make sure everything drawn through the renderer is correctly scaled
	err := scr.renderer.SetScale(float32(scr.pixelWidth)*scr.pixelScaleX, scr.pixelScaleY)
	if err != nil {
		return err
	}

	scr.setMasking(scr.unmasked)

	return nil
}

// setMasking alters which scanlines are actually shown. i.e. when unmasked, we
// can see the vblank and hblank areas of the screen. this can cause the window size
// to change
func (scr *screen) setMasking(unmasked bool) error {
	var w, h int32

	scr.unmasked = unmasked

	if scr.unmasked {
		w = int32(float32(scr.maxWidth) * scr.pixelScaleX * float32(scr.pixelWidth))
		h = int32(float32(scr.maxHeight) * scr.pixelScaleY)
		scr.destRect = scr.maxMask
		scr.srcRect = scr.maxMask
	} else {
		w = int32(float32(scr.playWidth) * scr.pixelScaleX * float32(scr.pixelWidth))
		h = int32(float32(scr.playHeight) * scr.pixelScaleY)
		scr.destRect = scr.playDstMask
		scr.srcRect = scr.playSrcMask
	}

	cw, ch := scr.window.GetSize()
	if cw != w || ch != h {
		// BUG: SetSize causes window to gain focus
		scr.window.SetSize(w, h)
	}

	return nil
}

func (scr *screen) setRegPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return scr.setPixel(&scr.pixels, x, y, red, green, blue, vblank)
}

func (scr *screen) setAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return scr.setPixel(&scr.altPixels, x, y, red, green, blue, vblank)
}

func (scr *screen) setPixel(pixels *[]byte, x, y int32, red, green, blue byte, vblank bool) error {
	scr.lastX = x
	scr.lastY = y

	// do not plot pixel if VBLANK is on. some ROMs use VBLANK to output black,
	// rather than having to play around with the current color of the sprites
	//
	// ROMs affected:
	//	* Custer's Revenge
	//	* Ladybug
	if !vblank {
		i := (y*scr.maxWidth + x) * scrDepth
		if i < int32(len(scr.pixels))-scrDepth && i >= 0 {
			(*pixels)[i] = red
			(*pixels)[i+1] = green
			(*pixels)[i+2] = blue
			(*pixels)[i+3] = 255
		}
	}

	return nil
}

func (scr *screen) update(paused bool) error {
	// enforce a maximum frames-per-second
	scr.fpsLimiter.Wait()

	var err error

	// clear image from rendered. using a non-video-black color if screen is
	// unmasked
	if scr.unmasked {
		scr.renderer.SetDrawColor(5, 5, 5, 255)
	} else {
		scr.renderer.SetDrawColor(0, 0, 0, 255)
	}
	scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))
	err = scr.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if paused {
		if scr.gtv.allowDebugging && scr.useAltPixels {
			err = scr.textureFade.Update(nil, scr.altPixelsFade, int(scr.maxWidth*scrDepth))
		} else {
			err = scr.textureFade.Update(nil, scr.pixelsFade, int(scr.maxWidth*scrDepth))
		}
		if err != nil {
			return err
		}
		err = scr.renderer.Copy(scr.textureFade, scr.srcRect, scr.destRect)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	// - decide which set of pixels to use
	// - if tv is paused this overwrites the faded image (drawn above) up to
	// the pixel where the current frame has reached
	if scr.gtv.allowDebugging && scr.useAltPixels {
		err = scr.texture.Update(nil, scr.altPixels, int(scr.maxWidth*scrDepth))
	} else {
		err = scr.texture.Update(nil, scr.pixels, int(scr.maxWidth*scrDepth))
	}
	if err != nil {
		return err
	}

	err = scr.renderer.Copy(scr.texture, scr.srcRect, scr.destRect)
	if err != nil {
		return err
	}

	// show hblank overlay
	if scr.unmasked {
		scr.renderer.SetDrawColor(100, 100, 100, 20)
		scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(television.ClocksPerHblank), H: int32(scr.spec.ScanlinesTotal)})
	}

	// show metasignal overlay
	if scr.gtv.allowDebugging && scr.showMetaVideo {
		err = scr.metaVideo.update(paused)
		if err != nil {
			return err
		}
	}

	// add cursor if tv is paused
	// - drawing last so that cursor isn't masked
	if paused {
		// cursor coordinates
		x := int(scr.lastX)
		y := int(scr.lastY)

		// cursor is one step ahead of pixel -- move to new scanline if
		// necessary
		if x >= television.ClocksPerScanline {
			x = 0
			y++
		}

		// note whether cursor is "off-screen" (according to current masking)
		offscreenCursorPos := false

		// adjust coordinates if screen is masked
		if !scr.unmasked {
			x -= int(scr.srcRect.X)
			y -= int(scr.srcRect.Y)

			if x < 0 {
				offscreenCursorPos = true
				x = 0
			}
			if y < 0 {
				offscreenCursorPos = true
				y = 0
			}
		}

		// cursor color depends on whether cursor is off-screen or not
		if offscreenCursorPos {
			scr.renderer.SetDrawColor(100, 100, 255, 100)
		} else {
			scr.renderer.SetDrawColor(255, 255, 255, 100)
		}
		scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))

		// leave the current pixel visible at the top-left corner of the cursor
		scr.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y), W: 1, H: 1})
		scr.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y + 1), W: 1, H: 1})
		scr.renderer.DrawRect(&sdl.Rect{X: int32(x), Y: int32(y + 1), W: 1, H: 1})
	}

	return nil
}

func (scr *screen) newFrame() {
	if scr.gtv.allowDebugging {
		// swap pixel array with pixelsFade array
		// -- note that we don't do this with the texture instead because
		// updating the the extra texture if we don't need to (faded pixels
		// only show when the emulation is paused) is expensive
		swp := scr.pixels
		scr.pixels = scr.pixelsFade
		scr.pixelsFade = swp

		// clear pixels in metavideo overlay
		scr.metaVideo.newFrame()

		// swap pixel array with pixelsFade array
		// -- see comment above
		swp = scr.altPixels
		scr.altPixels = scr.altPixelsFade
		scr.altPixelsFade = swp

		// clear altpixels
		for i := 0; i < len(scr.altPixels); i++ {
			scr.altPixels[i] = 0
		}
	}

	// clear regular pixels
	for i := 0; i < len(scr.pixels); i++ {
		scr.pixels[i] = 0
	}
}
