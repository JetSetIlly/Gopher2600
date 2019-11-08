package sdldebug

import (
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

type metapixelOverlay struct {
	scr *pixels

	texture     *sdl.Texture
	textureFade *sdl.Texture

	pixels     []byte
	pixelsFade []byte

	labels [][]string
}

func newMetapixelOverlay(scr *pixels) (*metapixelOverlay, error) {
	ovl := new(metapixelOverlay)
	ovl.scr = scr

	// our acutal screen data
	ovl.pixels = make([]byte, ovl.scr.maxWidth*ovl.scr.maxHeight*pixelDepth)
	ovl.pixelsFade = make([]byte, ovl.scr.maxWidth*ovl.scr.maxHeight*pixelDepth)

	// labels
	ovl.labels = make([][]string, ovl.scr.maxHeight)
	for i := 0; i < len(ovl.labels); i++ {
		ovl.labels[i] = make([]string, ovl.scr.maxWidth)
	}

	var err error

	ovl.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(ovl.scr.maxWidth), int32(ovl.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	ovl.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	ovl.texture.SetAlphaMod(100)

	ovl.textureFade, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(ovl.scr.maxWidth), int32(ovl.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	ovl.textureFade.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))
	ovl.textureFade.SetAlphaMod(50)

	return ovl, nil
}

func (ovl *metapixelOverlay) setPixel(sig gui.MetaPixel) error {
	i := (ovl.scr.lastY*ovl.scr.maxWidth + ovl.scr.lastX) * pixelDepth

	if i >= int32(len(ovl.pixels)) {
		return nil
	}

	ovl.pixels[i] = sig.Red
	ovl.pixels[i+1] = sig.Green
	ovl.pixels[i+2] = sig.Blue
	ovl.pixels[i+3] = sig.Alpha

	// silently allow empty labels
	ovl.labels[ovl.scr.lastY][ovl.scr.lastX] = sig.Label

	return nil
}

func (ovl *metapixelOverlay) newFrame() {
	// swap pixel array with pixelsFade array
	// -- see comment in sdl.screen.newFrame() function for why we do this
	swp := ovl.pixels
	ovl.pixels = ovl.pixelsFade
	ovl.pixelsFade = swp

	// clear regular pixels
	for i := 0; i < len(ovl.pixels); i++ {
		ovl.pixels[i] = 0
	}
}

func (ovl *metapixelOverlay) update(paused bool) error {
	if paused {
		err := ovl.textureFade.Update(nil, ovl.pixelsFade, int(ovl.scr.maxWidth*pixelDepth))
		if err != nil {
			return err
		}

		err = ovl.scr.renderer.Copy(ovl.textureFade, ovl.scr.srcRect, ovl.scr.destRect)
		if err != nil {
			return err
		}
	}

	err := ovl.texture.Update(nil, ovl.pixels, int(ovl.scr.maxWidth*pixelDepth))
	if err != nil {
		return err
	}

	err = ovl.scr.renderer.Copy(ovl.texture, ovl.scr.srcRect, ovl.scr.destRect)
	if err != nil {
		return err
	}

	return nil
}

// SetMetaPixel recieves (and processes) additional emulator information from the emulator
func (pxtv *SdlDebug) SetMetaPixel(sig gui.MetaPixel) error {
	return pxtv.pxl.metaPixels.setPixel(sig)
}
