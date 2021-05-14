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

package sdlimgui

import (
	"strings"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shaders"
)

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

// helper function to convert bool to int32
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

// compile and link shader programs
func (sh *shader) createProgram(vertProgram string, fragProgram string) {
	sh.destroy()

	sh.handle = gl.CreateProgram()

	vertHandle := gl.CreateShader(gl.VERTEX_SHADER)
	fragHandle := gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source string) {
		csource, free := gl.Strs(source + "\x00")
		defer free()

		gl.ShaderSource(handle, 1, csource, nil)
	}

	// vertex and fragment glsl source defined in shaders.go (a generated file)
	glShaderSource(vertHandle, vertProgram)
	glShaderSource(fragHandle, fragProgram)

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
	sh := &guiShader{}
	if yflipped {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.ColorShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.ColorShader))
	}
	return sh
}

type effectsShader struct {
	shader

	img *SdlImgui

	screenDim       int32
	numScanlines    int32
	numClocks       int32
	curve           int32
	shadowMask      int32
	scanlines       int32
	noise           int32
	fringing        int32
	curveAmount     int32
	maskBright      int32
	scanlinesBright int32
	noiseLevel      int32
	fringingAmount  int32
	time            int32
}

func newEffectsShader(img *SdlImgui, yflip bool) shaderProgram {
	sh := &effectsShader{
		img: img,
	}
	if yflip {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTEffectsFragShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTEffectsFragShader))
	}

	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.numScanlines = gl.GetUniformLocation(sh.handle, gl.Str("NumScanlines"+"\x00"))
	sh.numClocks = gl.GetUniformLocation(sh.handle, gl.Str("NumClocks"+"\x00"))
	sh.curve = gl.GetUniformLocation(sh.handle, gl.Str("Curve"+"\x00"))
	sh.shadowMask = gl.GetUniformLocation(sh.handle, gl.Str("ShadowMask"+"\x00"))
	sh.scanlines = gl.GetUniformLocation(sh.handle, gl.Str("Scanlines"+"\x00"))
	sh.noise = gl.GetUniformLocation(sh.handle, gl.Str("Noise"+"\x00"))
	sh.fringing = gl.GetUniformLocation(sh.handle, gl.Str("Fringing"+"\x00"))
	sh.curveAmount = gl.GetUniformLocation(sh.handle, gl.Str("CurveAmount"+"\x00"))
	sh.maskBright = gl.GetUniformLocation(sh.handle, gl.Str("MaskBright"+"\x00"))
	sh.scanlinesBright = gl.GetUniformLocation(sh.handle, gl.Str("ScanlinesBright"+"\x00"))
	sh.noiseLevel = gl.GetUniformLocation(sh.handle, gl.Str("NoiseLevel"+"\x00"))
	sh.fringingAmount = gl.GetUniformLocation(sh.handle, gl.Str("FringingAmount"+"\x00"))
	sh.time = gl.GetUniformLocation(sh.handle, gl.Str("Time"+"\x00"))

	return sh
}

// most shader attributes can be discerened automatically but number of
// scanlines, clocks and whether to add noise to the image is context sensitive
func (sh *effectsShader) setAttributesArgs(env shaderEnvironment, numScanlines int, numClocks int, noise bool) {
	sh.shader.setAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1i(sh.numScanlines, int32(numScanlines))
	gl.Uniform1i(sh.numClocks, int32(numClocks))
	gl.Uniform1i(sh.curve, boolToInt32(sh.img.crtPrefs.Curve.Get().(bool)))
	gl.Uniform1i(sh.shadowMask, boolToInt32(sh.img.crtPrefs.Mask.Get().(bool)))
	gl.Uniform1i(sh.scanlines, boolToInt32(sh.img.crtPrefs.Scanlines.Get().(bool)))
	gl.Uniform1i(sh.noise, boolToInt32(noise))
	gl.Uniform1i(sh.fringing, boolToInt32(sh.img.crtPrefs.Fringing.Get().(bool)))
	gl.Uniform1f(sh.curveAmount, float32(sh.img.crtPrefs.CurveAmount.Get().(float64)))
	gl.Uniform1f(sh.maskBright, float32(sh.img.crtPrefs.MaskBright.Get().(float64)))
	gl.Uniform1f(sh.scanlinesBright, float32(sh.img.crtPrefs.ScanlinesBright.Get().(float64)))
	gl.Uniform1f(sh.noiseLevel, float32(sh.img.crtPrefs.NoiseLevel.Get().(float64)))
	gl.Uniform1f(sh.fringingAmount, float32(sh.img.crtPrefs.FringingAmount.Get().(float64)))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
}

type phosphorShader struct {
	shader

	img *SdlImgui

	newFrame          int32
	latency           int32
	correctVideoBlack int32
}

func newPhosphorShader(img *SdlImgui) shaderProgram {
	sh := &phosphorShader{
		img: img,
	}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTPhosphorFragShader))
	sh.newFrame = gl.GetUniformLocation(sh.handle, gl.Str("NewFrame"+"\x00"))
	sh.latency = gl.GetUniformLocation(sh.handle, gl.Str("Latency"+"\x00"))
	sh.correctVideoBlack = gl.GetUniformLocation(sh.handle, gl.Str("CorrectVideoBlack"+"\x00"))
	return sh
}

func (sh *phosphorShader) setAttributesArgs(env shaderEnvironment, latency float32, newFrame uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.latency, latency)

	// video black correction happens when screen is curved
	var correction bool
	if sh.img.isPlaymode() {
		correction = sh.img.crtPrefs.Curve.Get().(bool) && sh.img.crtPrefs.Enabled.Get().(bool)
	} else {
		correction = sh.img.crtPrefs.Curve.Get().(bool) && sh.img.wm.dbgScr.crt
	}
	gl.Uniform1i(sh.correctVideoBlack, boolToInt32(correction))

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, uint32(newFrame))
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
	gl.Uniform2f(sh.blur, blur/float32(env.width), blur/float32(env.height))
}

type blendShader struct {
	shader
	newFrame int32
	modulate int32
	fade     int32
}

func newBlendShader() shaderProgram {
	sh := &blendShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTBlendFragShader))
	sh.newFrame = gl.GetUniformLocation(sh.handle, gl.Str("NewFrame"+"\x00"))
	sh.modulate = gl.GetUniformLocation(sh.handle, gl.Str("Modulate"+"\x00"))
	sh.fade = gl.GetUniformLocation(sh.handle, gl.Str("Fade"+"\x00"))
	return sh
}

func (sh *blendShader) setAttributesArgs(env shaderEnvironment, modulate float32, fade float32, newFrame uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.modulate, modulate)
	gl.Uniform1f(sh.fade, fade)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, uint32(newFrame))
	gl.Uniform1i(sh.newFrame, 1)
}
