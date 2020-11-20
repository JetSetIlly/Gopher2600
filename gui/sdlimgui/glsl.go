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
	"math"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/crt/shaders"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type glsl struct {
	imguiIO imgui.IO
	img     *SdlImgui

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
	attribTexture  int32 // uniform
	attribProjMtx  int32 // uniform
	attribPosition int32
	attribUV       int32
	attribColor    int32

	// imagetype differentaites the screen texture from the rest of the imgui
	// interface
	attribImageType int32 // uniform

	// the following attrib variables are strictly for the screen texture
	attribScreenDim     int32 // uniform
	attribCropScreenDim int32 // uniform
	attribDrawMode      int32 // uniform
	attribScalingX      int32 // uniform
	attribScalingY      int32 // uniform
	attribCropped       int32 // uniform
	attribLastX         int32 // uniform
	attribLastY         int32 // uniform
	attribHblank        int32 // uniform
	attribTopScanline   int32 // uniform
	attribBotScanline   int32 // uniform
	attribAnimTime      int32 // uniform
	attribRandSeed      int32 // uniform

	attribCRT                 int32 // uniform
	attribInputGamma          int32 // uniform
	attribOutputGamma         int32 // uniform
	attribMask                int32 // uniform
	attribScanlines           int32 // uniform
	attribNoise               int32 // uniform
	attribMaskBrightness      int32 // uniform
	attribScanlinesBrightness int32 // uniform
	attribNoiseLevel          int32 // uniform
	attribVignette            int32 // uniform
	attribMaskScanlineScaling int32 // uniform
}

func newGlsl(io imgui.IO, img *SdlImgui) (*glsl, error) {
	err := gl.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialise OpenGL: %v", err)
	}

	rnd := &glsl{
		imguiIO: io,
		img:     img,
	}

	rnd.setup()

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
		return 1
	}
	return 0
}

