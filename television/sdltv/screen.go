package sdltv

import (
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

type screen struct {
	tv *television.HeadlessTV

	window   *sdl.Window
	renderer *sdl.Renderer

	playWidth  int32
	playHeight int32
	maxWidth   int32
	maxHeight  int32
	depth      int32
	pitch      int

	// the width of each VCS colour clock (in SDL pixels)
	pixelWidth int

	// by how much each pixel should be scaled
	pixelScale float32

	noMask      *sdl.Rect
	maskRectDst *sdl.Rect
	maskRectSrc *sdl.Rect

	texture     *sdl.Texture
	fadeTexture *sdl.Texture

	pixelsA    []byte
	pixelsB    []byte
	pixels     []byte
	pixelsFade []byte

	// whether we're using the max screen
	//  - destRect and srcRect change depending on the value of unmasked
	unmasked bool
	destRect *sdl.Rect
	srcRect  *sdl.Rect
}

func newScreen(tv *television.HeadlessTV) (*screen, error) {
	var err error

	scr := new(screen)
	scr.tv = tv

	// SDL window - the correct size for the window will be determined below
	scr.window, err = sdl.CreateWindow("Gopher2600", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 0, 0, sdl.WINDOW_HIDDEN|sdl.WINDOW_OPENGL)
	if err != nil {
		return nil, err
	}

	// SDL renderer
	scr.renderer, err = sdl.CreateRenderer(scr.window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return nil, err
	}

	scr.playWidth = int32(tv.Spec.ClocksPerVisible)
	scr.playHeight = int32(tv.Spec.ScanlinesPerVisible)
	scr.maxWidth = int32(tv.Spec.ClocksPerScanline)
	scr.maxHeight = int32(tv.Spec.ScanlinesTotal)
	scr.depth = 4
	scr.pitch = int(scr.maxWidth * scr.depth)

	// pixelWidth is the number of tv pixels per color clock. we don't need to
	// worry about this again once we've created the window and set the scaling
	// for the renderer
	scr.pixelWidth = 2

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, regardless of whether the tv is paused
	// or unpaused
	scr.texture, err = scr.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, scr.maxWidth, scr.maxHeight)
	if err != nil {
		return nil, err
	}
	scr.texture.SetBlendMode(sdl.BLENDMODE_BLEND)

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	scr.fadeTexture, err = scr.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, scr.maxWidth, scr.maxHeight)
	if err != nil {
		return nil, err
	}
	scr.fadeTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	scr.fadeTexture.SetAlphaMod(50)

	// our acutal screen data
	scr.pixelsA = make([]byte, scr.maxWidth*scr.maxHeight*scr.depth)
	scr.pixelsB = make([]byte, scr.maxWidth*scr.maxHeight*scr.depth)

	scr.pixels = scr.pixelsA
	scr.pixelsFade = scr.pixelsB

	scr.noMask = &sdl.Rect{X: 0, Y: 0, W: scr.maxWidth, H: scr.maxHeight}
	scr.maskRectDst = &sdl.Rect{X: 0, Y: 0, W: scr.playWidth, H: scr.playHeight}
	scr.maskRectSrc = &sdl.Rect{X: int32(tv.Spec.ClocksPerHblank), Y: int32(tv.Spec.ScanlinesPerVBlank + tv.Spec.ScanlinesPerVSync), W: scr.playWidth, H: scr.playHeight}

	return scr, nil
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

func (scr *screen) setMasking(unmasked bool) {
	var w, h int32

	scr.unmasked = unmasked

	if scr.unmasked {
		w = int32(float32(scr.maxWidth) * scr.pixelScale * float32(scr.pixelWidth))
		h = int32(float32(scr.maxHeight) * scr.pixelScale)
		scr.destRect = scr.noMask
		scr.srcRect = scr.noMask
	} else {
		w = int32(float32(scr.playWidth) * scr.pixelScale * float32(scr.pixelWidth))
		h = int32(float32(scr.playHeight) * scr.pixelScale)
		scr.destRect = scr.maskRectDst
		scr.srcRect = scr.maskRectSrc
	}

	scr.window.SetSize(w, h)
}

func (scr *screen) toggleMasking() {
	scr.setMasking(!scr.unmasked)
}

func (scr *screen) setPixel(x, y int32, red, green, blue byte) {
	i := (y*scr.maxWidth + x) * scr.depth
	if i < int32(len(scr.pixels))-scr.depth && i >= 0 {
		scr.pixels[i] = red
		scr.pixels[i+1] = green
		scr.pixels[i+2] = blue
		scr.pixels[i+3] = 255
	}
}

func (scr *screen) update(paused bool) error {
	var err error

	// clear image from rendered
	scr.renderer.SetDrawColor(5, 10, 5, 255)
	scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
	err = scr.renderer.Clear()
	if err != nil {
		return err
	}

	// if tv is paused then show the previous frame's faded image
	if paused {
		err = scr.fadeTexture.Update(nil, scr.pixelsFade, scr.pitch)
		if err != nil {
			return err
		}
		err = scr.renderer.Copy(scr.fadeTexture, scr.srcRect, scr.destRect)
		if err != nil {
			return err
		}
	}

	// show current frame's pixels
	// - if tv is paused this overwrites the faded image (drawn above) up to
	// the pixel where the current frame has reached
	err = scr.texture.Update(nil, scr.pixels, scr.pitch)
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
		scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

		// hblank mask
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(scr.tv.Spec.ClocksPerHblank), H: scr.srcRect.H})
	} else {
		scr.renderer.SetDrawColor(0, 0, 0, 255)
		scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
	}

	// top vblank mask
	if scr.tv.VBlankOff < scr.tv.VBlankOn {
		h := int32(scr.tv.VBlankOff) - scr.srcRect.Y
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: scr.srcRect.W, H: h})

		// bottom vblank mask
		y := int32(scr.tv.VBlankOn) - scr.srcRect.Y
		h = int32(scr.tv.Spec.ScanlinesTotal - scr.tv.VBlankOn)
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: y, W: scr.srcRect.W, H: h})
	}

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
		accurateCursorPos := true

		// adjust coordinates if screen is masked
		if !scr.unmasked {
			x -= scr.tv.Spec.ClocksPerHblank
			y -= scr.tv.Spec.ScanlinesPerVBlank + scr.tv.Spec.ScanlinesPerVSync
			if x < 0 {
				accurateCursorPos = false
				x = 0
			}
			if y < 0 {
				accurateCursorPos = false
				y = 0
			}
		}

		// cursor color depends on whether cursor positioning is accurate
		if !accurateCursorPos {
			scr.renderer.SetDrawColor(100, 100, 255, 100)
		} else {
			scr.renderer.SetDrawColor(255, 255, 255, 100)
		}
		scr.renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)

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
