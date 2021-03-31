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
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/crt/shaders"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// texture units to use for the various phosphor textures. unlike the other
// textures we have to use different units because (I think - my OpenGL-fu
// isn't very advanced) the other textures are put into the imgui drawlist
// which handles loading of the textures into unit 0.
//
// when used with gl.ActiveTexture() they should be added to gl.TEXTURE0, like
// so:
//
//		gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitPlayScr)
//		gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
//
// and when used to load the texture into the shader the unit specified rather
// than an offset from whatever gl.TEXTURE0 is:
//
//		gl.Uniform1i(rnd.attribPhosphorTexture, phosphorTextureUnitPlayScr)
//
const (
	phosphorTextureUnitDbgScr   = 1
	phosphorTextureUnitPlayScr  = 2
	phosphorTextureUnitPrefsCRT = 3
)

type glsl struct {
	img *SdlImgui

	gopher2600Icons     imgui.Font
	gopher2600IconsSize float32

	// font texture given to imgui. we take charge of its destruction
	fontTexture uint32

	// handle for the compiled and linked shader program. we don't need to keep
	// references to the component parts of the program, they're created and
	// destroyed all within the setup() function
	shaderHandle uint32

	vboHandle      uint32
	elementsHandle uint32

	// "attrib" variables are the "communication" points between the shader
	// program and the host language. "uniform" variables remain constant for
	// the duration of each shader program executrion. non-uniform variables
	// meanwhile change from one iteration to the next.
	attribProjMtx  int32 // uniform
	attribPosition int32
	attribUV       int32
	attribColor    int32

	attribTexture         int32 // uniform
	attribPhosphorTexture int32 // uniform

	// imagetype differentaites the screen texture from the rest of the imgui
	// interface
	attribImageType int32 // uniform

	// the following attrib variables are strictly for the screen texture
	attribShowCursor         int32 // uniform
	attribIsCropped          int32 // uniform
	attribScreenDim          int32 // uniform
	attribUncroppedScreenDim int32 // uniform
	attribScalingX           int32 // uniform
	attribScalingY           int32 // uniform
	attribLastX              int32 // uniform
	attribLastY              int32 // uniform
	attribHblank             int32 // uniform
	attribTopScanline        int32 // uniform
	attribBotScanline        int32 // uniform
	attribOverlayAlpha       int32 // uniform

	attribEnableCRT           int32 // uniform
	attribEnablePhosphor      int32 // uniform
	attribEnableShadowMask    int32 // uniform
	attribEnableScanlines     int32 // uniform
	attribEnableNoise         int32 // uniform
	attribEnableBlur          int32 // uniform
	attribEnableVignette      int32 // uniform
	attribPhosphorSpeed       int32 // uniform
	attribMaskBrightness      int32 // uniform
	attribScanlinesBrightness int32 // uniform
	attribNoiseLevel          int32 // uniform
	attribBlurLevel           int32 // uniform
	attribRandSeed            int32 // uniform
}

func newGlsl(img *SdlImgui) (*glsl, error) {
	err := gl.Init()
	if err != nil {
		return nil, fmt.Errorf("glsl: %v", err)
	}

	rnd := &glsl{img: img}

	rnd.setup()
	err = rnd.setupFonts()
	if err != nil {
		return nil, fmt.Errorf("glsl: %v", err)
	}

	return rnd, nil
}

func (rnd *glsl) destroy() {
	if rnd.vboHandle != 0 {
		gl.DeleteBuffers(1, &rnd.vboHandle)
	}
	rnd.vboHandle = 0

	if rnd.elementsHandle != 0 {
		gl.DeleteBuffers(1, &rnd.elementsHandle)
	}
	rnd.elementsHandle = 0

	if rnd.shaderHandle != 0 {
		gl.DeleteProgram(rnd.shaderHandle)
	}
	rnd.shaderHandle = 0

	if rnd.fontTexture != 0 {
		gl.DeleteTextures(1, &rnd.fontTexture)
		imgui.CurrentIO().Fonts().SetTextureID(0)
		rnd.fontTexture = 0
	}
}