// render translates the ImGui draw data to OpenGL3 commands.
func (rnd *glsl) render(displaySize [2]float32, framebufferSize [2]float32, drawData imgui.DrawData) {
	st := storeGLState()
	defer st.restore()

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
	gl.UseProgram(rnd.shaderHandle)
	gl.Uniform1i(rnd.attribTexture, 0)
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

	gl.ActiveTexture(gl.TEXTURE0)

	for _, list := range drawData.CommandLists() {
		var indexBufferOffset uintptr

		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, rnd.vboHandle)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.IndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, rnd.elementsHandle)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		// !!TODO: different scaling values for different screen windows

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				vertScaling := rnd.img.wm.dbgScr.getScaling(false)
				horizScaling := rnd.img.wm.dbgScr.getScaling(true)

				// crt preferences
				gl.Uniform1i(rnd.attribCRT, boolToInt32(rnd.img.wm.dbgScr.crt))
				gl.Uniform1f(rnd.attribInputGamma, float32(rnd.img.crtPrefs.InputGamma.Get().(float64)))
				gl.Uniform1f(rnd.attribOutputGamma, float32(rnd.img.crtPrefs.OutputGamma.Get().(float64)))
				gl.Uniform1i(rnd.attribMask, boolToInt32(rnd.img.crtPrefs.Mask.Get().(bool)))
				gl.Uniform1i(rnd.attribScanlines, boolToInt32(rnd.img.crtPrefs.Scanlines.Get().(bool)))
				gl.Uniform1i(rnd.attribNoise, boolToInt32(rnd.img.crtPrefs.Noise.Get().(bool)))
				gl.Uniform1f(rnd.attribMaskBrightness, float32(rnd.img.crtPrefs.MaskBrightness.Get().(float64)))
				gl.Uniform1f(rnd.attribScanlinesBrightness, float32(rnd.img.crtPrefs.ScanlinesBrightness.Get().(float64)))
				gl.Uniform1f(rnd.attribNoiseLevel, float32(rnd.img.crtPrefs.NoiseLevel.Get().(float64)))
				gl.Uniform1i(rnd.attribVignette, boolToInt32(rnd.img.crtPrefs.Vignette.Get().(bool)))
				gl.Uniform1i(rnd.attribMaskScanlineScaling, int32(rnd.img.crtPrefs.MaskScanlineScaling.Get().(int)))

				// critical section
				rnd.img.screen.crit.section.Lock()

				// the resolution information is used to scale the Last
				gl.Uniform2f(rnd.attribScreenDim, rnd.img.wm.dbgScr.getScaledWidth(false), rnd.img.wm.dbgScr.getScaledHeight(false))
				gl.Uniform2f(rnd.attribCropScreenDim, rnd.img.wm.dbgScr.getScaledWidth(true), rnd.img.wm.dbgScr.getScaledHeight(true))
				gl.Uniform1f(rnd.attribScalingX, rnd.img.wm.dbgScr.getScaling(true))
				gl.Uniform1f(rnd.attribScalingY, rnd.img.wm.dbgScr.getScaling(false))

				// screen geometry
				gl.Uniform1f(rnd.attribHblank, specification.HorizClksHBlank*horizScaling)
				gl.Uniform1f(rnd.attribTopScanline, float32(rnd.img.screen.crit.topScanline)*vertScaling)
				gl.Uniform1f(rnd.attribBotScanline, float32(rnd.img.screen.crit.topScanline+rnd.img.screen.crit.scanlines)*vertScaling)

				// the coordinates of the last plot. specual handling for StateGotoCoords
				var cursorX int
				var cursorY int
				if rnd.img.state == gui.StateGotoCoords {
					cursorX = rnd.img.screen.gotoCoordsX
					cursorY = rnd.img.screen.gotoCoordsY
				} else {
					cursorX = rnd.img.screen.crit.lastX
					cursorY = rnd.img.screen.crit.lastY
				}

				// scale cordinates. horizontal scaling depends on whether the
				// screen is cropped
				if rnd.img.wm.dbgScr.cropped {
					gl.Uniform1f(rnd.attribLastX, float32(cursorX-specification.HorizClksHBlank)*horizScaling)
				} else {
					gl.Uniform1f(rnd.attribLastX, float32(cursorX)*horizScaling)
				}
				gl.Uniform1f(rnd.attribLastY, float32(cursorY)*vertScaling)

				rnd.img.screen.crit.section.Unlock()
				// end of critical section

				// set DrawMode according to emulation state
				switch rnd.img.state {
				case gui.StatePaused:
					gl.Uniform1i(rnd.attribDrawMode, 1)
				case gui.StateRunning:
					// if FPS is low enough then show screen draw even though
					// emulation is running
					if rnd.img.lz.TV.ReqFPS < 3.0 {
						gl.Uniform1i(rnd.attribDrawMode, 1)
					} else {
						gl.Uniform1i(rnd.attribDrawMode, 0)
					}
				case gui.StateRewinding:
					gl.Uniform1i(rnd.attribDrawMode, 0)
				case gui.StateGotoCoords:
					gl.Uniform1i(rnd.attribDrawMode, 2)
				}

				if rnd.img.wm.dbgScr.cropped {
					gl.Uniform1i(rnd.attribCropped, 1)
				} else {
					gl.Uniform1i(rnd.attribCropped, -1)
				}

				// animation time
				anim := math.Sin(float64(time.Now().Nanosecond()) / 1000000000.0)
				anim = math.Abs(anim)
				gl.Uniform1f(rnd.attribAnimTime, float32(anim))

				// random seed (for noise generator)
				gl.Uniform1f(rnd.attribRandSeed, float32(time.Now().Nanosecond())/1000000000.0)

				// notify the shader which texture to work with
				textureID := uint32(cmd.TextureID())
				switch textureID {
				case rnd.img.wm.dbgScr.screenTexture:
					gl.Uniform1i(rnd.attribImageType, 1)
				case rnd.img.wm.dbgScr.overlayTexture:
					gl.Uniform1i(rnd.attribImageType, 2)
				case rnd.img.wm.playScr.screenTexture:
					gl.Uniform1i(rnd.attribImageType, 3)
				case rnd.img.wm.crtPrefs.crtTexture:
					gl.Uniform1i(rnd.attribImageType, 4)
				default:
					gl.Uniform1i(rnd.attribImageType, 0)
				}

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

