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

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
)

const (
	guiShaderID int = iota
	colorShaderID
	dbgscrShaderID
	overlayShaderID
	playscrShaderID
	numShaders
)

type glsl struct {
	img   *SdlImgui
	fonts *glslFonts

	shaders [numShaders]shaderProgram

	vboHandle      uint32
	elementsHandle uint32
}

func newGlsl(img *SdlImgui) (*glsl, error) {
	err := gl.Init()
	if err != nil {
		return nil, fmt.Errorf("glsl: %v", err)
	}

	rnd := &glsl{img: img}

	rnd.setupShaders()

	// dererring font setup until later

	gl.GenBuffers(1, &rnd.vboHandle)
	gl.GenBuffers(1, &rnd.elementsHandle)

	return rnd, nil
}

func (rnd *glsl) setupShaders() {
	rnd.shaders[guiShaderID] = newGUIShader()
	rnd.shaders[colorShaderID] = newColorShader(false)
	rnd.shaders[dbgscrShaderID] = newDbgScrShader(rnd.img)
	rnd.shaders[overlayShaderID] = newOverlayShader(rnd.img)
	rnd.shaders[playscrShaderID] = newPlayscrShader(rnd.img)
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

	rnd.fonts.destroy()

	for i := range rnd.shaders {
		if rnd.shaders[i] != nil {
			rnd.shaders[i].destroy()
		}
	}
}

// preRender clears the framebuffer.
func (rnd *glsl) preRender() {
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
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

	// the environment used by the shader
	env := shaderEnvironment{}

	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.
	env.presentationProj = [4][4]float32{
		{2.0 / displayWidth, 0.0, 0.0, 0.0},
		{0.0, 2.0 / -displayHeight, 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	var vaoHandle uint32
	gl.GenVertexArrays(1, &vaoHandle)
	gl.BindVertexArray(vaoHandle)
	gl.BindBuffer(gl.ARRAY_BUFFER, rnd.vboHandle)

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
				// texture id
				env.srcTextureID = uint32(cmd.TextureID())

				// select shader program to use
				var shader shaderProgram

				switch env.srcTextureID {
				case rnd.img.wm.dbgScr.normalTexture:
					shader = rnd.shaders[dbgscrShaderID]
				case rnd.img.wm.dbgScr.elementsTexture:
					shader = rnd.shaders[dbgscrShaderID]
				case rnd.img.wm.dbgScr.overlayTexture:
					shader = rnd.shaders[overlayShaderID]
				case rnd.img.playScr.screenTexture:
					shader = rnd.shaders[playscrShaderID]
				case rnd.img.wm.selectROM.thmbTexture:
					shader = rnd.shaders[colorShaderID]
				case rnd.img.wm.timeline.thmbTexture:
					shader = rnd.shaders[colorShaderID]
				case rnd.img.wm.windows[winComparisonID].(*winComparison).cmpTexture:
					shader = rnd.shaders[colorShaderID]
				case rnd.img.wm.windows[winComparisonID].(*winComparison).diffTexture:
					shader = rnd.shaders[colorShaderID]
				case rnd.img.wm.windows[winBotID].(*winBot).obsTexture:
					shader = rnd.shaders[colorShaderID]
				case rnd.img.wm.windows[winBotID].(*winBot).mouseTexture:
					shader = rnd.shaders[colorShaderID]
				default:
					if rnd.img.wm.windows[winCDFStreamsID].(*winCDFStreams).isStreamTexture(env.srcTextureID) {
						shader = rnd.shaders[colorShaderID]
					} else {
						shader = rnd.shaders[guiShaderID]
					}
				}

				env.draw = func() {
					gl.DrawElementsWithOffset(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), indexBufferOffset)
				}

				// set attributes for the selected shader
				shader.setAttributes(env)

				// draw using the currently selected shader to the real framebuffer
				gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

				// viewport and scissors. these might have changed during
				// execution of the shader
				gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				// process
				env.draw()
			}
			indexBufferOffset += uintptr(cmd.ElementCount() * indexSize)
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)
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
