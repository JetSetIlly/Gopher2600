package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

const pixelDepth = 4
const pixelWidth = 2.0

type pixels struct {
	scr *SdlDebug

	renderer *sdl.Renderer

	// textures are used to present the pixels to the renderer
	texture     *sdl.Texture
	textureFade *sdl.Texture

	// maxWidth and maxHeight are the maximum possible sizes for the current tv
	// specification
	maxWidth  int32
	maxHeight int32
	maxMask   *sdl.Rect

	// by how much each pixel should be scaled
	pixelScaleY float32
	pixelScaleX float32

	// play variables differ depending on the ROM
	playWidth   int32
	playHeight  int32
	playSrcMask *sdl.Rect
	playDstMask *sdl.Rect
	playTop     int32

	// destRect and srcRect change depending on the value of unmasked
	srcRect  *sdl.Rect
	destRect *sdl.Rect

	// whether we're using an unmasked screen
	unmasked bool

	// the remaining attributes change every update

	// last plot coordinates. used for:
	// - drawing cursor
	// - adding metaPixels
	lastX int32
	lastY int32

	// pixels arrays are of maximum screen size - actual smaller play screens
	// are masked appropriately
	pixels     []byte
	pixelsFade []byte

	// altPixels mirrors the pixels array with alternative color palette
	// - useful for switching between regular and debug colors
	// - allocated but only used if scr.allowDebugging and useAltPixels is true
	altPixels     []byte
	altPixelsFade []byte
	useAltPixels  bool

	// metaPixels for screen showing additional debugging information
	// - always allocated but only used when tv.allowDebugging and
	// overlayActive are true
	metaPixels    *metapixelOverlay
	useMetaPixels bool
}

func newScreen(scr *SdlDebug) (*pixels, error) {
	var err error

	pxl := pixels{scr: scr}

	// SDL renderer
	pxl.renderer, err = sdl.CreateRenderer(scr.window, -1, uint32(sdl.RENDERER_ACCELERATED)|uint32(sdl.RENDERER_PRESENTVSYNC))
	if err != nil {
		return nil, errors.New(errors.SDL, err)
	}

	return &pxl, nil
}

func (pxl *pixels) reset() error {
	pxl.newFrame()
	pxl.lastX = 0
	pxl.lastY = 0
	return nil
}

// initialise screen sets up SDL according to the current television
// specification. it is called on startup but also whenever a change in the TV
// spec is requested
func (pxl *pixels) resize(topScanline, numScanlines int) error {
	var err error

	pxl.maxWidth = int32(television.HorizClksScanline)
	pxl.maxHeight = int32(pxl.scr.GetSpec().ScanlinesTotal)
	pxl.maxMask = &sdl.Rect{X: 0, Y: 0, W: pxl.maxWidth, H: pxl.maxHeight}

	pxl.playTop = int32(topScanline)
	pxl.playWidth = int32(television.HorizClksVisible)
	pxl.setPlayArea(int32(numScanlines), int32(topScanline))

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, regardless of whether the tv is paused
	// or unpaused
	pxl.texture, err = pxl.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(pxl.maxWidth), int32(pxl.maxHeight))
	if err != nil {
		return errors.New(errors.SDL, err)
	}
	pxl.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	pxl.textureFade, err = pxl.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(pxl.maxWidth), int32(pxl.maxHeight))
	if err != nil {
		return errors.New(errors.SDL, err)
	}
	pxl.textureFade.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	pxl.textureFade.SetAlphaMod(50)

	// our acutal screen data
	pxl.pixels = make([]byte, pxl.maxWidth*pxl.maxHeight*pixelDepth)
	pxl.pixelsFade = make([]byte, pxl.maxWidth*pxl.maxHeight*pixelDepth)
	pxl.altPixels = make([]byte, pxl.maxWidth*pxl.maxHeight*pixelDepth)
	pxl.altPixelsFade = make([]byte, pxl.maxWidth*pxl.maxHeight*pixelDepth)

	// new overlay
	pxl.metaPixels, err = newMetapixelOverlay(pxl)
	if err != nil {
		return errors.New(errors.SDL, err)
	}

	return nil
}

// setPlayArea defines the limits of the "play area"
func (pxl *pixels) setPlayArea(scanlines int32, top int32) {
	pxl.playHeight = scanlines
	pxl.playDstMask = &sdl.Rect{X: 0, Y: 0, W: pxl.playWidth, H: pxl.playHeight}
	pxl.playSrcMask = &sdl.Rect{X: int32(television.HorizClksHBlank), Y: top, W: pxl.playWidth, H: pxl.playHeight}
	pxl.setMasking(pxl.unmasked)
}

// setScaling alters how big each pixel is on the physical screen. any change
// in the scale will cause the window size to change (via a call to
// the setMasking() function)
func (pxl *pixels) setScaling(scale float32) error {
	// pixel scale is the number of pixels each VCS "pixel" is to be occupy on
	// the screen
	pxl.pixelScaleY = scale
	pxl.pixelScaleX = scale * pxl.scr.GetSpec().AspectBias

	// make sure everything drawn through the renderer is correctly scaled
	err := pxl.renderer.SetScale(pixelWidth*pxl.pixelScaleX, pxl.pixelScaleY)
	if err != nil {
		return err
	}

	pxl.setMasking(pxl.unmasked)

	return nil
}

