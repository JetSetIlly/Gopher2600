package sdltv

import (
	"gopher2600/debugger/monitor"

	"github.com/veandco/go-sdl2/sdl"
)

type systemStateOverlay struct {
	scr     *screen
	pixels  []byte
	texture *sdl.Texture
}

func newSystemStateOverlay(scr *screen) (*systemStateOverlay, error) {
	overlay := new(systemStateOverlay)
	overlay.scr = scr

	// our acutal screen data
	overlay.pixels = make([]byte, overlay.scr.maxWidth*overlay.scr.maxHeight*scrDepth)

	var err error

	overlay.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(overlay.scr.maxWidth), int32(overlay.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	overlay.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	return overlay, nil
}

func (overlay *systemStateOverlay) setPixel(attr monitor.SystemState) {
	i := (overlay.scr.lastY*overlay.scr.maxWidth + overlay.scr.lastX) * scrDepth

	if i >= int32(len(overlay.pixels)) {
		return
	}

	// work SystemState information into an overlay

	var r, g, b, a byte

	if attr.Wsync {
		r = 0
		g = 0
		b = 255
		a = 100
	}
	if attr.Hmove {
		r = 0
		g = 255
		b = 0
		a = 100
	}
	if attr.Rsync {
		r = 255
		g = 0
		b = 0
		a = 100
	}

	if a > 0 {
		overlay.pixels[i] = r   // red
		overlay.pixels[i+1] = g // green
		overlay.pixels[i+2] = b // blue
		overlay.pixels[i+3] = a // alpha
	}
}

func (overlay *systemStateOverlay) clearPixels() {
	for i := 0; i < len(overlay.pixels); i++ {
		overlay.pixels[i] = 0
	}
}

func (overlay *systemStateOverlay) update() error {
	err := overlay.texture.Update(nil, overlay.pixels, int(overlay.scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}
	return nil
}

// SystemStateRecord recieves (and processes) additional emulator information from the emulator
func (tv *SDLTV) SystemStateRecord(attr monitor.SystemState) error {
	// don't do anything if debugging is not enabled
	if !tv.allowDebugging {
		return nil
	}

	err := tv.HeadlessTV.SystemStateRecord(attr)
	if err != nil {
		return err
	}

	tv.scr.systemState.setPixel(attr)

	return nil
}
