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
	"image"
	"image/jpeg"
	"os"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/logger"
)

// Sequence represents the sequence of textures that can be assigned to a framebuffer.
type Sequence struct {
	textures []uint32
	fbo      uint32
	rbo      uint32
	width    int32
	height   int32

	// empty pixels used to clear texture on intiialisation and during Clear()
	emptyPixels []uint8
}

// NewSequence is the preferred method of initialisation of the Sequence type.
func NewSequence(numTextures int) *Sequence {
	seq := &Sequence{}
	seq.textures = make([]uint32, numTextures)
	gl.GenFramebuffers(1, &seq.fbo)
	return seq
}

// Destroy framebuffer.
func (seq *Sequence) Destroy() {
	gl.DeleteFramebuffers(1, &seq.fbo)
}

// Setup framebuffer for sepecified width and height. Previous texture data is
// lost. Returns true if Setup has caused a change in texture data.
//
// Changes the state of the frame buffer.
func (seq *Sequence) Setup(width int32, height int32) bool {
	gl.BindFramebuffer(gl.FRAMEBUFFER, seq.fbo)

	if seq.width == width && seq.height == height {
		return false
	}

	seq.width = width
	seq.height = height
	seq.emptyPixels = make([]uint8, width*height*4)

	for i := range seq.textures {
		gl.GenTextures(1, &seq.textures[i])
		gl.BindTexture(gl.TEXTURE_2D, seq.textures[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, seq.width, seq.height, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(seq.emptyPixels))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	}

	gl.BindRenderbuffer(gl.RENDERBUFFER, seq.rbo)

	return true
}

// Len returns the number of textures employed in the framebuffer sequence.
func (seq *Sequence) Len() int {
	return len(seq.textures)
}

// Texture returns the texture ID related to the idxTexture.
func (seq *Sequence) Texture(idxTexture int) uint32 {
	return seq.textures[idxTexture]
}

func (seq *Sequence) bind(idxTexture int) uint32 {
	id := seq.textures[idxTexture]
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, id, 0)
	return id
}

// Clear texture. Black pixels.
func (seq *Sequence) Clear(idxTexture int) uint32 {
	id := seq.bind(idxTexture)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, seq.width, seq.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(seq.emptyPixels))
	return id
}

// Process assigns the texture related to idxTexture to the framebuffer and runs
// the supplied draw() function.
//
// Returns the texture ID (not the index) that has been assigned to the framebuffer.
//
// Changes the state of the frame buffer.
func (seq *Sequence) Process(idxTexture int, draw func()) uint32 {
	id := seq.bind(idxTexture)
	draw()
	return id
}

// SavesJPEG writes the texture to the specified path. Does not return any
// errors but will log using logTag if logTag is not empty.
func (seq *Sequence) SaveJPEG(idxTexture int, path string, logTag string) {
	img := image.NewRGBA(image.Rect(0, 0, int(seq.width), int(seq.height)))
	if img == nil {
		if logTag != "" {
			logger.Log(logTag, "save failed: cannot allocate image data")
		}
	}

	seq.bind(idxTexture)
	gl.ReadPixels(0, 0, seq.width, seq.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))

	go func() {
		f, err := os.Create(path)
		if err != nil {
			if logTag != "" {
				logger.Logf(logTag, "save failed: %v", err.Error())
			}
			return
		}

		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
		if err != nil {
			if logTag != "" {
				logger.Logf(logTag, "save failed: %v", err.Error())
			}
			_ = f.Close()
			return
		}

		err = f.Close()
		if err != nil {
			if logTag != "" {
				logger.Logf(logTag, "save failed: %v", err.Error())
			}
			return
		}

		// indicate success
		logger.Logf(logTag, "saved: %s", path)
	}()
}