// setMasking alters which scanlines are actually shown. i.e. when unmasked, we
// can see the vblank and hblank areas of the screen. this can cause the window size
// to change
func (pxl *pixels) setMasking(unmasked bool) {
	var w, h int32

	pxl.unmasked = unmasked

	if pxl.unmasked {
		w = int32(float32(pxl.maxWidth) * pxl.pixelScaleX * pixelWidth)
		h = int32(float32(pxl.maxHeight) * pxl.pixelScaleY)
		pxl.destRect = pxl.maxMask
		pxl.srcRect = pxl.maxMask
	} else {
		w = int32(float32(pxl.playWidth) * pxl.pixelScaleX * pixelWidth)
		h = int32(float32(pxl.playHeight) * pxl.pixelScaleY)
		pxl.destRect = pxl.playDstMask
		pxl.srcRect = pxl.playSrcMask
	}

	// BUG: SetSize causes window to gain focus
	cw, ch := pxl.scr.window.GetSize()
	if cw != w || ch != h {
		pxl.scr.window.SetSize(w, h)
	}
}

func (pxl *pixels) setRegPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return pxl.setPixel(&pxl.pixels, x, y, red, green, blue, vblank)
}

func (pxl *pixels) setAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return pxl.setPixel(&pxl.altPixels, x, y, red, green, blue, vblank)
}

func (pxl *pixels) setPixel(pixels *[]byte, x, y int32, red, green, blue byte, vblank bool) error {
	pxl.lastX = x
	pxl.lastY = y

	if !vblank {
		i := (y*pxl.maxWidth + x) * pixelDepth
		if i < int32(len(pxl.pixels))-pixelDepth && i >= 0 {
			(*pixels)[i] = red
			(*pixels)[i+1] = green
			(*pixels)[i+2] = blue
			(*pixels)[i+3] = 255
		}
	}

	return nil
}

func (pxl *pixels) update() error {
	var err error

	// clear image from rendered. using a non-video-black color if screen is
	// unmasked
	if pxl.unmasked {
		pxl.renderer.SetDrawColor(5, 5, 5, 255)
	} else {
		pxl.renderer.SetDrawColor(0, 0, 0, 255)
	}
	pxl.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))
	err = pxl.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if pxl.scr.paused {
		if pxl.useAltPixels {
			err = pxl.textureFade.Update(nil, pxl.altPixelsFade, int(pxl.maxWidth*pixelDepth))
		} else {
			err = pxl.textureFade.Update(nil, pxl.pixelsFade, int(pxl.maxWidth*pixelDepth))
		}
		if err != nil {
			return err
		}
		err = pxl.renderer.Copy(pxl.textureFade, pxl.srcRect, pxl.destRect)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	// - decide which set of pixels to use
	// - if tv is paused this overwrites the faded image (drawn above) up to
	// the pixel where the current frame has reached
	if pxl.useAltPixels {
		err = pxl.texture.Update(nil, pxl.altPixels, int(pxl.maxWidth*pixelDepth))
	} else {
		err = pxl.texture.Update(nil, pxl.pixels, int(pxl.maxWidth*pixelDepth))
	}
	if err != nil {
		return err
	}

	err = pxl.renderer.Copy(pxl.texture, pxl.srcRect, pxl.destRect)
	if err != nil {
		return err
	}

	// show hblank overlay
	if pxl.unmasked {
		pxl.renderer.SetDrawColor(100, 100, 100, 20)
		pxl.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
		pxl.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(television.HorizClksHBlank), H: int32(pxl.scr.GetSpec().ScanlinesTotal)})
	}

	// show overlay
	if pxl.useMetaPixels {
		err = pxl.metaPixels.update(pxl.scr.paused)
		if err != nil {
			return err
		}
	}

	// add cursor if tv is paused
	// - drawing last so that cursor isn't masked
	if pxl.scr.paused {
		// cursor coordinates
		x := int(pxl.lastX)
		y := int(pxl.lastY)

		// cursor is one step ahead of pixel -- move to new scanline if
		// necessary
		if x >= television.HorizClksScanline {
			x = 0
			y++
		}

		// note whether cursor is "off-screen" (according to current masking)
		offscreenCursorPos := false

		// adjust coordinates if pxleen is masked
		if !pxl.unmasked {
			x -= int(pxl.srcRect.X)
			y -= int(pxl.srcRect.Y)

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
			pxl.renderer.SetDrawColor(100, 100, 255, 100)
		} else {
			pxl.renderer.SetDrawColor(255, 255, 255, 100)
		}
		pxl.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))

		// leave the current pixel visible at the top-left corner of the cursor
		pxl.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y), W: 1, H: 1})
		pxl.renderer.DrawRect(&sdl.Rect{X: int32(x + 1), Y: int32(y + 1), W: 1, H: 1})
		pxl.renderer.DrawRect(&sdl.Rect{X: int32(x), Y: int32(y + 1), W: 1, H: 1})
	}

	pxl.renderer.Present()

	return nil
}

func (pxl *pixels) newFrame() {
	// swap pixel array with pixelsFade array
	// -- note that we don't do this with the texture instead because
	// updating the the extra texture if we don't need to (faded pixels
	// only show when the emulation is paused) is expensive
	swp := pxl.pixels
	pxl.pixels = pxl.pixelsFade
	pxl.pixelsFade = swp

	// clear pixels in overlay
	pxl.metaPixels.newFrame()

	// swap pixel array with pixelsFade array
	// -- see comment above
	swp = pxl.altPixels
	pxl.altPixels = pxl.altPixelsFade
	pxl.altPixelsFade = swp

	// clear altpixels
	for i := 0; i < len(pxl.altPixels); i++ {
		pxl.altPixels[i] = 0
	}

	// clear regular pixels
	for i := 0; i < len(pxl.pixels); i++ {
		pxl.pixels[i] = 0
	}
}
