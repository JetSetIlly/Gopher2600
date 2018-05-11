package television

import (
	"fmt"
	"gopher2600/hardware/tia/video"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	HeadlessTV

	width  int
	height int
	depth  int

	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	pixels   []byte
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
		tv.clear()
		return nil
	}
	tv.forceUpdate = func() error {
		return tv.update()
	}

	// image dimensions
	tv.width = tv.HeadlessTV.spec.clocksPerScanline
	tv.height = tv.HeadlessTV.spec.scanlinesTotal
	tv.depth = 4

	// set up sdl
	err = sdl.Init(uint32(0))
	if err != nil {
		return nil, err
	}

	tv.window, err = sdl.CreateWindow("Gopher2600", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(float32(tv.width*2)*scale), int32(float32(tv.height)*scale), sdl.WINDOW_HIDDEN)
	if err != nil {
		return nil, err
	}
	tv.renderer, err = sdl.CreateRenderer(tv.window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, err
	}
	err = tv.renderer.SetScale(scale, scale)
	if err != nil {
		return nil, err
	}
	tv.texture, err = tv.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, int32(tv.width), int32(tv.height))
	if err != nil {
		return nil, err
	}
	// our screen data
	tv.pixels = make([]byte, tv.width*tv.height*tv.depth)

	// finish up by updating the tv with a black image
	// -- note that we've elected not to show the window on startup
	tv.clear()
	err = tv.update()
	if err != nil {
		return nil, err
	}

	return tv, nil
}

// Signal is principle method of communication between the VCS and televsion
// -- most of the work is done in the embedded HeadlessTV instance
func (tv *SDLTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color) {
	tv.HeadlessTV.Signal(vsync, vblank, frontPorch, hsync, cburst, color)
	r, g, b := tv.decodeColor(color)
	tv.setPixel(int32(tv.horizPos.value+tv.spec.clocksPerHblank), int32(tv.scanline.value), r, g, b, tv.pixels)
}

func (tv SDLTV) decodeColor(color video.Color) (byte, byte, byte) {
	if color == video.NoColor {
		return 0, 0, 0
	}
	// TODO: color decoding
	return 255, 0, 0
}

func (tv *SDLTV) clear() {
	for y := 0; y < tv.height; y++ {
		for x := 0; x < tv.width; x++ {
			tv.setPixel(int32(x), int32(y), 0, 0, 0, tv.pixels)
		}
	}
}

func (tv *SDLTV) setPixel(x, y int32, red, green, blue byte, pixels []byte) {
	i := (y*int32(tv.width) + x) * int32(tv.depth)
	if i < int32(len(pixels)-tv.depth) && i >= 0 {
		pixels[i] = red
		pixels[i+1] = green
		pixels[i+2] = blue
		pixels[i+3] = 255
	}
}

func (tv *SDLTV) update() error {
	err := tv.texture.Update(nil, tv.pixels, int(tv.width*tv.depth))
	if err != nil {
		return err
	}
	err = tv.renderer.Copy(tv.texture, nil, nil)
	if err != nil {
		return err
	}
	tv.renderer.Present()
	return nil
}

// SetVisibility toggles the visiblity of the SDLTV window
func (tv SDLTV) SetVisibility(visible bool) error {
	if visible == true {
		tv.window.Show()
	} else {
		tv.window.Hide()
	}
	return nil
}
