package sdl

import (
	"gopher2600/debugger/metavideo"

	"github.com/veandco/go-sdl2/sdl"
)

type metaVideoOverlay struct {
	scr *screen

	pixels  []byte
	texture *sdl.Texture

	labels [][]string
}

func newMetaVideoOverlay(scr *screen) (*metaVideoOverlay, error) {
	mpx := new(metaVideoOverlay)
	mpx.scr = scr

	// our acutal screen data
	mpx.pixels = make([]byte, mpx.scr.maxWidth*mpx.scr.maxHeight*scrDepth)

	// labels
	mpx.labels = make([][]string, mpx.scr.maxHeight)
	for i := 0; i < len(mpx.labels); i++ {
		mpx.labels[i] = make([]string, mpx.scr.maxWidth)
	}

	var err error

	mpx.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(mpx.scr.maxWidth), int32(mpx.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	mpx.texture.SetAlphaMod(100)
	mpx.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	return mpx, nil
}

func (mpx *metaVideoOverlay) setPixel(sig metavideo.MetaSignalAttributes) error {
	i := (mpx.scr.lastY*mpx.scr.maxWidth + mpx.scr.lastX) * scrDepth

	if i >= int32(len(mpx.pixels)) {
		return nil
	}

	mpx.pixels[i] = sig.Red
	mpx.pixels[i+1] = sig.Green
	mpx.pixels[i+2] = sig.Blue
	mpx.pixels[i+3] = 255

	// silently allow empty labels
	mpx.labels[mpx.scr.lastY][mpx.scr.lastX] = sig.Label

	return nil
}

func (mpx *metaVideoOverlay) clearPixels() {
	for i := 0; i < len(mpx.pixels); i++ {
		mpx.pixels[i] = 0
	}
}

func (mpx *metaVideoOverlay) update() error {
	err := mpx.texture.Update(nil, mpx.pixels, int(mpx.scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}
	return nil
}

// MetaSignal recieves (and processes) additional emulator information from the emulator
func (gtv *GUI) MetaSignal(sig metavideo.MetaSignalAttributes) error {
	// don't do anything if debugging is not enabled
	if !gtv.allowDebugging {
		return nil
	}

	err := gtv.Television.MetaSignal(sig)
	if err != nil {
		return err
	}

	return gtv.scr.metaPixels.setPixel(sig)
}
