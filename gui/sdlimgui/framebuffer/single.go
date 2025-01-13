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

	texture texture

	width  int32
	height int32

	fbo uint32
	rbo uint32

	// empty pixels used to clear texture on create()
	//
	// the length of the array is based on the dimensions of the texture. to
	// avoid excessive reallocation the length of the array never reduces and we
	// simply take the slice we require if the texture is smaller
	emptyPixels []uint8
}

// NewFlip is the preferred method of initialisation of the Single type
func NewSingle(clearOnRender bool) *Single {
	fb := &Single{
		clearOnRender: clearOnRender,
	}
	gl.GenFramebuffers(1, &fb.fbo)
	gl.GenTextures(1, &fb.texture.id)
	return fb
}

// Destroy should be called when the Single is no longer required
func (fb *Single) Destroy() {
	gl.DeleteFramebuffers(1, &fb.fbo)
}

func (fb *Single) Clear() {
	gl.BindTexture(gl.TEXTURE_2D, fb.texture.id)
	gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// Setup Single for specified dimensions
//
// Returns true if any previous texture data has been lost. This can happen when
// the dimensions have changed. By definition, the first call to Setup() will
// always return false.
func (fb *Single) Setup(width int32, height int32) bool {
	if width <= 0 || height <= 0 {
		return false
	}

	// no change to framebuffer
	if fb.width == width && fb.height == height {
		return false
	}

	fb.width = width
	fb.height = height

	// make sure emptyPixels is big enough
	sz := int(fb.width * fb.height * 4)
	if sz > cap(fb.emptyPixels) {
		fb.emptyPixels = make([]uint8, sz, sz*2)
	}

	// mark texture for creation
	fb.texture.create = true

	return true
}

// Dimensions returns the width and height of the frame buffer used in the
// Single
func (fb *Single) Dimensions() (width int32, height int32) {
	return fb.width, fb.height
}

// TextureID returns the texture ID used by the single framebuffer
func (fb *Single) TextureID() uint32 {
	return fb.texture.id
}

func (fb *Single) create() {
	fb.texture.create = false
	gl.BindTexture(gl.TEXTURE_2D, fb.texture.id)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, fb.width, fb.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(fb.emptyPixels[:fb.width*fb.height*4]))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
}

// Process the Single using the suppied draw function. The draw function should
// typically invoke a GLSL shader. The texture ID of the shader will be returned
// by the Process function. This is ID is the same as the ID returned by
// the Texture() function.
func (fb *Single) Process(draw func()) uint32 {
	if fb.texture.create {
		fb.create()
		fb.texture.create = false
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.texture.id, 0)
	if fb.clearOnRender {
		gl.ClearColor(0.0, 0.0, 0.0, 0.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}
	draw()
	return fb.texture.id
}

// bindForCopy implements the FBO interface
func (fb *Single) bindForCopy() {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.texture.id, 0)
}

// Copy another framebuffer to the Single instance. Framebuffers must be of the
// same dimensions
func (fb *Single) Copy(src FBO) uint32 {
	if fb.texture.create {
		fb.create()
		fb.texture.create = false
	}

	src.bindForCopy()

	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.DRAW_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.texture.id, 0)
	gl.BlitFramebuffer(0, 0, fb.width, fb.height,
		0, 0, fb.width, fb.height,
		gl.COLOR_BUFFER_BIT, gl.NEAREST)
	return fb.texture.id
}
