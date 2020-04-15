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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/gui/shaders"
	"github.com/jetsetilly/gopher2600/television"
)

type glsl struct {
	imguiIO imgui.IO
	img     *SdlImgui

	// font texture given to imgui. we take charge of its destruction
	fontTextureID uint32

	// texture created and managed by the screen type
	screenTextureID uint32

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
	attribPixelPerfect   int32 // uniform
	attribDim            int32 // uniform
	attribCropDim        int32 // uniform
	attribShowScreenDraw int32 // uniform
	attribCropped        int32 // uniform
	attribLastX          int32 // uniform
	attribLastY          int32 // uniform
	attribHblank         int32 // uniform
	attribTopScanline    int32 // uniform
	attribBotScanline    int32 // uniform
	attribAnimTime       int32 // uniform
}

func newGlsl(io imgui.IO, img *SdlImgui) (*glsl, error) {
	err := gl.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenGL: %v", err)
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

	if rnd.fontTextureID != 0 {
		gl.DeleteTextures(1, &rnd.fontTextureID)
		imgui.CurrentIO().Fonts().SetTextureID(0)
		rnd.fontTextureID = 0
	}
}

// preRender clears the framebuffer.
func (rnd *glsl) preRender(clearColor [4]float32) {
	gl.ClearColor(clearColor[0], clearColor[1], clearColor[2], clearColor[3])
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// render translates the ImGui draw data to OpenGL3 commands.
func (rnd *glsl) render(displaySize [2]float32, framebufferSize [2]float32, drawData imgui.DrawData) {
	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := displaySize[0], displaySize[1]
	fbWidth, fbHeight := framebufferSize[0], framebufferSize[1]
	if (fbWidth <= 0) || (fbHeight <= 0) {
		return
	}
	drawData.ScaleClipRects(imgui.Vec2{
		X: fbWidth / displayWidth,
		Y: fbHeight / displayHeight,
	})

	// Backup GL state
	var lastActiveTexture int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &lastActiveTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	var lastProgram int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &lastProgram)
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	var lastSampler int32
	gl.GetIntegerv(gl.SAMPLER_BINDING, &lastSampler)
	var lastArrayBuffer int32
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	var lastElementArrayBuffer int32
	gl.GetIntegerv(gl.ELEMENT_ARRAY_BUFFER_BINDING, &lastElementArrayBuffer)
	var lastVertexArray int32
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)
	var lastPolygonMode [2]int32
	gl.GetIntegerv(gl.POLYGON_MODE, &lastPolygonMode[0])
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])
	var lastScissorBox [4]int32
	gl.GetIntegerv(gl.SCISSOR_BOX, &lastScissorBox[0])
	var lastBlendSrcRgb int32
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &lastBlendSrcRgb)
	var lastBlendDstRgb int32
	gl.GetIntegerv(gl.BLEND_DST_RGB, &lastBlendDstRgb)
	var lastBlendSrcAlpha int32
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &lastBlendSrcAlpha)
	var lastBlendDstAlpha int32
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &lastBlendDstAlpha)
	var lastBlendEquationRgb int32
	gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &lastBlendEquationRgb)
	var lastBlendEquationAlpha int32
	gl.GetIntegerv(gl.BLEND_EQUATION_ALPHA, &lastBlendEquationAlpha)
	lastEnableBlend := gl.IsEnabled(gl.BLEND)
	lastEnableCullFace := gl.IsEnabled(gl.CULL_FACE)
	lastEnableDepthTest := gl.IsEnabled(gl.DEPTH_TEST)
	lastEnableScissorTest := gl.IsEnabled(gl.SCISSOR_TEST)

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
	gl.VertexAttribPointer(uint32(rnd.attribPosition), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetPos)))
	gl.VertexAttribPointer(uint32(rnd.attribUV), 2, gl.FLOAT, false, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetUv)))
	gl.VertexAttribPointer(uint32(rnd.attribColor), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), unsafe.Pointer(uintptr(vertexOffsetCol)))
	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	if indexSize == 4 {
		drawType = gl.UNSIGNED_INT
	}

	// Draw
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

				textureID := uint32(cmd.TextureID())
				switch textureID {

				case rnd.screenTextureID:

					// critical section
					rnd.img.screen.crit.section.Lock()

					vertScaling := rnd.img.screen.vertScaling()
					horizScaling := rnd.img.screen.horizScaling()

					// id of screen type
					gl.Uniform1i(rnd.attribImageType, 1)

					// pixel perfect rendering or whether to apply the CRT
					// filters
					if rnd.img.screen.pixelPerfect {
						gl.Uniform1i(rnd.attribPixelPerfect, 1)
					} else {
						gl.Uniform1i(rnd.attribPixelPerfect, 0)
					}

					// the resolution information is used to scale the Last
					gl.Uniform2f(rnd.attribDim, rnd.img.screen.scaledWidth(), rnd.img.screen.scaledHeight())
					gl.Uniform2f(rnd.attribCropDim, rnd.img.screen.scaledCroppedWidth(), rnd.img.screen.scaledCroppedHeight())

					// screen geometry
					gl.Uniform1f(rnd.attribHblank, television.HorizClksHBlank*horizScaling)
					gl.Uniform1f(rnd.attribTopScanline, float32(rnd.img.screen.crit.topScanline)*vertScaling)
					gl.Uniform1f(rnd.attribBotScanline, float32(rnd.img.screen.crit.topScanline+rnd.img.screen.crit.scanlines)*vertScaling)

					// the coordinates of the last plot
					if rnd.img.screen.cropped {
						gl.Uniform1f(rnd.attribLastX, float32(rnd.img.screen.crit.lastX-television.HorizClksHBlank)*horizScaling)
					} else {
						gl.Uniform1f(rnd.attribLastX, float32(rnd.img.screen.crit.lastX)*horizScaling)
					}
					gl.Uniform1f(rnd.attribLastY, float32(rnd.img.screen.crit.lastY)*vertScaling)

					// set ShowScreenDraw if emulation is paused or a low frame
					// rate has been requested
					if rnd.img.paused || rnd.img.tv.GetReqFPS() < 3.0 {
						gl.Uniform1i(rnd.attribShowScreenDraw, 1)
					} else {
						gl.Uniform1i(rnd.attribShowScreenDraw, -1)
					}

					if rnd.img.screen.cropped {
						gl.Uniform1i(rnd.attribCropped, 1)
					} else {
						gl.Uniform1i(rnd.attribCropped, -1)
					}

					// animation time
					anim := math.Sin(float64(time.Now().Nanosecond()) / 1000000000.0)
					anim = math.Abs(anim)
					gl.Uniform1f(rnd.attribAnimTime, float32(anim))

					// end of critical section
					rnd.img.screen.crit.section.Unlock()

				default:
					gl.Uniform1i(rnd.attribImageType, 0)

				}

				// clipping
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				gl.BindTexture(gl.TEXTURE_2D, uint32(textureID))
				gl.DrawElements(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), unsafe.Pointer(indexBufferOffset))
			}
			indexBufferOffset += uintptr(cmd.ElementCount() * indexSize)
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)

	// Restore modified GL state
	gl.UseProgram(uint32(lastProgram))
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.BindSampler(0, uint32(lastSampler))
	gl.ActiveTexture(uint32(lastActiveTexture))
	gl.BindVertexArray(uint32(lastVertexArray))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(lastElementArrayBuffer))
	gl.BlendEquationSeparate(uint32(lastBlendEquationRgb), uint32(lastBlendEquationAlpha))
	gl.BlendFuncSeparate(uint32(lastBlendSrcRgb), uint32(lastBlendDstRgb), uint32(lastBlendSrcAlpha), uint32(lastBlendDstAlpha))
	if lastEnableBlend {
		gl.Enable(gl.BLEND)
	} else {
		gl.Disable(gl.BLEND)
	}
	if lastEnableCullFace {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	if lastEnableDepthTest {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	if lastEnableScissorTest {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.PolygonMode(gl.FRONT_AND_BACK, uint32(lastPolygonMode[0]))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.Scissor(lastScissorBox[0], lastScissorBox[1], lastScissorBox[2], lastScissorBox[3])
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
	rnd.attribPixelPerfect = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("PixelPerfect"+"\x00"))
	rnd.attribDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Dim"+"\x00"))
	rnd.attribCropDim = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("CropDim"+"\x00"))
	rnd.attribShowScreenDraw = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("ShowScreenDraw"+"\x00"))
	rnd.attribCropped = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Cropped"+"\x00"))
	rnd.attribLastX = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastX"+"\x00"))
	rnd.attribLastY = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("LastY"+"\x00"))
	rnd.attribHblank = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("Hblank"+"\x00"))
	rnd.attribTopScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("TopScanline"+"\x00"))
	rnd.attribBotScanline = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("BotScanline"+"\x00"))
	rnd.attribAnimTime = gl.GetUniformLocation(rnd.shaderHandle, gl.Str("AnimTime"+"\x00"))

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

	// Upload texture to graphics system
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GenTextures(1, &rnd.fontTextureID)
	gl.BindTexture(gl.TEXTURE_2D, rnd.fontTextureID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height),
		0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	// Store our identifier
	rnd.imguiIO.Fonts().SetTextureID(imgui.TextureID(rnd.fontTextureID))

	// Restore state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
}

// getShaderCompileError returns the most recent error generated
// by the shader compiler
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
