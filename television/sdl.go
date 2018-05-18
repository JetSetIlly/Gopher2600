package television

import (
	"fmt"
	"gopher2600/hardware/tia/video"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTV is the SDL implementation of a simple television
type SDLTV struct {
	HeadlessTV

	width      int32
	height     int32
	pixelDepth int32

	pixelWidth int

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

	tv.window, err = sdl.CreateWindow("Gopher2600", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(float32(tv.width)*scale*float32(tv.pixelWidth)), int32(float32(tv.height)*scale), sdl.WINDOW_HIDDEN)
	if err != nil {
		return nil, err
	}

	tv.renderer, err = sdl.CreateRenderer(tv.window, -1, sdl.RENDERER_ACCELERATED)
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

	tv.texture, err = tv.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, tv.width, tv.height)
	if err != nil {
		return nil, err
	}

	// our acutal screen data
	tv.pixels = make([]byte, tv.width*tv.height*tv.pixelDepth)

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
// -- note that most of the work is done in the embedded HeadlessTV instance
func (tv *SDLTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color) {
	tv.HeadlessTV.Signal(vsync, vblank, frontPorch, hsync, cburst, color)
	r, g, b := tv.decodeColor(color)
	tv.setPixel(int32(tv.pixelX()), int32(tv.pixelY()), r, g, b, tv.pixels)
}

func (tv SDLTV) decodeColor(color video.Color) (byte, byte, byte) {
	if color == video.NoColor {
		return 0, 0, 0
	}
	// TODO: color decoding
	return 255, 255, 0
}

func (tv *SDLTV) clear() {
	for y := int32(0); y < tv.height; y++ {
		for x := int32(0); x < tv.width; x++ {
			tv.setPixel(x, y, 0, 0, 0, tv.pixels)
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
	err := tv.texture.Update(nil, tv.pixels, int(tv.width*tv.pixelDepth))
	if err != nil {
		return err
	}
	err = tv.renderer.Copy(tv.texture, nil, nil)
	if err != nil {
		return err
	}

	// single pixel marker overlay
	tv.renderer.SetDrawColor(40, 40, 0, 10)
	for i := 0; i < int(tv.width); i += 2 {
		tv.renderer.DrawRect(&sdl.Rect{int32(i), 0, 1, int32(tv.spec.scanlinesTotal)})
	}
	for i := 0; i < int(tv.height); i += 2 {
		tv.renderer.DrawRect(&sdl.Rect{0, int32(i), int32(tv.spec.clocksPerScanline), 1})
	}

	// tens-pixel marker overlay
	tv.renderer.SetDrawColor(40, 40, 0, 25)
	for i := 0; i < int(tv.width); i += 20 {
		tv.renderer.DrawRect(&sdl.Rect{int32(i), int32(tv.spec.scanlinesPerVBlank) - 1, 10, 1})
	}
	for i := 0; i < int(tv.height); i += 20 {
		tv.renderer.DrawRect(&sdl.Rect{int32(tv.spec.clocksPerHblank) - 1, int32(i), 1, 10})
	}

	// add screen boundary overlay
	tv.renderer.SetDrawColor(100, 100, 100, 25)
	tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	tv.renderer.FillRect(&sdl.Rect{0, 0, int32(tv.spec.clocksPerHblank), int32(tv.spec.scanlinesTotal)})
	tv.renderer.FillRect(&sdl.Rect{0, 0, int32(tv.spec.clocksPerScanline), int32(tv.spec.scanlinesPerVBlank)})
	tv.renderer.FillRect(&sdl.Rect{0, int32(tv.spec.scanlinesTotal - tv.spec.scanlinesPerOverscan), int32(tv.spec.clocksPerScanline), int32(tv.spec.scanlinesPerOverscan)})

	// add cursor overlay
	tv.renderer.SetDrawColor(255, 255, 255, 100)
	cursorX := tv.pixelX()
	cursorY := tv.pixelY()
	if cursorX >= tv.spec.clocksPerScanline+tv.spec.clocksPerHblank {
		cursorX = 0
		cursorY++
	}
	tv.renderer.SetDrawBlendMode(sdl.BLENDMODE_ADD)
	tv.renderer.DrawRect(&sdl.Rect{int32(cursorX), int32(cursorY), 2, 2})

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
