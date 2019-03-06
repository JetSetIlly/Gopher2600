package sdltv

import (
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// with a bit of work this metaSignalOverlay struct can be be adapted to
// effectively display many different types of meta signal

type metasignalOverlay struct {
	scr     *screen
	pixels  []byte
	texture *sdl.Texture
}

func newMetasignalOverlay(scr *screen) (*metasignalOverlay, error) {
	overlay := new(metasignalOverlay)
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

func (overlay *metasignalOverlay) setPixel(attr television.MetaSignalAttributes) {
	i := (overlay.scr.lastY*overlay.scr.maxWidth + overlay.scr.lastX) * scrDepth

	if i >= int32(len(overlay.pixels)) {
		return
	}

	// work meta-signal information into an overlay

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

func (overlay *metasignalOverlay) clearPixels() {
	for i := 0; i < len(overlay.pixels); i++ {
		overlay.pixels[i] = 0
	}
}

func (overlay *metasignalOverlay) update() error {
	err := overlay.texture.Update(nil, overlay.pixels, int(overlay.scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}
	return nil
}

// MetaSignal recieves (and processes) additional emulator information from the emulator
func (tv *SDLTV) MetaSignal(attr television.MetaSignalAttributes) error {
	// don't do anything if debugging is not enabled
	if !tv.allowDebugging {
		return nil
	}

	err := tv.HeadlessTV.MetaSignal(attr)
	if err != nil {
		return err
	}

	tv.scr.metasignals.setPixel(attr)

	return nil
}
