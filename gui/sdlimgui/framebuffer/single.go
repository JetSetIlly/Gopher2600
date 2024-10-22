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

package framebuffer

import (
	"github.com/go-gl/gl/v3.2-core/gl"
)

// Single provides a single framebuffer
type Single struct {
	clearOnRender bool

	texture uint32

	width  int32
	height int32

	fbo uint32
	rbo uint32

	// empty pixels used to clear texture on intiialisation and on clear
	emptyPixels []uint8
}

// NewFlip is the preferred method of initialisation of the Single type
func NewSingle(clearOnRender bool) *Single {
	fb := &Single{
		clearOnRender: clearOnRender,
	}
	gl.GenFramebuffers(1, &fb.fbo)
	return fb
}

// Destroy should be called when the Single is no longer required
func (fb *Single) Destroy() {
	gl.DeleteFramebuffers(1, &fb.fbo)
}

func (fb *Single) Clear() {
	if len(fb.emptyPixels) == 0 {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, fb.texture)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.texture, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, fb.width, fb.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(fb.emptyPixels))
}

// Setup Single for specified dimensions
//
// Returns true if any previous texture data has been lost. This can happen when
// the dimensions have changed. By definition, the first call to Setup() will
// always return false.
//
// If the supplied width or height are less than zero the function will return
// false with no explanation.
func (fb *Single) Setup(width int32, height int32) bool {
	if width <= 0 || height <= 0 {
		return false
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, fb.fbo)

	// no change to framebuffer
	if fb.width == width && fb.height == height {
		return false
	}

	changed := fb.width != 0 || fb.height != 0

	fb.width = width
	fb.height = height
	fb.emptyPixels = make([]uint8, width*height*4)

	gl.GenTextures(1, &fb.texture)
	gl.BindTexture(gl.TEXTURE_2D, fb.texture)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, fb.width, fb.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(fb.emptyPixels))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

	gl.BindRenderbuffer(gl.RENDERBUFFER, fb.rbo)

	return changed
}

// Dimensions returns the width and height of the frame buffer used in the
// Single
func (fb *Single) Dimensions() (width int32, height int32) {
	return fb.width, fb.height
}

// Texture returns the texture ID used by the single framebuffer
func (fb *Single) Texture() uint32 {
	return fb.texture
}

// Process the Single using the suppied draw function. The draw function should
// typically invoke a GLSL shader. The texture ID of the shader will be returned
// by the Process function. This is ID is the same as the ID returned by
// the Texture() function.
func (fb *Single) Process(draw func()) uint32 {
	gl.BindTexture(gl.TEXTURE_2D, fb.texture)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.texture, 0)
	if fb.clearOnRender {
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}
	draw()

	return fb.texture
}
