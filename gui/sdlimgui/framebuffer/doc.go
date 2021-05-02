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

// Package framebuffer provides a convenient way of working with OpenGL
// framebuffers. The Sequence type conceptualises a sequence of textures all of
// which may be attached to the framebuffer object for a drawing operation.
//
// The key to the Sequence type is the texture index. This is not to be
// confused with the texture ID. The number of textures (and therefore texture
// indices) is defined at Seqeunce creation, with NewSequence().
//
// For example, to create a framebuffer sequence with two textures:
//
//		seq := NewSequence(2)
//
// The Setup() function must be called at least once after NewSequence() and
// called as often as necessary to ensure the dimensions (width and height) are
// correct.
//
//		hasChanged := seq.Setup(800, 600)
//
// Setup() returns true if the texture data has been recreated in accordance
// with the new dimensions.
//
// The Process() function is used to assign the framebuffer object and for
// convenience, runs the supplied the draw() function. The texture ID is
// returned and can be used for presentation of as the input for the next call
// to Process() (via the draw() function).
//
//		texture := seq.Process(0, func() {
//			// 1. set up shader
//			// 2. OpenGL draw (eg. gl.DrawElements()
//		})
//
// Note that much of the work of chaning a sequence of shaders must be
// performed by the user of the package. The package does however, hide a lot
// of detail behind the Process() and Setup() functions.
//
package framebuffer
