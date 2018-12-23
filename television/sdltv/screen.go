package sdltv

import (
	"github.com/veandco/go-sdl2/sdl"
)

// the number of bytes required for each screen pixel
// 4 == red + green + blue + alpha
const scrDepth int32 = 4

type screen struct {
	tv *SDLTV

	window   *sdl.Window
	renderer *sdl.Renderer

	// maxWidth and maxHeight are the maximum possible sizes for the current tv
	// specification
	maxWidth  int32
	maxHeight int32
	maxMask   *sdl.Rect

	// pixels arrays are of maximum screen size - actual smalled screens are
	// masked appropriately
	pixels     []byte
	pixelsFade []byte

	// textures are used to present the pixels to the renderer
	texture     *sdl.Texture
	textureFade *sdl.Texture

	// the width of each VCS colour clock (in SDL pixels)
	pixelWidth int

	// by how much each pixel should be scaled
	pixelScale float32

	// play variables differ depending on the ROM
	playWidth   int32
	playHeight  int32
	playSrcMask *sdl.Rect
	playDstMask *sdl.Rect

	// whether we're using the max screen
	//  - destRect and srcRect change depending on the value of unmasked
	unmasked bool
	destRect *sdl.Rect
	srcRect  *sdl.Rect

	stability stability
}

func newScreen(tv *SDLTV) (*screen, error) {
	var err error

	scr := new(screen)
	scr.tv = tv

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

	scr.maxWidth = int32(tv.Spec.ClocksPerScanline)
	scr.maxHeight = int32(tv.Spec.ScanlinesTotal)
	scr.maxMask = &sdl.Rect{X: 0, Y: 0, W: scr.maxWidth, H: scr.maxHeight}

	scr.playWidth = int32(tv.Spec.ClocksPerVisible)
	scr.setPlayHeight(int32(tv.Spec.ScanlinesPerVisible))

	// pixelWidth is the number of tv pixels per color clock. we don't need to
	// worry about this again once we've created the window and set the scaling
	// for the renderer
	scr.pixelWidth = 2

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, regardless of whether the tv is paused
	// or unpaused
	scr.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(scr.maxWidth), int32(scr.maxHeight))
	if err != nil {
		return nil, err
	}
	scr.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	scr.textureFade, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(scr.maxWidth), int32(scr.maxHeight))
	if err != nil {
		return nil, err
	}
	scr.textureFade.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	scr.textureFade.SetAlphaMod(50)

	// our acutal screen data

	scr.pixels = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)
	scr.pixelsFade = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)

	return scr, nil
}

func (scr *screen) setPlayHeight(scanlines int32) error {
	scr.playHeight = scanlines
	scr.playDstMask = &sdl.Rect{X: 0, Y: 0, W: scr.playWidth, H: scr.playHeight}
	scr.playSrcMask = &sdl.Rect{X: int32(scr.tv.Spec.ClocksPerHblank), Y: int32(scr.tv.VBlankOff), W: scr.playWidth, H: scr.playHeight}

	return scr.setMasking(scr.unmasked)
}

func (scr *screen) setScaling(scale float32) error {
	// pixel scale is the number of pixels each VCS "pixel" is to be occupy on
	// the screen
	scr.pixelScale = scale

	// make sure everything drawn through the renderer is correctly scaled
	err := scr.renderer.SetScale(float32(scr.pixelWidth)*scr.pixelScale, scr.pixelScale)
	if err != nil {
		return err
	}

	scr.setMasking(scr.unmasked)

	return nil
}

func (scr *screen) setMasking(unmasked bool) error {
	var w, h int32

	scr.unmasked = unmasked

	if scr.unmasked {
		w = int32(float32(scr.maxWidth) * scr.pixelScale * float32(scr.pixelWidth))
		h = int32(float32(scr.maxHeight) * scr.pixelScale)
		scr.destRect = scr.maxMask
		scr.srcRect = scr.maxMask
	} else {
		w = int32(float32(scr.playWidth) * scr.pixelScale * float32(scr.pixelWidth))
		h = int32(float32(scr.playHeight) * scr.pixelScale)
		scr.destRect = scr.playDstMask
		scr.srcRect = scr.playSrcMask
	}

	// minimum window size
	if h < int32(float32(scr.tv.Spec.ScanlinesPerVisible)*scr.pixelScale) {
		h = int32(float32(scr.tv.Spec.ScanlinesPerVisible) * scr.pixelScale)
	}

	scr.window.SetSize(w, h)

	return nil
}

