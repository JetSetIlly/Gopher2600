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
	"unsafe"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shaders"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type shaderProgram interface {
	createProgram(vertProgram string, fragProgram string)
	destroy()
	setAttributes(shaderEnvironment)
}

type shaderEnvironment struct {
	img *SdlImgui

	// vertex projection
	proj [4][4]float32

	// the function used to trigger the shader program
	draw func()

	// the texture the shader will work with
	srcTextureID uint32

	// width and height of texture. optional depending on the shader
	width  int32
	height int32
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

	gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.proj[0][0])
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	gl.EnableVertexAttribArray(uint32(sh.uv))
	gl.EnableVertexAttribArray(uint32(sh.position))
	gl.EnableVertexAttribArray(uint32(sh.color))

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointer(uint32(sh.uv), 2, gl.FLOAT, false, int32(vertexSize),
		unsafe.Pointer(uintptr(vertexOffsetUv)))
	gl.VertexAttribPointer(uint32(sh.position), 2, gl.FLOAT, false, int32(vertexSize),
		unsafe.Pointer(uintptr(vertexOffsetPos)))
	gl.VertexAttribPointer(uint32(sh.color), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize),
		unsafe.Pointer(uintptr(vertexOffsetCol)))
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
	if log := getShaderCompileError(vertHandle); log != "" {
		panic(log)
	}

	gl.CompileShader(fragHandle)
	if log := getShaderCompileError(fragHandle); log != "" {
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

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

type guiShader struct {
	shader
}

func newGUIShader() shaderProgram {
	sh := &guiShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.GUIShader))
	return sh
}

type colorShader struct {
	shader
}

func newColorShader() shaderProgram {
	sh := &guiShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.ColorShader))
	return sh
}

type dbgScreenShader struct {
	shader

	showCursor         int32 // uniform
	isCropped          int32 // uniform
	screenDim          int32 // uniform
	uncroppedScreenDim int32 // uniform
	scalingX           int32 // uniform
	scalingY           int32 // uniform
	lastX              int32 // uniform
	lastY              int32 // uniform
	hblank             int32 // uniform
	topScanline        int32 // uniform
	botScanline        int32 // uniform
	overlayAlpha       int32 // uniform
}

func newDbgScrShader() shaderProgram {
	sh := &dbgScreenShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.DbgScrShader))

	sh.showCursor = gl.GetUniformLocation(sh.handle, gl.Str("ShowCursor"+"\x00"))
	sh.isCropped = gl.GetUniformLocation(sh.handle, gl.Str("IsCropped"+"\x00"))
	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.uncroppedScreenDim = gl.GetUniformLocation(sh.handle, gl.Str("UncroppedScreenDim"+"\x00"))
	sh.scalingX = gl.GetUniformLocation(sh.handle, gl.Str("ScalingX"+"\x00"))
	sh.scalingY = gl.GetUniformLocation(sh.handle, gl.Str("ScalingY"+"\x00"))
	sh.lastX = gl.GetUniformLocation(sh.handle, gl.Str("LastX"+"\x00"))
	sh.lastY = gl.GetUniformLocation(sh.handle, gl.Str("LastY"+"\x00"))
	sh.hblank = gl.GetUniformLocation(sh.handle, gl.Str("Hblank"+"\x00"))
	sh.topScanline = gl.GetUniformLocation(sh.handle, gl.Str("TopScanline"+"\x00"))
	sh.botScanline = gl.GetUniformLocation(sh.handle, gl.Str("BotScanline"+"\x00"))
	sh.overlayAlpha = gl.GetUniformLocation(sh.handle, gl.Str("OverlayAlpha"+"\x00"))

	return sh
}