// preRender clears the framebuffer.
func (rnd *glsl) preRender() {
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func boolToInt32(v bool) int32 {
	if v {
		return shaders.True
	}
	return shaders.False
}

// render translates the ImGui draw data to OpenGL3 commands.
func (rnd *glsl) render() {
	displaySize := rnd.img.plt.displaySize()
	framebufferSize := rnd.img.plt.framebufferSize()
	drawData := imgui.RenderedDrawData()

	st := storeGLState()
	defer st.restoreGLState()

	// Avoid rendering when minimised, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := displaySize[0], displaySize[1]
	fbWidth, fbHeight := framebufferSize[0], framebufferSize[1]
	if (fbWidth <= 0) || (fbHeight <= 0) {
		return
	}
	drawData.ScaleClipRects(imgui.Vec2{
		X: fbWidth / displayWidth,
		Y: fbHeight / displayHeight,
	})

	// Setup render state: alpha-blending enabled, no face culling, no depth testing, scissor enabled, polygon fill
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Setup viewport, orthographic projection matrix
	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.
	gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
	orthoProjection := [4][4]float32{
		{2.0 / displayWidth, 0.0, 0.0, 0.0},
		{0.0, 2.0 / -displayHeight, 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	// shader options for shader program
	gl.UseProgram(rnd.shaderHandle)

	gl.Uniform1i(rnd.attribTexture, 0)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.UniformMatrix4fv(rnd.attribProjMtx, 1, false, &orthoProjection[0][0])
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	var vaoHandle uint32
	gl.GenVertexArrays(1, &vaoHandle)
	gl.BindVertexArray(vaoHandle)
	gl.BindBuffer(gl.ARRAY_BUFFER, rnd.vboHandle)
	gl.EnableVertexAttribArray(uint32(rnd.attribPosition))
	gl.EnableVertexAttribArray(uint32(rnd.attribUV))
	gl.EnableVertexAttribArray(uint32(rnd.attribColor))
	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointer(uint32(rnd.attribUV), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetUv)))
	gl.VertexAttribPointer(uint32(rnd.attribPosition), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetPos)))
	gl.VertexAttribPointer(uint32(rnd.attribColor), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetCol)))
	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	if indexSize == 4 {
		drawType = gl.UNSIGNED_INT
	}

	for _, list := range drawData.CommandLists() {
		var indexBufferOffset uintptr

		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, rnd.vboHandle)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.IndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, rnd.elementsHandle)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				// notify the shader which texture to work with
				textureID := uint32(cmd.TextureID())
				switch textureID {
				case rnd.img.wm.dbgScr.screenTexture:
					rnd.debugScr()
				case rnd.img.wm.dbgScr.elementsTexture:
					rnd.elements()
				case rnd.img.wm.dbgScr.overlayTexture:
					rnd.overlay()
				case rnd.img.playScr.screenTexture:
					rnd.playScr()
				case rnd.img.wm.crtPrefs.crtTexture:
					rnd.prefsCRT()
				default:
					rnd.gui()
				}

				rnd.setOptions(textureID)

				// clipping
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				gl.BindTexture(gl.TEXTURE_2D, textureID)
				gl.DrawElements(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), unsafe.Pointer(indexBufferOffset))
			}
			indexBufferOffset += uintptr(cmd.ElementCount() * indexSize)
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)
}

func (rnd *glsl) gui() {
	gl.Uniform1i(rnd.attribImageType, shaders.GUI)
}

func (rnd *glsl) debugScr() {
	gl.Uniform1i(rnd.attribImageType, shaders.DebugScr)
	gl.Uniform1i(rnd.attribPhosphorTexture, phosphorTextureUnitDbgScr)
}

func (rnd *glsl) elements() {
	gl.Uniform1i(rnd.attribImageType, shaders.Elements)
}

func (rnd *glsl) overlay() {
	gl.Uniform1i(rnd.attribImageType, shaders.Overlay)
	gl.Uniform1f(rnd.attribOverlayAlpha, rnd.img.wm.dbgScr.overlayAlpha)
}

