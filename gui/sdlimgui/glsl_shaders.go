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
	destroy()
	setAttributes(shaderEnvironment)
}

type shaderEnvironment struct {
	img *SdlImgui

	// the function used to trigger the shader program
	draw func()

	// vertex projection
	presentationProj [4][4]float32

	// projection to use for texture-to-texture processing
	internalProj [4][4]float32

	isInternalShader bool

	// the texture the shader will work with
	srcTextureID uint32

	// width and height of texture. optional depending on the shader
	width  int32
	height int32
}

type framebuffer struct {
	textures []uint32
	fbo      uint32
	rbo      uint32
	width    int32
	height   int32

	// the texture of the most recent texture plugged into the framebuffer.
	currentID uint32
}

func newFramebuffer(numTextures int) *framebuffer {
	fb := &framebuffer{}
	fb.textures = make([]uint32, numTextures)
	gl.GenFramebuffers(1, &fb.fbo)
	return fb
}

func (fb *framebuffer) destroy() {
	gl.DeleteFramebuffers(1, &fb.fbo)
}

func (fb *framebuffer) setup(width int32, height int32) bool {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fb.fbo)

	if fb.width == width && fb.height == height {
		return false
	}

	fb.width = width
	fb.height = height

	for i := range fb.textures {
		gl.GenTextures(1, &fb.textures[i])
		gl.BindTexture(gl.TEXTURE_2D, fb.textures[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, fb.width, fb.height, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(nil))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	}

	gl.BindRenderbuffer(gl.RENDERBUFFER, fb.rbo)

	return true
}

// clear texture
func (fb *framebuffer) clear(bufferIdx int) {
	gl.ClearTexImage(fb.textures[bufferIdx], 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(nil))
}

// returns the texture ID that has been assigned to the framebuffer.
func (fb *framebuffer) draw(bufferIdx int, draw func()) uint32 {
	fb.currentID = fb.textures[bufferIdx]
	gl.BindTexture(gl.TEXTURE_2D, fb.currentID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.currentID, 0)
	draw()
	return fb.currentID
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

	if env.isInternalShader {
		gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.internalProj[0][0])
	} else {
		gl.UniformMatrix4fv(sh.projMtx, 1, false, &env.presentationProj[0][0])
	}
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	gl.EnableVertexAttribArray(uint32(sh.uv))
	gl.EnableVertexAttribArray(uint32(sh.position))
	gl.EnableVertexAttribArray(uint32(sh.color))

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointer(uint32(sh.uv), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetUv)))
	gl.VertexAttribPointer(uint32(sh.position), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetPos)))
	gl.VertexAttribPointer(uint32(sh.color), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetCol)))
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

func newColorShader(yflipped bool) shaderProgram {
	sh := &guiShader{}
	if yflipped {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.ColorShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.ColorShader))
	}
	return sh
}

