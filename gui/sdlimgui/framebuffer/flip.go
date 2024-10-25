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

// Flip provides a two paged framebuffer
type Flip struct {
	clearOnRender bool

	flip    [2]uint32
	flipIdx int

	width  int32
	height int32

	fbo uint32
	rbo uint32

	// empty pixels used to clear texture on intiialisation and on clear
	emptyPixels []uint8
}

// NewFlip is the preferred method of initialisation of the Flip type
func NewFlip(clearOnRender bool) *Flip {
	fb := &Flip{
		clearOnRender: clearOnRender,
	}
	gl.GenFramebuffers(1, &fb.fbo)
	return fb
}

// id implements the FBO interface
func (fb *Flip) id() uint32 {
	return fb.fbo
}

// Destroy should be called when the Flip is no longer required
func (fb *Flip) Destroy() {
	gl.DeleteFramebuffers(1, &fb.fbo)
}

func (fb *Flip) Clear() {
	if len(fb.emptyPixels) == 0 {
		return
	}
	for _, id := range fb.flip {
		gl.BindTexture(gl.TEXTURE_2D, id)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, fb.width, fb.height, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(fb.emptyPixels))
	}
}

// Setup Flip for specified dimensions
//
// Returns true if any previous texture data has been lost. This can happen when
// the dimensions have changed. By definition, the first call to Setup() will
// always return false.
//
// If the supplied width or height are less than zero the function will return
// false with no explanation.
func (fb *Flip) Setup(width int32, height int32) bool {
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

	for i := range fb.flip {
		gl.GenTextures(1, &fb.flip[i])
		gl.BindTexture(gl.TEXTURE_2D, fb.flip[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, fb.width, fb.height, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(fb.emptyPixels))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	}

	gl.BindRenderbuffer(gl.RENDERBUFFER, fb.rbo)

	return changed
}

// Dimensions returns the width and height of the frame buffer used in the Flip
func (fb *Flip) Dimensions() (width int32, height int32) {
	return fb.width, fb.height
}

// Texture returns the texture ID of the last Flip texture to be processed.
// Using this ID can be an effective way of chaining shaders
func (fb *Flip) Texture() uint32 {
	return fb.flip[fb.flipIdx]
}

// Process the Flip using the suppied draw function. The draw function should
// typically invoke a GLSL shader. The texture ID of the shader will be returned
// by the Process function. This is ID is the same as the ID returned by
// the Texture() function
func (fb *Flip) Process(draw func()) uint32 {
	fb.flipIdx++
	if fb.flipIdx >= len(fb.flip) {
		fb.flipIdx = 0
	}

	id := fb.flip[fb.flipIdx]
	gl.BindFramebuffer(gl.FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, id, 0)

	if fb.clearOnRender {
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}

	draw()

	return id
}

// bindForCopy implements the FBO interface
func (fb *Flip) bindForCopy() {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.flip[fb.flipIdx], 0)
}

// Copy another framebuffer to the Flip instance. Framebuffers must be of the
// same dimensions
func (fb *Flip) Copy(src FBO) uint32 {
	src.bindForCopy()
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, fb.fbo)
	gl.FramebufferTexture2D(gl.DRAW_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.flip[fb.flipIdx], 0)
	gl.BlitFramebuffer(0, 0, fb.width, fb.height,
		0, 0, fb.width, fb.height,
		gl.COLOR_BUFFER_BIT, gl.NEAREST)
	return fb.flip[fb.flipIdx]
}