func (rnd *glsl) playScr() {
	gl.Uniform1i(rnd.attribImageType, shaders.PlayScr)
	gl.Uniform1i(rnd.attribPhosphorTexture, phosphorTextureUnitPlayScr)
}

func (rnd *glsl) prefsCRT() {
	gl.Uniform1i(rnd.attribImageType, shaders.PrefsCRT)
	gl.Uniform1i(rnd.attribPhosphorTexture, phosphorTextureUnitPrefsCRT)
}

func (rnd *glsl) setOptions(textureID uint32) {
	// scaling of screen
	var vertScaling float32
	var horizScaling float32
	if rnd.img.isPlaymode() {
		vertScaling = rnd.img.playScr.scaling
		horizScaling = rnd.img.playScr.horizScaling()
	} else {
		vertScaling = rnd.img.wm.dbgScr.scaling
		horizScaling = rnd.img.wm.dbgScr.horizScaling()
	}

	// crt preferences. for playmode the stored preferences are used and for
	// the debug screen the local CRT boolean is used
	var crt bool
	if rnd.img.isPlaymode() {
		crt = rnd.img.crtPrefs.Enabled.Get().(bool)
	} else {
		crt = rnd.img.wm.dbgScr.crt
	}

	// crt preferences. we always set these because they're the same whatever
	// the texture that uses them.
	gl.Uniform1i(rnd.attribEnableCRT, boolToInt32(crt))
	gl.Uniform1i(rnd.attribEnablePhosphor, boolToInt32(rnd.img.crtPrefs.Phosphor.Get().(bool)))
	gl.Uniform1i(rnd.attribEnableShadowMask, boolToInt32(rnd.img.crtPrefs.Mask.Get().(bool)))
	gl.Uniform1i(rnd.attribEnableScanlines, boolToInt32(rnd.img.crtPrefs.Scanlines.Get().(bool)))
	gl.Uniform1i(rnd.attribEnableNoise, boolToInt32(rnd.img.crtPrefs.Noise.Get().(bool)))
	gl.Uniform1i(rnd.attribEnableBlur, boolToInt32(rnd.img.crtPrefs.Blur.Get().(bool)))
	gl.Uniform1i(rnd.attribEnableVignette, boolToInt32(rnd.img.crtPrefs.Vignette.Get().(bool)))
	gl.Uniform1f(rnd.attribPhosphorSpeed, float32(rnd.img.crtPrefs.PhosphorSpeed.Get().(float64)))
	gl.Uniform1f(rnd.attribMaskBrightness, float32(rnd.img.crtPrefs.MaskBrightness.Get().(float64)))
	gl.Uniform1f(rnd.attribScanlinesBrightness, float32(rnd.img.crtPrefs.ScanlinesBrightness.Get().(float64)))
	gl.Uniform1f(rnd.attribNoiseLevel, float32(rnd.img.crtPrefs.NoiseLevel.Get().(float64)))
	gl.Uniform1f(rnd.attribBlurLevel, float32(rnd.img.crtPrefs.BlurLevel.Get().(float64)))
	gl.Uniform1f(rnd.attribRandSeed, float32(time.Now().Nanosecond())/100000000.0)

	// critical section
	rnd.img.screen.crit.section.Lock()

	// the resolution information is used to scale the debugging guides
	switch textureID {
	case rnd.img.wm.dbgScr.screenTexture:
		fallthrough
	case rnd.img.wm.dbgScr.elementsTexture:
		fallthrough
	case rnd.img.wm.dbgScr.overlayTexture:
		gl.Uniform1f(rnd.attribScalingX, rnd.img.wm.dbgScr.horizScaling())
		gl.Uniform1f(rnd.attribScalingY, rnd.img.wm.dbgScr.scaling)
		gl.Uniform2f(rnd.attribUncroppedScreenDim, rnd.img.wm.dbgScr.scaledWidth(false), rnd.img.wm.dbgScr.scaledHeight(false))
		gl.Uniform2f(rnd.attribScreenDim, rnd.img.wm.dbgScr.scaledWidth(true), rnd.img.wm.dbgScr.scaledHeight(true))
		gl.Uniform1i(rnd.attribIsCropped, boolToInt32(rnd.img.wm.dbgScr.cropped))

		cursorX := rnd.img.screen.crit.lastX
		cursorY := rnd.img.screen.crit.lastY

		if rnd.img.wm.dbgScr.cropped {
			gl.Uniform1f(rnd.attribLastX, float32(cursorX-specification.ClksHBlank)*horizScaling)
		} else {
			gl.Uniform1f(rnd.attribLastX, float32(cursorX)*horizScaling)
		}
		gl.Uniform1f(rnd.attribLastY, float32(cursorY)*vertScaling)

	case rnd.img.playScr.screenTexture:
		gl.Uniform2f(rnd.attribScreenDim, rnd.img.playScr.scaledWidth(), rnd.img.playScr.scaledHeight())
		gl.Uniform1f(rnd.attribScalingX, rnd.img.playScr.horizScaling())
		gl.Uniform1f(rnd.attribScalingY, rnd.img.playScr.scaling)
		gl.Uniform1i(rnd.attribIsCropped, shaders.True)

	case rnd.img.wm.crtPrefs.crtTexture:
		gl.Uniform2f(rnd.attribScreenDim, rnd.img.wm.crtPrefs.getScaledWidth(), rnd.img.wm.crtPrefs.getScaledHeight())
		gl.Uniform1f(rnd.attribScalingX, rnd.img.wm.crtPrefs.getScaling(true))
		gl.Uniform1f(rnd.attribScalingY, rnd.img.wm.crtPrefs.getScaling(false))
		gl.Uniform1i(rnd.attribIsCropped, shaders.True)
	}

	// screen geometry
	gl.Uniform1f(rnd.attribHblank, specification.ClksHBlank*horizScaling)
	gl.Uniform1f(rnd.attribTopScanline, float32(rnd.img.screen.crit.topScanline)*vertScaling)
	gl.Uniform1f(rnd.attribBotScanline, float32(rnd.img.screen.crit.bottomScanline)*vertScaling)

	rnd.img.screen.crit.section.Unlock()
	// end of critical section

	// whether we show the cursor depends on the current GUI state
	switch rnd.img.state {
	case gui.StatePaused:
		gl.Uniform1i(rnd.attribShowCursor, shaders.True)
	case gui.StateRunning:
		// if FPS is low enough then show screen draw even though
		// emulation is running
		if rnd.img.lz.TV.ReqFPS < television.ThreshVisual {
			gl.Uniform1i(rnd.attribShowCursor, shaders.True)
		} else {
			gl.Uniform1i(rnd.attribShowCursor, shaders.False)
		}
	case gui.StateStepping:
		gl.Uniform1i(rnd.attribShowCursor, shaders.True)
	case gui.StateRewinding:
		gl.Uniform1i(rnd.attribShowCursor, shaders.False)
	}
}