func (sh *dbgScreenShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)

	// scaling of screen
	var vertScaling float32
	var horizScaling float32
	if env.img.isPlaymode() {
		vertScaling = env.img.playScr.scaling
		horizScaling = env.img.playScr.horizScaling()
	} else {
		vertScaling = env.img.wm.dbgScr.scaling
		horizScaling = env.img.wm.dbgScr.horizScaling()
	}

	// critical section
	env.img.screen.crit.section.Lock()

	gl.Uniform1f(sh.scalingX, env.img.wm.dbgScr.horizScaling())
	gl.Uniform1f(sh.scalingY, env.img.wm.dbgScr.scaling)
	gl.Uniform2f(sh.uncroppedScreenDim, env.img.wm.dbgScr.scaledWidth(false), env.img.wm.dbgScr.scaledHeight(false))
	gl.Uniform2f(sh.screenDim, env.img.wm.dbgScr.scaledWidth(true), env.img.wm.dbgScr.scaledHeight(true))
	if env.img.wm.dbgScr.cropped {
		gl.Uniform1i(sh.isCropped, 1)
	} else {
		gl.Uniform1i(sh.isCropped, 0)
	}

	cursorX := env.img.screen.crit.lastX
	cursorY := env.img.screen.crit.lastY

	if env.img.wm.dbgScr.cropped {
		gl.Uniform1f(sh.lastX, float32(cursorX-specification.ClksHBlank)*horizScaling)
	} else {
		gl.Uniform1f(sh.lastX, float32(cursorX)*horizScaling)
	}
	gl.Uniform1f(sh.lastY, float32(cursorY)*vertScaling)

	// screen geometry
	gl.Uniform1f(sh.hblank, specification.ClksHBlank*horizScaling)
	gl.Uniform1f(sh.topScanline, float32(env.img.screen.crit.topScanline)*vertScaling)
	gl.Uniform1f(sh.botScanline, float32(env.img.screen.crit.bottomScanline)*vertScaling)

	env.img.screen.crit.section.Unlock()
	// end of critical section

	// show cursor
	if env.img.isRewindSlider {
		gl.Uniform1i(sh.showCursor, 0)
	} else {
		switch env.img.state {
		case gui.StatePaused:
			gl.Uniform1i(sh.showCursor, 1)
		case gui.StateRunning:
			// if FPS is low enough then show screen draw even though
			// emulation is running
			if env.img.lz.TV.ReqFPS < television.ThreshVisual {
				gl.Uniform1i(sh.showCursor, 1)
			} else {
				gl.Uniform1i(sh.showCursor, 0)
			}
		case gui.StateStepping:
			gl.Uniform1i(sh.showCursor, 1)
		case gui.StateRewinding:
			gl.Uniform1i(sh.showCursor, 1)
		}
	}
}

type overlayShader struct {
	shader
	alpha int32 // uniform
}

func newOverlayShader() shaderProgram {
	sh := &overlayShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.OverlayShader))
	sh.alpha = gl.GetUniformLocation(sh.handle, gl.Str("Alpha"+"\x00"))
	return sh
}

func (sh *overlayShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.alpha, env.img.wm.dbgScr.overlayAlpha)
}

type crtShader struct {
	shader

	screenDim int32
	scalingX  int32
	scalingY  int32

	shadowMask          int32
	scanlines           int32
	noise               int32
	blur                int32
	vignette            int32
	flicker             int32
	maskBrightness      int32
	scanlinesBrightness int32
	noiseLevel          int32
	blurLevel           int32
	flickerLevel        int32
	time                int32
}

