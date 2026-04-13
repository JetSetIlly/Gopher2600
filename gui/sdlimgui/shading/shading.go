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

package shading

import (
	"strings"

	"github.com/go-gl/gl/v3.2-core/gl"
)

// Program defines the key functions of a Go shader
type Program interface {
	Destroy()
	SetAttributes(Environment)
}

// Environment controls how the texture will be drawn by the shader. For example, the projection
// matrix for the vertex shader.
type Environment struct {
	// the function used to trigger the shader program
	Draw func()

	// projection
	ProjMtx [4][4]float32

	// whether to flip the Y coordinates of the texture when rendering
	FlipY bool

	// the texture the shader will work with
	TextureID uint32

	// width and height of texture. optional depending on the shader
	Width  int32
	Height int32

	// user configuration from texture
	Config any

	// information about the vertex buffer
	vertexSize      int
	vertexOffsetPos int
	vertexOffsetUv  int
	vertexOffsetCol int
}

// SetVertexBufferLayout should be called with a function that returns information about the vertex
// buffer. if using imgui, then the VertexBufferLayout() function is peerfect
func (env *Environment) SetVertexBufferLayout(f func() (int, int, int, int)) {
	env.vertexSize, env.vertexOffsetPos, env.vertexOffsetUv, env.vertexOffsetCol = f()
}

// Base should be embedded by all Go shaders. It implements the Destroy() and
// SetAttributes() function, required by the Program interface.
type Base struct {
	Handle uint32

	// vertex
	projMtx  int32 // uniform
	flipY    int32 // uniform
	position int32
	uv       int32
	color    int32

	// fragment
	texture int32 // uniform
}

// Destroy implements the Program interface
func (sh *Base) Destroy() {
	if sh.Handle != 0 {
		gl.DeleteProgram(sh.Handle)
		sh.Handle = 0
	}
}

// SetAttributes implements the Program interface
func (sh *Base) SetAttributes(env Environment) {
	gl.UseProgram(sh.Handle)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, env.TextureID)
	gl.Uniform1i(sh.texture, 0)

	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.
	gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.ProjMtx[0][0])
	gl.Uniform1i(sh.flipY, BoolToInt32(env.FlipY))

	gl.EnableVertexAttribArray(uint32(sh.uv))
	gl.EnableVertexAttribArray(uint32(sh.position))
	gl.EnableVertexAttribArray(uint32(sh.color))

	gl.VertexAttribPointerWithOffset(uint32(sh.uv), 2, gl.FLOAT, false, int32(env.vertexSize), uintptr(env.vertexOffsetUv))
	gl.VertexAttribPointerWithOffset(uint32(sh.position), 2, gl.FLOAT, false, int32(env.vertexSize), uintptr(env.vertexOffsetPos))
	gl.VertexAttribPointerWithOffset(uint32(sh.color), 4, gl.UNSIGNED_BYTE, true, int32(env.vertexSize), uintptr(env.vertexOffsetCol))
}

// version string to attach to all shaders
const glslVersion = "#version 150\n"

// CreateProgram compiles a links vertex and fragment shaders
func (sh *Base) CreateProgram(vertProgram string, fragProgram ...string) {
	sh.Destroy()

	sh.Handle = gl.CreateProgram()

	vertHandle := gl.CreateShader(gl.VERTEX_SHADER)
	fragHandle := gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source ...string) {
		b := strings.Builder{}
		b.WriteString(glslVersion)
		for _, s := range source {
			b.WriteString(s)
		}
		b.WriteRune('\x00')

		src, free := gl.Strs(b.String())
		defer free()

		gl.ShaderSource(handle, 1, src, nil)
	}

	// vertex and fragment glsl source defined in shaders.go (a generated file)
	glShaderSource(vertHandle, vertProgram)
	glShaderSource(fragHandle, fragProgram...)

	gl.CompileShader(vertHandle)
	if log := getShaderCompileError(vertHandle); log != "" {
		panic(log)
	}

	gl.CompileShader(fragHandle)
	if log := getShaderCompileError(fragHandle); log != "" {
		panic(log)
	}

	gl.AttachShader(sh.Handle, vertHandle)
	gl.AttachShader(sh.Handle, fragHandle)
	gl.LinkProgram(sh.Handle)

	// now that the shader promer has linked we no longer need the individual
	// shader programs
	gl.DeleteShader(fragHandle)
	gl.DeleteShader(vertHandle)

	// get references to shader attributes and uniforms variables
	sh.projMtx = sh.GetUniformLocation("ProjMtx")
	sh.flipY = sh.GetUniformLocation("FlipY")
	sh.position = sh.GetAttribLocation("Position")
	sh.uv = sh.GetAttribLocation("UV")
	sh.color = sh.GetAttribLocation("Color")
	sh.texture = sh.GetUniformLocation("Texture")
}

func (sh *Base) GetUniformLocation(name string) int32 {
	return gl.GetUniformLocation(sh.Handle, gl.Str(name+"\x00"))
}

func (sh *Base) GetAttribLocation(name string) int32 {
	return gl.GetAttribLocation(sh.Handle, gl.Str(name+"\x00"))
}

// getShaderCompileError returns the most recent error generated by the shader compiler
func getShaderCompileError(shader uint32) string {
	var isCompiled int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &isCompiled)
	if isCompiled == 0 {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		if logLength > 0 {
			// The maxLength includes the NULL character
			log := strings.Repeat("\x00", int(logLength+1))
			gl.GetShaderInfoLog(shader, logLength, &logLength, gl.Str(log))
			return log
		}
	}
	return ""
}

// BoolToInt32 converts a boolean value into either 1 or 0 (of type int32)
func BoolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}
