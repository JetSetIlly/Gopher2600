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

//go:build !gl21

package sdlimgui

import (
	"strings"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shaders"
)

// version string to attach to all shaders
const fragVersion = "#version 150\n\n"

type shaderProgram interface {
	destroy()
	setAttributes(shaderEnvironment)
}

type shaderEnvironment struct {
	// the function used to trigger the shader program
	draw func()

	// vertex projection
	presentationProj [4][4]float32

	// projection to use for texture-to-texture processing
	internalProj [4][4]float32

	// whether to use the internalProj matrix
	useInternalProj bool

	// the texture the shader will work with
	srcTextureID uint32

	// width and height of texture. optional depending on the shader
	width  int32
	height int32
}

// helper function to convert bool to int32.
func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

type shader struct {
	handle uint32

	// vertex
	projMtx  int32 // uniform
	position int32
	uv       int32
	color    int32

	// fragment
	texture int32 // uniform
}

func (sh *shader) destroy() {
	if sh.handle != 0 {
		gl.DeleteProgram(sh.handle)
		sh.handle = 0
	}
}

func (sh *shader) setAttributes(env shaderEnvironment) {
	gl.UseProgram(sh.handle)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, env.srcTextureID)
	gl.Uniform1i(sh.texture, 0)

	if env.useInternalProj {
		gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.internalProj[0][0])
	} else {
		gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.presentationProj[0][0])
	}
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	gl.EnableVertexAttribArray(uint32(sh.uv))
	gl.EnableVertexAttribArray(uint32(sh.position))
	gl.EnableVertexAttribArray(uint32(sh.color))

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointerWithOffset(uint32(sh.uv), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetUv))
	gl.VertexAttribPointerWithOffset(uint32(sh.position), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetPos))
	gl.VertexAttribPointerWithOffset(uint32(sh.color), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), uintptr(vertexOffsetCol))
}

// compile and link shader programs.
func (sh *shader) createProgram(vertProgram string, fragProgram ...string) {
	sh.destroy()

	sh.handle = gl.CreateProgram()

	vertHandle := gl.CreateShader(gl.VERTEX_SHADER)
	fragHandle := gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source ...string) {
		b := strings.Builder{}
		b.WriteString(fragVersion)
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
	if log := sh.getShaderCompileError(vertHandle); log != "" {
		panic(log)
	}

	gl.CompileShader(fragHandle)
	if log := sh.getShaderCompileError(fragHandle); log != "" {
		panic(log)
	}

	gl.AttachShader(sh.handle, vertHandle)
	gl.AttachShader(sh.handle, fragHandle)
	gl.LinkProgram(sh.handle)

	// now that the shader promer has linked we no longer need the individual
	// shader programs
	gl.DeleteShader(fragHandle)
	gl.DeleteShader(vertHandle)

	// get references to shader attributes and uniforms variables
	sh.projMtx = gl.GetUniformLocation(sh.handle, gl.Str("ProjMtx"+"\x00"))
	sh.position = gl.GetAttribLocation(sh.handle, gl.Str("Position"+"\x00"))
	sh.uv = gl.GetAttribLocation(sh.handle, gl.Str("UV"+"\x00"))
	sh.color = gl.GetAttribLocation(sh.handle, gl.Str("Color"+"\x00"))
	sh.texture = gl.GetUniformLocation(sh.handle, gl.Str("Texture"+"\x00"))
}

// getShaderCompileError returns the most recent error generated
// by the shader compiler.
func (sh *shader) getShaderCompileError(shader uint32) string {
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

type colorShader struct {
	shader
}

func newColorShader(yflipped bool) shaderProgram {
	sh := &colorShader{}
	if yflipped {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.ColorShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.ColorShader))
	}
	return sh
}

type blackCorrectionShader struct {
	shader
	blackLevel int32
}

func newBlackCorrectionShader() shaderProgram {
	sh := &blackCorrectionShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTBlackCorrection))
	sh.blackLevel = gl.GetUniformLocation(sh.handle, gl.Str("BlackLevel"+"\x00"))
	return sh
}

func (sh *blackCorrectionShader) setAttributesArgs(env shaderEnvironment, blackLevel float32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.blackLevel, blackLevel)
}

type phosphorShader struct {
	shader

	newFrame int32
	latency  int32
}

func newPhosphorShader() shaderProgram {
	sh := &phosphorShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTPhosphorFragShader))
	sh.newFrame = gl.GetUniformLocation(sh.handle, gl.Str("NewFrame"+"\x00"))
	sh.latency = gl.GetUniformLocation(sh.handle, gl.Str("Latency"+"\x00"))
	return sh
}

func (sh *phosphorShader) setAttributesArgs(env shaderEnvironment, latency float32, newFrame uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.latency, latency)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, newFrame)
	gl.Uniform1i(sh.newFrame, 1)
}

type blurShader struct {
	shader
	blur int32
}

func newBlurShader() shaderProgram {
	sh := &blurShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTBlurFragShader))
	sh.blur = gl.GetUniformLocation(sh.handle, gl.Str("Blur"+"\x00"))
	return sh
}

func (sh *blurShader) setAttributesArgs(env shaderEnvironment, blur float32) {
	sh.shader.setAttributes(env)

	// normalise blur amount depending on screen size
	blur *= float32(env.height) / 960.0

	gl.Uniform2f(sh.blur, blur/float32(env.width), blur/float32(env.height))
}

type ghostingShader struct {
	shader
	img       *SdlImgui
	screenDim int32
	amount    int32
}

func newGhostingShader() shaderProgram {
	sh := &ghostingShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTGhostingFragShader))
	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.amount = gl.GetUniformLocation(sh.handle, gl.Str("Amount"+"\x00"))
	return sh
}

func (sh *ghostingShader) setAttributesArgs(env shaderEnvironment, amount float32) {
	sh.shader.setAttributes(env)
	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1f(sh.amount, amount)
}

type sharpenShader struct {
	shader
	texture   int32
	sharpness int32
}

func newSharpenShader(yflipped bool) shaderProgram {
	sh := &sharpenShader{}
	if yflipped {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.SharpenShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.SharpenShader))
	}
	sh.texture = gl.GetUniformLocation(sh.handle, gl.Str("Texture"+"\x00"))
	sh.sharpness = gl.GetUniformLocation(sh.handle, gl.Str("Sharpness"+"\x00"))
	return sh
}

// textureSpec is used by the sharpen shader
type textureSpec interface {
	// returns texture ID and the width and height of the texture
	textureSpec() (uint32, float32, float32)
}

// nolint: unparam
func (sh *sharpenShader) setAttributesArgs(env shaderEnvironment, scalingImage textureSpec, sharpness int) {
	t, _, _ := scalingImage.textureSpec()
	sh.shader.setAttributes(env)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, t)
	gl.Uniform1i(sh.texture, 1)
	gl.Uniform1i(sh.sharpness, int32(sharpness))
}

type guiShader struct {
	shader
}

func newGUIShader() shaderProgram {
	sh := &guiShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.GUIShader))
	return sh
}