type dbgScreenShader struct {
	shader

	fb  *framebuffer
	crt shaderProgram

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

	sh.fb = newFramebuffer(3)
	sh.crt = newCRTShader(sh.fb)

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

func (sh *dbgScreenShader) destroy() {
	sh.fb.destroy()
	sh.crt.destroy()
}

func (sh *dbgScreenShader) setAttributes(env shaderEnvironment) {
	env.img.screen.crit.section.Lock()
	width := env.img.wm.dbgScr.scaledWidth(env.img.wm.dbgScr.cropped)
	height := env.img.wm.dbgScr.scaledHeight(env.img.wm.dbgScr.cropped)
	env.img.screen.crit.section.Unlock()

	env.width = int32(width)
	env.height = int32(height)

	ox := int32(env.img.wm.dbgScr.screenOrigin.X)
	oy := int32(env.img.wm.dbgScr.screenOrigin.Y)
	gl.Viewport(-ox, -oy, env.width+ox, env.height+oy)
	gl.Scissor(-ox, -oy, env.width+ox, env.height+oy)

	env.internalProj = [4][4]float32{
		{2.0 / (width + float32(ox)), 0.0, 0.0, 0.0},
		{0.0, 2.0 / -(height + float32(oy)), 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	env.srcTextureID = sh.crt.(*crtShader).setAttributesCRT(env, env.img.wm.dbgScr.crt, true)

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

type effectsShader struct {
	shader

	screenDim       int32
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

func newEffectsShader(yflip bool) shaderProgram {
	sh := &effectsShader{}
	if yflip {
		sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTEffectsFragShader))
	} else {
		sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTEffectsFragShader))
	}

	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
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

func (sh *effectsShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1i(sh.curve, boolToInt32(env.img.crtPrefs.Curve.Get().(bool)))
	gl.Uniform1i(sh.shadowMask, boolToInt32(env.img.crtPrefs.Mask.Get().(bool)))
	gl.Uniform1i(sh.scanlines, boolToInt32(env.img.crtPrefs.Scanlines.Get().(bool)))
	gl.Uniform1i(sh.noise, boolToInt32(env.img.crtPrefs.Noise.Get().(bool)))
	gl.Uniform1i(sh.fringing, boolToInt32(env.img.crtPrefs.Fringing.Get().(bool)))
	gl.Uniform1f(sh.curveAmount, float32(env.img.crtPrefs.CurveAmount.Get().(float64)))
	gl.Uniform1f(sh.maskBright, float32(env.img.crtPrefs.MaskBright.Get().(float64)))
	gl.Uniform1f(sh.scanlinesBright, float32(env.img.crtPrefs.ScanlinesBright.Get().(float64)))
	gl.Uniform1f(sh.noiseLevel, float32(env.img.crtPrefs.NoiseLevel.Get().(float64)))
	gl.Uniform1f(sh.fringingAmount, float32(env.img.crtPrefs.FringingAmount.Get().(float64)))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
}

type phosphorShader struct {
	shader
	latency int32
}

func newPhosphorShader() shaderProgram {
	sh := &phosphorShader{}
	sh.createProgram(string(shaders.YFlipVertexShader), string(shaders.CRTPhosphorFragShader))
	sh.latency = gl.GetUniformLocation(sh.handle, gl.Str("Latency"+"\x00"))
	return sh
}

func (sh *phosphorShader) setAttributesArgs(env shaderEnvironment, latency float32, textureB uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.latency, latency)

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

func (sh *blurShader) setAttributesArgs(env shaderEnvironment, blur float32) {
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

func (sh *blendShader) setAttributesArgs(env shaderEnvironment, modulate float32, textureB uint32) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.modulate, modulate)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, uint32(textureB))
	gl.Uniform1i(sh.texture, 1)
}

type crtShader struct {
	shader

	fb *framebuffer

	phosphorShader       shaderProgram
	blurShader           shaderProgram
	blendShader          shaderProgram
	effectsShader        shaderProgram
	colorShader          shaderProgram
	effectsShaderFlipped shaderProgram
	colorShaderFlipped   shaderProgram
}

func newCRTShader(fb *framebuffer) shaderProgram {
	sh := &crtShader{
		fb:                   fb,
		phosphorShader:       newPhosphorShader(),
		blurShader:           newBlurShader(),
		blendShader:          newBlendShader(),
		effectsShader:        newEffectsShader(false),
		colorShader:          newColorShader(false),
		effectsShaderFlipped: newEffectsShader(true),
		colorShaderFlipped:   newColorShader(true),
	}
	return sh
}

func (sh *crtShader) destroy() {
	sh.phosphorShader.destroy()
	sh.blurShader.destroy()
	sh.blendShader.destroy()
	sh.effectsShader.destroy()
	sh.colorShader.destroy()
	sh.shader.destroy()
	sh.fb.destroy()
}