func (rnd *glsl) setup() {
	// compile and link shader programs
	rnd.shaderHandle = gl.CreateProgram()
	vertHandle := gl.CreateShader(gl.VERTEX_SHADER)
	fragHandle := gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source string) {
		csource, free := gl.Strs(source + "\x00")
		defer free()

		gl.ShaderSource(handle, 1, csource, nil)
	}

	// vertex and fragment glsl source defined in shaders.go (a generated file)
	glShaderSource(vertHandle, shaders.Vertex)
	glShaderSource(fragHandle, shaders.Fragment)

	gl.CompileShader(vertHandle)
	if log := getShaderCompileError(vertHandle); log != "" {
		panic(log)
	}

	gl.CompileShader(fragHandle)
	if log := getShaderCompileError(fragHandle); log != "" {
		panic(log)
	}

	gl.AttachShader(rnd.shaderHandle, vertHandle)
	gl.AttachShader(rnd.shaderHandle, fragHandle)
	gl.LinkProgram(rnd.shaderHandle)

	// now that the shader promer has linked we no longer need the individual
	// shader programs
	gl.DeleteShader(fragHandle)
	gl.DeleteShader(vertHandle)

	// get references to shader attributes and uniforms variables

	rnd.attribProjMtx = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ProjMtx"+"\x00"))
	rnd.attribPosition = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("Position"+"\x00"))
	rnd.attribUV = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("UV"+"\x00"))
	rnd.attribColor = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("Color"+"\x00"))

	rnd.attribTexture = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Texture"+"\x00"))
	rnd.attribPhosphorTexture = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("PhosphorTexture"+"\x00"))

	rnd.attribImageType = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ImageType"+"\x00"))

	rnd.attribShowCursor = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ShowCursor"+"\x00"))
	rnd.attribIsCropped = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("IsCropped"+"\x00"))
	rnd.attribScreenDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScreenDim"+"\x00"))
	rnd.attribUncroppedScreenDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("UncroppedScreenDim"+"\x00"))
	rnd.attribScalingX = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScalingX"+"\x00"))
	rnd.attribScalingY = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScalingY"+"\x00"))
	rnd.attribLastX = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastX"+"\x00"))
	rnd.attribLastY = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastY"+"\x00"))
	rnd.attribHblank = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Hblank"+"\x00"))
	rnd.attribTopScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("TopScanline"+"\x00"))
	rnd.attribBotScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("BotScanline"+"\x00"))
	rnd.attribOverlayAlpha = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("OverlayAlpha"+"\x00"))

	rnd.attribEnableCRT = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableCRT"+"\x00"))
	rnd.attribEnablePhosphor = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnablePhosphor"+"\x00"))
	rnd.attribEnableShadowMask = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableShadowMask"+"\x00"))
	rnd.attribEnableScanlines = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableScanlines"+"\x00"))
	rnd.attribEnableNoise = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableNoise"+"\x00"))
	rnd.attribEnableBlur = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableBlur"+"\x00"))
	rnd.attribEnableVignette = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("EnableVignette"+"\x00"))

	rnd.attribPhosphorSpeed = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("PhosphorSpeed"+"\x00"))
	rnd.attribMaskBrightness = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("MaskBrightness"+"\x00"))
	rnd.attribScanlinesBrightness = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScanlinesBrightness"+"\x00"))
	rnd.attribNoiseLevel = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("NoiseLevel"+"\x00"))
	rnd.attribBlurLevel = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("BlurLevel"+"\x00"))
	rnd.attribRandSeed = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("RandSeed"+"\x00"))

	gl.GenBuffers(1, &rnd.vboHandle)
	gl.GenBuffers(1, &rnd.elementsHandle)
}

