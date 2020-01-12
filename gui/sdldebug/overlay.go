package sdldebug

import (
	"github.com/veandco/go-sdl2/sdl"
)

// the overlay type stored and renders the meta-pixels for the current tv frame
type overlay struct {
	renderer *sdl.Renderer
	pixels   []byte
	clr      []byte
	texture  *sdl.Texture
}

// create a new instance of the overlay type. called everytime the screen
// dimensions change.
func newOverlay(renderer *sdl.Renderer, w, h int) (*overlay, error) {
	l := w * h * pixelDepth

	ovl := &overlay{
		renderer: renderer,
		pixels:   make([]byte, l),
		clr:      make([]byte, l),
	}

	for i := pixelDepth - 1; i < l; i += pixelDepth {
		ovl.pixels[i] = 255
	}

	var err error

	ovl.texture, err = renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888),
		int(sdl.TEXTUREACCESS_STREAMING),
		int32(w), int32(h))
	if err != nil {
		return nil, err
	}

	err = ovl.texture.SetBlendMode(sdl.BLENDMODE_BLEND)
	if err != nil {
		return nil, err
	}

	err = ovl.texture.SetAlphaMod(200)
	if err != nil {
		return nil, err
	}

	return ovl, nil
}

func (ovl *overlay) render(cpyRect *sdl.Rect, pitch int) error {
	// update texture
	err := ovl.texture.Update(nil, ovl.pixels, pitch)
	if err != nil {
		return err
	}

	// draw texture to renderer
	err = ovl.renderer.Copy(ovl.texture, cpyRect, nil)
	if err != nil {
		return err
	}

	return nil
}

func (ovl *overlay) length() int {
	return len(ovl.pixels)
}

func (ovl *overlay) clear() {
	copy(ovl.pixels, ovl.clr)
}