// moreProcessing should be true if more shaders are to be applied to the framebuffer before presentation
func (sh *crtShader) setAttributesCRT(env shaderEnvironment, enabled bool, moreProcessing bool) uint32 {
	// make sure our framebuffer is correct
	//
	// any changes to the framebuffer will effect how the next frame is drawn.
	// we get rid of any phosphor effects and there is no blending stage
	//
	// there is an artifact whereby the screen seems to brighten when the frame
	// is being changed. I'm not sure what's causing this but it is something
	// that should be fixed
	//
	// !!TODO: eliminate frame brightening on size change
	changed := sh.fb.setup(env.width, env.height)

	env.isInternalShader = true
	src := env.srcTextureID

	const (
		currentID  = 0
		phosphorID = 1
		finalID    = 2
	)

	if enabled {
		if !changed {
			if env.img.crtPrefs.Phosphor.Get().(bool) {
				// use blur shader to add bloom to previous phosphor
				env.srcTextureID = sh.fb.textures[phosphorID]
				env.srcTextureID = sh.fb.draw(phosphorID, func() {
					phosphorBloom := env.img.crtPrefs.PhosphorBloom.Get().(float64)
					sh.blurShader.(*blurShader).setAttributesArgs(env, float32(phosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.srcTextureID = sh.fb.draw(phosphorID, func() {
				env.srcTextureID = src
				phosphorLatency := env.img.crtPrefs.PhosphorLatency.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(phosphorLatency), sh.fb.textures[phosphorID])
				env.draw()
			})
		}
	} else {
		if !changed {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.srcTextureID = sh.fb.draw(phosphorID, func() {
				fade := env.img.crtPrefs.PixelPerfectFade.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(fade), sh.fb.textures[phosphorID])
				env.draw()
			})
		}
	}

	if enabled {
		// blur for current frame
		env.srcTextureID = sh.fb.draw(currentID, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
			env.draw()
		})

		if !changed {
			// blend blur with original source texture
			env.srcTextureID = sh.fb.draw(currentID, func() {
				env.srcTextureID = src
				sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, sh.fb.textures[currentID])
				env.draw()
			})
		}

		if moreProcessing {
			sh.fb.clear(finalID)
			env.srcTextureID = sh.fb.draw(finalID, func() {
				sh.effectsShaderFlipped.setAttributes(env)
				env.draw()
			})
		} else {
			env.isInternalShader = false
			sh.effectsShader.setAttributes(env)
		}
	} else {
		if moreProcessing {
			env.srcTextureID = sh.fb.draw(finalID, func() {
				sh.colorShaderFlipped.setAttributes(env)
				env.draw()
			})
		} else {
			env.isInternalShader = false
			sh.colorShader.setAttributes(env)
		}
	}

	return env.srcTextureID
}

type playscrShader struct {
	crt shaderProgram
}

func newPlayscrShader() shaderProgram {
	sh := &playscrShader{
		crt: newCRTShader(newFramebuffer(2)),
	}
	return sh
}

func (sh *playscrShader) destroy() {
	sh.crt.destroy()
}

func (sh *playscrShader) setAttributes(env shaderEnvironment) {
	if !env.img.isPlaymode() {
		return
	}

	env.img.screen.crit.section.Lock()
	env.width = int32(env.img.playScr.scaledWidth())
	env.height = int32(env.img.playScr.scaledHeight())
	env.img.screen.crit.section.Unlock()

	env.internalProj = env.presentationProj

	// set scissor and viewport
	gl.Viewport(int32(-env.img.playScr.imagePosMin.X),
		int32(-env.img.playScr.imagePosMin.Y),
		env.width+(int32(env.img.playScr.imagePosMin.X*2)),
		env.height+(int32(env.img.playScr.imagePosMin.Y*2)),
	)
	gl.Scissor(int32(-env.img.playScr.imagePosMin.X),
		int32(-env.img.playScr.imagePosMin.Y),
		env.width+(int32(env.img.playScr.imagePosMin.X*2)),
		env.height+(int32(env.img.playScr.imagePosMin.Y*2)),
	)

	enabled := env.img.crtPrefs.Enabled.Get().(bool)
	sh.crt.(*crtShader).setAttributesCRT(env, enabled, false)
}