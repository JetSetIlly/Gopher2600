// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package sdldebug

import (
	"io"

	"github.com/veandco/go-sdl2/sdl"
)

// overlay type stored and renders additional information for the current tv frame.
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

func (ovl *overlay) destroy(output io.Writer) {
	err := ovl.texture.Destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}
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