func (rnd *glsl) setupFonts() error {
	// add default font
	atlas := imgui.CurrentIO().Fonts()
	atlas.AddFontDefault()

	// config for font loading. merging with default font and adjusting offset
	// so that the icons align better.
	mergeConfig := imgui.NewFontConfig()
	defer mergeConfig.Delete()
	mergeConfig.SetMergeMode(true)
	mergeConfig.SetPixelSnapH(true)
	mergeConfig.SetGlyphOffsetY(1.0)

	// limit what glyphs we load
	var glyphBuilder imgui.GlyphRangesBuilder
	glyphBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	// load font
	font := atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, 13.0, mergeConfig, glyphBuilder.Build().GlyphRanges)
	if font == 0 {
		return curated.Errorf("font: error loading font from memory")
	}

	// load large icons
	gopher2600IconConfig := imgui.NewFontConfig()
	defer gopher2600IconConfig.Delete()
	gopher2600IconConfig.SetPixelSnapH(true)
	gopher2600IconConfig.SetGlyphOffsetY(1.0)

	var largeIconBuilder imgui.GlyphRangesBuilder
	largeIconBuilder.Add(fonts.Gopher2600IconMin, fonts.Gopher2600IconMax)

	rnd.gopher2600IconsSize = 52.0
	rnd.gopher2600Icons = atlas.AddFontFromMemoryTTFV(fonts.Gopher2600Icons, rnd.gopher2600IconsSize, gopher2600IconConfig, largeIconBuilder.Build().GlyphRanges)
	if font == 0 {
		return curated.Errorf("font: error loading font from memory")
	}

	// create font texture
	image := atlas.TextureDataAlpha8()
	gl.GenTextures(1, &rnd.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, rnd.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height), 0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)
	atlas.SetTextureID(imgui.TextureID(rnd.fontTexture))

	return nil
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

