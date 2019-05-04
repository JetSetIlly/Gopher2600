package sdl

import (
	"github.com/veandco/go-sdl2/sdl"
)

// the number of bytes required for each screen pixel
// 4 == red + green + blue + alpha
const scrDepth int32 = 4

type screen struct {
	gtv *GUI

	window   *sdl.Window
	renderer *sdl.Renderer

	// maxWidth and maxHeight are the maximum possible sizes for the current tv
	// specification
	maxWidth  int32
	maxHeight int32
	maxMask   *sdl.Rect

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

	// whether we're using an unmasked screen
	unmasked bool

	// destRect and srcRect change depending on the value of unmasked
	srcRect  *sdl.Rect
	destRect *sdl.Rect

	// stabiliser to make sure image remains solid
	stb *screenStabiliser

	// overlay for screen showing metasignal information
	// -- always allocated but only used when tv.allowDebugging and
	// showMetaPixels is true
	metaPixels     *metaVideoOverlay
	showMetaPixels bool
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

	scr.maxWidth = int32(gtv.GetSpec().ClocksPerScanline)
	scr.maxHeight = int32(gtv.GetSpec().ScanlinesTotal)
	scr.maxMask = &sdl.Rect{X: 0, Y: 0, W: scr.maxWidth, H: scr.maxHeight}

	scr.playWidth = int32(gtv.GetSpec().ClocksPerVisible)
	scr.setPlayHeight(int32(gtv.GetSpec().ScanlinesPerVisible), int32(gtv.GetSpec().ScanlinesPerVBlank+gtv.GetSpec().ScanlinesPerVSync))

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
	scr.altPixels = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)
	scr.altPixelsFade = make([]byte, scr.maxWidth*scr.maxHeight*scrDepth)

	// new stabiliser
	scr.stb = newScreenStabiliser(scr)

	// new overlay
	scr.metaPixels, err = newMetaVideoOverlay(scr)
	if err != nil {
		return nil, err
	}

	return scr, nil
}

// setPlayHeight should be used when the number of visible scanlines change.
// when we want to show the overscan areas then we should use the setMasking()
// function.
func (scr *screen) setPlayHeight(scanlines int32, top int32) error {
	scr.playHeight = scanlines
	scr.playDstMask = &sdl.Rect{X: 0, Y: 0, W: scr.playWidth, H: scr.playHeight}
	scr.playSrcMask = &sdl.Rect{X: int32(scr.gtv.GetSpec().ClocksPerHblank), Y: top, W: scr.playWidth, H: scr.playHeight}

	return scr.setMasking(scr.unmasked)
}

// setScaling alters how big each pixel is on the physical screen. any change
// in the scale will cause the window size to change (via a call to
// the setMasking() function)
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

// setMasking alters which scanlines are actually shown. i.e. when unmasked, we
// can see the vblank and hblank areas of the screen. this can cause the window size
// to change
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
	if h < int32(float32(scr.gtv.GetSpec().ScanlinesPerVisible)*scr.pixelScale) {
		h = int32(float32(scr.gtv.GetSpec().ScanlinesPerVisible) * scr.pixelScale)
	}

	scr.window.SetSize(w, h)

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
		scr.renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(scr.gtv.GetSpec().ClocksPerHblank), H: int32(scr.gtv.GetSpec().ScanlinesTotal)})
	}

	// show metasignal overlay
	if scr.gtv.allowDebugging && scr.showMetaPixels {
		err = scr.metaPixels.update()
		if err != nil {
			return err
		}

		if scr.unmasked {
			err = scr.renderer.Copy(scr.metaPixels.texture, scr.srcRect, scr.destRect)
			if err != nil {
				return err
			}
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
		if x >= scr.gtv.GetSpec().ClocksPerScanline+scr.gtv.GetSpec().ClocksPerHblank {
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

func (scr *screen) clearPixels() {
	if scr.gtv.allowDebugging {
		// "fade" alternative pixels and clear
		swp := scr.altPixels
		scr.altPixels = scr.altPixelsFade
		scr.altPixelsFade = swp
		for i := 0; i < len(scr.altPixels); i++ {
			scr.altPixels[i] = 0
		}

		// clear pixels in additional overlays
		scr.metaPixels.clearPixels()

		// "fade" regular pixels
		swp = scr.pixels
		scr.pixels = scr.pixelsFade
		scr.pixelsFade = swp
	}

	// clear regular pixels
	for i := 0; i < len(scr.pixels); i++ {
		scr.pixels[i] = 0
	}
}
