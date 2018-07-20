package sdltv

import "github.com/veandco/go-sdl2/sdl"

type screen struct {
	width      int32
	height     int32
	pixelDepth int32

	texture     *sdl.Texture
	fadeTexture *sdl.Texture

	pixels     []byte
	pixelsFade []byte
	pixelSwapA []byte
	pixelSwapB []byte
}

func newScreen(width, height int32, renderer *sdl.Renderer) (*screen, error) {
	var err error

	scr := new(screen)
	scr.width = width
	scr.height = height
	scr.pixelDepth = 4

	// screen texture is used to draw the pixels onto the sdl window (by the
	// renderer). it is used evey frame, regardless of whether the tv is paused
	// or unpaused
	scr.texture, err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, scr.width, scr.height)
	if err != nil {
		return nil, err
	}
	scr.texture.SetBlendMode(sdl.BLENDMODE_BLEND)

	// fade texture is only used when the tv is paused. it is used to display
	// the previous frame as a guide, in case the current frame is not completely
	// rendered
	scr.fadeTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, scr.width, scr.height)
	if err != nil {
		return nil, err
	}
	scr.fadeTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	scr.fadeTexture.SetAlphaMod(50)

	// our acutal screen data
	scr.pixelSwapA = make([]byte, scr.width*scr.height*scr.pixelDepth)
	scr.pixelSwapB = make([]byte, scr.width*scr.height*scr.pixelDepth)
	scr.pixels = scr.pixelSwapA
	scr.pixelsFade = scr.pixelSwapB

	return scr, nil
}

func (scr *screen) swapBuffer() {
	// swap which pixel buffer we're using
	swp := scr.pixels
	scr.pixels = scr.pixelsFade
	scr.pixelsFade = swp
	scr.clearBuffer()
}

func (scr *screen) clearBuffer() {
	for y := int32(0); y < scr.height; y++ {
		for x := int32(0); x < scr.width; x++ {
			i := (y*scr.width + x) * scr.pixelDepth
			scr.pixels[i] = 0
			scr.pixels[i+1] = 0
			scr.pixels[i+2] = 0
			scr.pixels[i+3] = 0
		}
	}
}