// glState stores GL state with the intention of restoration after a short period.
type glState struct {
	lastActiveTexture      int32
	lastProgram            int32
	lastTexture            int32
	lastSampler            int32
	lastArrayBuffer        int32
	lastElementArrayBuffer int32
	lastVertexArray        int32
	lastPolygonMode        [2]int32
	lastViewport           [4]int32
	lastScissorBox         [4]int32
	lastBlendSrcRgb        int32
	lastBlendDstRgb        int32
	lastBlendSrcAlpha      int32
	lastBlendDstAlpha      int32
	lastBlendEquationRgb   int32
	lastBlendEquationAlpha int32
	lastEnableBlend        bool
	lastEnableCullFace     bool
	lastEnableDepthTest    bool
	lastEnableScissorTest  bool
}

// storeGLState is the best way of initialising an instance of glState.
func storeGLState() *glState {
	st := &glState{}
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &st.lastActiveTexture)
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &st.lastProgram)
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &st.lastTexture)
	gl.GetIntegerv(gl.SAMPLER_BINDING, &st.lastSampler)
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &st.lastArrayBuffer)
	gl.GetIntegerv(gl.ELEMENT_ARRAY_BUFFER_BINDING, &st.lastElementArrayBuffer)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &st.lastVertexArray)
	gl.GetIntegerv(gl.POLYGON_MODE, &st.lastPolygonMode[0])
	gl.GetIntegerv(gl.VIEWPORT, &st.lastViewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &st.lastScissorBox[0])
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &st.lastBlendSrcRgb)
	gl.GetIntegerv(gl.BLEND_DST_RGB, &st.lastBlendDstRgb)
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &st.lastBlendSrcAlpha)
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &st.lastBlendDstAlpha)
	gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &st.lastBlendEquationRgb)
	gl.GetIntegerv(gl.BLEND_EQUATION_ALPHA, &st.lastBlendEquationAlpha)
	st.lastEnableBlend = gl.IsEnabled(gl.BLEND)
	st.lastEnableCullFace = gl.IsEnabled(gl.CULL_FACE)
	st.lastEnableDepthTest = gl.IsEnabled(gl.DEPTH_TEST)
	st.lastEnableScissorTest = gl.IsEnabled(gl.SCISSOR_TEST)
	return st
}

// restoreGLState previously store glState.
func (st *glState) restoreGLState() {
	gl.UseProgram(uint32(st.lastProgram))
	gl.BindTexture(gl.TEXTURE_2D, uint32(st.lastTexture))
	gl.BindSampler(0, uint32(st.lastSampler))
	gl.ActiveTexture(uint32(st.lastActiveTexture))
	gl.BindVertexArray(uint32(st.lastVertexArray))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(st.lastArrayBuffer))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(st.lastElementArrayBuffer))
	gl.BlendEquationSeparate(uint32(st.lastBlendEquationRgb), uint32(st.lastBlendEquationAlpha))
	gl.BlendFuncSeparate(uint32(st.lastBlendSrcRgb), uint32(st.lastBlendDstRgb), uint32(st.lastBlendSrcAlpha), uint32(st.lastBlendDstAlpha))
	if st.lastEnableBlend {
		gl.Enable(gl.BLEND)
	} else {
		gl.Disable(gl.BLEND)
	}
	if st.lastEnableCullFace {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	if st.lastEnableDepthTest {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	if st.lastEnableScissorTest {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.PolygonMode(gl.FRONT_AND_BACK, uint32(st.lastPolygonMode[0]))
	gl.Viewport(st.lastViewport[0], st.lastViewport[1], st.lastViewport[2], st.lastViewport[3])
	gl.Scissor(st.lastScissorBox[0], st.lastScissorBox[1], st.lastScissorBox[2], st.lastScissorBox[3])
}
