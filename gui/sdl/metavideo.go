package sdl

import (
	"gopher2600/gui/metavideo"

	"github.com/veandco/go-sdl2/sdl"
)

type metaVideoOverlay struct {
	scr *screen

	texture     *sdl.Texture
	textureFade *sdl.Texture

	pixels     []byte
	pixelsFade []byte

	labels [][]string
}

func newMetaVideoOverlay(scr *screen) (*metaVideoOverlay, error) {
	mv := new(metaVideoOverlay)
	mv.scr = scr

	// our acutal screen data
	mv.pixels = make([]byte, mv.scr.maxWidth*mv.scr.maxHeight*scrDepth)
	mv.pixelsFade = make([]byte, mv.scr.maxWidth*mv.scr.maxHeight*scrDepth)

	// labels
	mv.labels = make([][]string, mv.scr.maxHeight)
	for i := 0; i < len(mv.labels); i++ {
		mv.labels[i] = make([]string, mv.scr.maxWidth)
	}

	var err error

	mv.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(mv.scr.maxWidth), int32(mv.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	mv.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	mv.texture.SetAlphaMod(100)

	mv.textureFade, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(mv.scr.maxWidth), int32(mv.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	mv.textureFade.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	mv.textureFade.SetAlphaMod(50)

	return mv, nil
}

func (mv *metaVideoOverlay) setPixel(sig metavideo.MetaSignalAttributes) error {
	i := (mv.scr.lastY*mv.scr.maxWidth + mv.scr.lastX) * scrDepth

	if i >= int32(len(mv.pixels)) {
		return nil
	}

	mv.pixels[i] = sig.Red
	mv.pixels[i+1] = sig.Green
	mv.pixels[i+2] = sig.Blue
	mv.pixels[i+3] = sig.Alpha

	// silently allow empty labels
	mv.labels[mv.scr.lastY][mv.scr.lastX] = sig.Label

	return nil
}

func (mv *metaVideoOverlay) newFrame() {
	// swap pixel array with pixelsFade array
	// -- see comment in sdl.screen.newFrame() function for why we do this
	swp := mv.pixels
	mv.pixels = mv.pixelsFade
	mv.pixelsFade = swp

	// clear regular pixels
	for i := 0; i < len(mv.pixels); i++ {
		mv.pixels[i] = 0
	}
}

func (mv *metaVideoOverlay) update(paused bool) error {
	if paused {
		err := mv.textureFade.Update(nil, mv.pixelsFade, int(mv.scr.maxWidth*scrDepth))
		if err != nil {
			return err
		}

		err = mv.scr.renderer.Copy(mv.textureFade, mv.scr.srcRect, mv.scr.destRect)
		if err != nil {
			return err
		}
	}

	err := mv.texture.Update(nil, mv.pixels, int(mv.scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}

	err = mv.scr.renderer.Copy(mv.texture, mv.scr.srcRect, mv.scr.destRect)
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

	return gtv.scr.metaVideo.setPixel(sig)
}