func (scr *screen) toggleMasking() {
	scr.setMasking(!scr.unmasked)
}

func (scr *screen) setPixel(x, y int32, red, green, blue byte) error {
	i := (y*scr.maxWidth + x) * scrDepth
	if i < int32(len(scr.pixels))-scrDepth && i >= 0 {
		scr.pixels[i] = red
		scr.pixels[i+1] = green
		scr.pixels[i+2] = blue
		scr.pixels[i+3] = 255
	}

	return nil
}

func (scr *screen) update(paused bool) error {
	var err error

	// before we go any further, check to frame stability
	err = scr.checkStability()
	if err != nil {
		return err
	}

	// clear image from rendered
	scr.renderer.SetDrawColor(5, 5, 5, 255)
	scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))
	err = scr.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if paused {
		err = scr.textureFade.Update(nil, scr.pixelsFade, int(scr.maxWidth*scrDepth))
		if err != nil {
			return err
		}
		err = scr.renderer.Copy(scr.textureFade, scr.srcRect, scr.destRect)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	// - if tv is paused this overwrites the faded image (drawn above) up to
	// the pixel where the current frame has reached
	err = scr.texture.Update(nil, scr.pixels, int(scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}
	err = scr.renderer.Copy(scr.texture, scr.srcRect, scr.destRect)
	if err != nil {
		return err
	}

	// draw masks
	if scr.unmasked {
		scr.renderer.SetDrawColor(15, 15, 15, 100)
		scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

		// hblank mask
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(scr.tv.Spec.ClocksPerHblank), H: scr.srcRect.H})
	} else {
		scr.renderer.SetDrawColor(0, 0, 0, 255)
		scr.renderer.SetDrawBlendMode(sdl.BlendMode(sdl.BLENDMODE_NONE))
	}

	// top vblank mask
	h := int32(scr.tv.VBlankOff) - scr.srcRect.Y
	scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: scr.srcRect.W, H: h})

	// bottom vblank mask
	y := int32(scr.tv.VBlankOn) - scr.srcRect.Y
	h = int32(scr.tv.Spec.ScanlinesTotal - scr.tv.VBlankOn)
	scr.renderer.FillRect(&sdl.Rect{X: 0, Y: y, W: scr.srcRect.W, H: h})

	// add cursor if tv is paused
	// -- drawing last so that cursor isn't masked or drawn behind any alpha
	// layers
	if paused {
		// cursor coordinates
		x := scr.tv.HorizPos.Value().(int) + scr.tv.Spec.ClocksPerHblank
		y := scr.tv.Scanline.Value().(int)

		// cursor is one step ahead of pixel -- move to new scanline if
		// necessary
		if x >= scr.tv.Spec.ClocksPerScanline+scr.tv.Spec.ClocksPerHblank {
			x = 0
			y++
		}

		// note whether cursor is "off-screen" (according to current masking)
		offscreenCursorPos := false

		// adjust coordinates if screen is masked
		if !scr.unmasked {
			x -= scr.tv.Spec.ClocksPerHblank
			y -= scr.tv.Spec.ScanlinesPerVBlank + scr.tv.Spec.ScanlinesPerVSync
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

		// cursor is a 2x2 rectangle
		scr.renderer.DrawRect(&sdl.Rect{X: int32(x), Y: int32(y), W: 2, H: 2})
	}

	return nil
}

func (scr *screen) swapPixels() {
	// swap which pixel buffer we're using in time for next roung of pixel
	// plotting
	swp := scr.pixels
	scr.pixels = scr.pixelsFade
	scr.pixelsFade = swp

	// clear pixels
	for i := 0; i < len(scr.pixels); i++ {
		scr.pixels[i] = 0
	}
}