func newCRTShader() shaderProgram {
	sh := &crtShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTFragShader))

	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.scalingX = gl.GetUniformLocation(sh.handle, gl.Str("ScalingX"+"\x00"))
	sh.scalingY = gl.GetUniformLocation(sh.handle, gl.Str("ScalingY"+"\x00"))
	sh.shadowMask = gl.GetUniformLocation(sh.handle, gl.Str("ShadowMask"+"\x00"))
	sh.scanlines = gl.GetUniformLocation(sh.handle, gl.Str("Scanlines"+"\x00"))
	sh.noise = gl.GetUniformLocation(sh.handle, gl.Str("Noise"+"\x00"))
	sh.blur = gl.GetUniformLocation(sh.handle, gl.Str("Blur"+"\x00"))
	sh.vignette = gl.GetUniformLocation(sh.handle, gl.Str("Vignette"+"\x00"))
	sh.flicker = gl.GetUniformLocation(sh.handle, gl.Str("Flicker"+"\x00"))
	sh.maskBrightness = gl.GetUniformLocation(sh.handle, gl.Str("MaskBrightness"+"\x00"))
	sh.scanlinesBrightness = gl.GetUniformLocation(sh.handle, gl.Str("ScanlinesBrightness"+"\x00"))
	sh.noiseLevel = gl.GetUniformLocation(sh.handle, gl.Str("NoiseLevel"+"\x00"))
	sh.blurLevel = gl.GetUniformLocation(sh.handle, gl.Str("BlurLevel"+"\x00"))
	sh.flickerLevel = gl.GetUniformLocation(sh.handle, gl.Str("FlickerLevel"+"\x00"))
	sh.time = gl.GetUniformLocation(sh.handle, gl.Str("Time"+"\x00"))

	return sh
}

func (sh *crtShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1f(sh.scalingX, env.img.playScr.horizScaling())
	gl.Uniform1f(sh.scalingY, env.img.playScr.scaling)

	gl.Uniform1i(sh.shadowMask, boolToInt32(env.img.crtPrefs.Mask.Get().(bool)))
	gl.Uniform1i(sh.scanlines, boolToInt32(env.img.crtPrefs.Scanlines.Get().(bool)))
	gl.Uniform1i(sh.noise, boolToInt32(env.img.crtPrefs.Noise.Get().(bool)))
	gl.Uniform1i(sh.blur, boolToInt32(env.img.crtPrefs.Blur.Get().(bool)))
	gl.Uniform1i(sh.vignette, boolToInt32(env.img.crtPrefs.Vignette.Get().(bool)))
	gl.Uniform1i(sh.flicker, boolToInt32(env.img.crtPrefs.Flicker.Get().(bool)))
	gl.Uniform1f(sh.maskBrightness, float32(env.img.crtPrefs.MaskBrightness.Get().(float64)))
	gl.Uniform1f(sh.scanlinesBrightness, float32(env.img.crtPrefs.ScanlinesBrightness.Get().(float64)))
	gl.Uniform1f(sh.noiseLevel, float32(env.img.crtPrefs.NoiseLevel.Get().(float64)))
	gl.Uniform1f(sh.blurLevel, float32(env.img.crtPrefs.BlurLevel.Get().(float64)))
	gl.Uniform1f(sh.flickerLevel, float32(env.img.crtPrefs.FlickerLevel.Get().(float64)))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
}

type accumulationShader struct {
	shader
	modulate int32
}

func newAccumulationShader() shaderProgram {
	sh := &accumulationShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTAccumulationFragShader))
	sh.modulate = gl.GetUniformLocation(sh.handle, gl.Str("Modulate"+"\x00"))
	return sh
}

func (sh *accumulationShader) setAttributesAccumation(env shaderEnvironment, modulate float32, textureB uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.modulate, modulate)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, uint32(textureB))
	gl.Uniform1i(sh.texture, 1)
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

func (sh *blurShader) setAttributesBlur(env shaderEnvironment, blur float32) {
	sh.shader.setAttributes(env)
	gl.Uniform2f(sh.blur, blur/float32(env.width), blur/float32(env.height))
}

type blendShader struct {
	shader
	modulate int32
}

func newBlendShader() shaderProgram {
	sh := &blendShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTBlendFragShader))
	sh.modulate = gl.GetUniformLocation(sh.handle, gl.Str("Modulate"+"\x00"))
	return sh
}

func (sh *blendShader) setAttributesBlend(env shaderEnvironment, modulate float32, textureB uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.modulate, modulate)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, uint32(textureB))
	gl.Uniform1i(sh.texture, 1)
}