func (rnd *glsl) setup() {
	// we'll be modifying the GL state during this function so we need to save
	// and restore the existing state.
	var lastTexture int32
	var lastArrayBuffer int32
	var lastVertexArray int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)
	defer gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	defer gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	defer gl.BindVertexArray(uint32(lastVertexArray))

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
		fmt.Println(log)
	}

	gl.CompileShader(fragHandle)
	if log := getShaderCompileError(fragHandle); log != "" {
		fmt.Println(log)
	}

	gl.AttachShader(rnd.shaderHandle, vertHandle)
	gl.AttachShader(rnd.shaderHandle, fragHandle)
	gl.LinkProgram(rnd.shaderHandle)

	// now that the shader promer has linked we no longer need the individual
	// shader programs
	gl.DeleteShader(fragHandle)
	gl.DeleteShader(vertHandle)

	// get references to shader attributes and uniforms variables
	rnd.attribImageType = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ImageType"+"\x00"))
	rnd.attribScreenDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScreenDim"+"\x00"))
	rnd.attribCropScreenDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("CropScreenDim"+"\x00"))
	rnd.attribDrawMode = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("DrawMode"+"\x00"))
	rnd.attribScalingX = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScalingX"+"\x00"))
	rnd.attribScalingY = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScalingY"+"\x00"))
	rnd.attribCropped = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Cropped"+"\x00"))
	rnd.attribLastX = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastX"+"\x00"))
	rnd.attribLastY = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastY"+"\x00"))
	rnd.attribHblank = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Hblank"+"\x00"))
	rnd.attribTopScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("TopScanline"+"\x00"))
	rnd.attribBotScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("BotScanline"+"\x00"))
	rnd.attribAnimTime = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("AnimTime"+"\x00"))
	rnd.attribRandSeed = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("RandSeed"+"\x00"))

	rnd.attribCRT = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("CRT"+"\x00"))
	rnd.attribInputGamma = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("InputGamma"+"\x00"))
	rnd.attribOutputGamma = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("OutputGamma"+"\x00"))
	rnd.attribMask = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Mask"+"\x00"))
	rnd.attribScanlines = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Scanlines"+"\x00"))
	rnd.attribNoise = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Noise"+"\x00"))
	rnd.attribMaskBrightness = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("MaskBrightness"+"\x00"))
	rnd.attribScanlinesBrightness = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ScanlinesBrightness"+"\x00"))
	rnd.attribNoiseLevel = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("NoiseLevel"+"\x00"))
	rnd.attribVignette = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Vignette"+"\x00"))
	rnd.attribMaskScanlineScaling = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("MaskScanlineScaling"+"\x00"))

	rnd.attribTexture = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Texture"+"\x00"))
	rnd.attribProjMtx = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ProjMtx"+"\x00"))
	rnd.attribPosition = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("Position"+"\x00"))
	rnd.attribUV = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("UV"+"\x00"))
	rnd.attribColor = gl.GetAttribLocation(rnd.shaderHandle, gl.Str("Color"+"\x00"))

	gl.GenBuffers(1, &rnd.vboHandle)
	gl.GenBuffers(1, &rnd.elementsHandle)

	// \/\/\/ font preparation \/\/\/

	// Build texture atlas
	image := rnd.imguiIO.Fonts().TextureDataAlpha8()

	// Upload font texture to graphics system
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GenTextures(1, &rnd.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, rnd.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height),
		0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	// Store our identifier
	rnd.imguiIO.Fonts().SetTextureID(imgui.TextureID(rnd.fontTexture))
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

// restore previously store glState.
func (st *glState) restore() {
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