type playscrShader struct {
	shader

	accumulationShader shaderProgram
	blurShader         shaderProgram
	blendShader        shaderProgram
	crtShader          shaderProgram
	colorShader        shaderProgram

	accumulation uint32
	blur         uint32

	fbo    uint32
	rbo    uint32
	width  int32
	height int32
}

func newPlayscrShader() shaderProgram {
	sh := &playscrShader{}
	sh.accumulationShader = newAccumulationShader()
	sh.blurShader = newBlurShader()
	sh.blendShader = newBlendShader()
	sh.crtShader = newCRTShader()
	sh.colorShader = newColorShader()

	gl.GenFramebuffers(1, &sh.fbo)

	return sh
}

func (sh *playscrShader) setupFrameBuffer(env *shaderEnvironment) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, sh.fbo)

	env.img.screen.crit.section.Lock()
	env.width = int32(env.img.playScr.scaledWidth())
	env.height = int32(env.img.playScr.scaledHeight())
	env.img.screen.crit.section.Unlock()

	if sh.width == env.width && sh.height == env.height {
		return
	}

	sh.width = env.width
	sh.height = env.height

	gl.GenTextures(1, &sh.blur)
	gl.BindTexture(gl.TEXTURE_2D, sh.blur)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, sh.width, sh.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(nil))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	gl.GenTextures(1, &sh.accumulation)
	gl.BindTexture(gl.TEXTURE_2D, sh.accumulation)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, sh.width, sh.height, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(nil))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	gl.GenRenderbuffers(1, &sh.rbo)
	gl.BindRenderbuffer(gl.RENDERBUFFER, sh.rbo)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, sh.width, sh.height)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, sh.rbo)
}

func (sh *playscrShader) setFrameBuffer(texture uint32) {
	gl.BindTexture(gl.TEXTURE_2D, uint32(texture))
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0)
}

func (sh *playscrShader) destroy() {
	sh.crtShader.destroy()
	sh.shader.destroy()
	gl.DeleteFramebuffers(1, &sh.fbo)
}

func (sh *playscrShader) setAttributes(env shaderEnvironment) {
	// return immediately if CRT effects are off
	if !env.img.crtPrefs.Enabled.Get().(bool) {
		sh.colorShader.setAttributes(env)
		return
	}

	// prevserve existing scissor and viewport settings. reverting
	// on defer
	scissor := gl.IsEnabled(gl.SCISSOR_TEST)
	if scissor {
		defer gl.Enable(gl.SCISSOR_TEST)
	}

	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	defer gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])

	// set scissor and viewport
	gl.Disable(gl.SCISSOR_TEST)
	gl.Viewport(int32(-env.img.playScr.imagePosMin.X),
		int32(-env.img.playScr.imagePosMin.Y),
		sh.width+(int32(env.img.playScr.imagePosMin.X*2)),
		sh.height+(int32(env.img.playScr.imagePosMin.Y*2)),
	)

	// make sure our framebuffer is correct
	sh.setupFrameBuffer(&env)

	src := env.srcTextureID

	// add new frame to accumulation buffer
	sh.setFrameBuffer(sh.accumulation)
	sh.accumulationShader.(*accumulationShader).setAttributesAccumation(env, 0.5, sh.accumulation)
	env.draw()

	// blur a small amount for current frame
	sh.setFrameBuffer(sh.blur)
	env.srcTextureID = sh.accumulation
	sh.blurShader.(*blurShader).setAttributesBlur(env, 0.17)
	env.draw()

	// blend blur with original source texture
	sh.setFrameBuffer(sh.blur)
	env.srcTextureID = src
	sh.blendShader.(*blendShader).setAttributesBlend(env, 1.0, sh.blur)
	env.draw()

	// blur a lot for next frame
	sh.setFrameBuffer(sh.accumulation)
	env.srcTextureID = sh.accumulation
	sh.blurShader.(*blurShader).setAttributesBlur(env, 1.0)
	env.draw()

	// crt final shader. copies to real frame buffer
	env.srcTextureID = sh.blur
	sh.crtShader.setAttributes(env)
}
