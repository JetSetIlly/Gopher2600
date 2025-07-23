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
	"fmt"
	"image"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/imgui-go/v5"
)

type gl32Texture struct {
	id     uint32
	typ    shaderType
	create bool
	config any
}

type gl32 struct {
	img *SdlImgui

	vboHandle      uint32
	elementsHandle uint32

	textures map[uint32]gl32Texture
	shaders  map[shaderType]shaderProgram

	scrsht *gl32Screenshot
	video  *gl32Video
}

func newRenderer(img *SdlImgui) renderer {
	rnd := &gl32{
		img:      img,
		textures: make(map[uint32]gl32Texture),
		shaders:  make(map[shaderType]shaderProgram),
		scrsht:   newGl32Screenshot(),
		video:    newGl32Video(),
	}
	return rnd
}

func (rnd *gl32) requires() requirement {
	return requiresOpenGL32
}

func (rnd *gl32) supportsCRT() bool {
	return true
}

func (rnd *gl32) start() error {
	err := gl.Init()
	if err != nil {
		return fmt.Errorf("glsl: %w", err)
	}

	// setup shaders
	rnd.shaders[shaderGUI] = newGUIShader()
	rnd.shaders[shaderColor] = newColorShader()
	rnd.shaders[shaderPlayscr] = newPlayscrShader(rnd.img)
	rnd.shaders[shaderBevel] = newBevelShader(rnd.img)
	rnd.shaders[shaderDbgScr] = newDbgScrShader(rnd.img)
	rnd.shaders[shaderDbgScrOverlay] = newDbgScrOverlayShader(rnd.img)

	// deferring font setup until later

	gl.GenBuffers(1, &rnd.vboHandle)
	gl.GenBuffers(1, &rnd.elementsHandle)

	// log GPU vendor information
	logger.Logf(logger.Allow, "glsl", "vendor: %s", gl.GoStr(gl.GetString(gl.VENDOR)))
	logger.Logf(logger.Allow, "glsl", "renderer: %s", gl.GoStr(gl.GetString(gl.RENDERER)))
	logger.Logf(logger.Allow, "glsl", "driver: %s", gl.GoStr(gl.GetString(gl.VERSION)))

	return nil
}

func (rnd *gl32) destroy() {
	if rnd.vboHandle != 0 {
		gl.DeleteBuffers(1, &rnd.vboHandle)
	}
	rnd.vboHandle = 0

	if rnd.elementsHandle != 0 {
		gl.DeleteBuffers(1, &rnd.elementsHandle)
	}
	rnd.elementsHandle = 0

	for i := range rnd.shaders {
		if rnd.shaders[i] != nil {
			rnd.shaders[i].destroy()
		}
	}

	for _, tex := range rnd.textures {
		gl.DeleteTextures(1, &tex.id)
	}

	clear(rnd.textures)
	clear(rnd.shaders)

	rnd.scrsht.destroy()
	rnd.video.destroy()
}

// preRender clears the framebuffer.
func (rnd *gl32) preRender() {
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// render translates the ImGui draw data to OpenGL3 commands.
func (rnd *gl32) render() {
	winw, winh := rnd.img.plt.windowSize()
	fbw, fbh := rnd.img.plt.framebufferSize()
	drawData := imgui.RenderedDrawData()

	err := rnd.video.start(
		unique.Filename("video", rnd.img.cache.VCS.Mem.Cart.ShortName),
		int(rnd.img.screen.lastFrameGenerated.Load()),
		int32(fbw), int32(fbh),
		float32(rnd.img.plt.mode.RefreshRate))
	if err != nil {
		logger.Log(logger.Allow, "gl32", err.Error())
	}

	defer rnd.scrsht.process(int32(fbw), int32(fbh))
	defer func() {
		if rnd.img.isPlaymode() {
			if rnd.video.isRecording() {
				rnd.video.process(int(rnd.img.screen.lastFrameGenerated.Load()), int32(fbw), int32(fbh))
			}
		}
	}()

	st := storeGLState()
	defer st.restoreGLState()

	// Avoid rendering when minimised, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	if (fbw <= 0) || (fbh <= 0) {
		return
	}
	drawData.ScaleClipRects(imgui.Vec2{
		X: fbw / winw,
		Y: fbh / winh,
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
	env.projMtx = [4][4]float32{
		{2.0 / winw, 0.0, 0.0, 0.0},
		{0.0, 2.0 / -winh, 0.0, 0.0},
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
				id := cmd.TextureID()
				env.textureID = uint32(id)

				// select shader program to use
				var shader shaderProgram

				if tex, ok := rnd.textures[env.textureID]; ok {
					shader = rnd.shaders[tex.typ]
					env.config = tex.config
				}

				if shader == nil {
					panic("no shader found for texture")
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
				gl.Viewport(0, 0, int32(fbw), int32(fbh))
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbh)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				// process
				env.draw()
			}
			indexBufferOffset += uintptr(cmd.ElementCount() * indexSize)
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)
}

func (rnd *gl32) screenshot(mode screenshotMode, finish chan screenshotResult) {
	rnd.scrsht.start(mode, finish)
}

func (rnd *gl32) isScreenshotting() bool {
	return !rnd.scrsht.finished()
}

func (rnd *gl32) record(enabled bool) {
	rnd.video.enabled = enabled
}

func (rnd *gl32) isRecording() bool {
	return rnd.video.isRecording()
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

func (rnd *gl32) addTexture(typ shaderType, linear bool, clamp bool, config any) texture {
	tex := gl32Texture{
		create: true,
		typ:    typ,
		config: config,
	}

	gl.GenTextures(1, &tex.id)

	// create 1x1 texture as a placeholder
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, 1, 1, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr([]uint8{0}))

	if linear {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	} else {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	}

	if clamp {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	}

	rnd.textures[tex.id] = tex

	return &tex
}

func (rnd *gl32) addFontTexture(fnts imgui.FontAtlas) texture {
	tex := rnd.addTexture(shaderGUI, true, false, nil)
	image := fnts.TextureDataAlpha8()

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.BindTexture(gl.TEXTURE_2D, tex.getID())
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height), 0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	return tex
}

func (tex *gl32Texture) getID() uint32 {
	return tex.id
}

func (tex *gl32Texture) markForCreation() {
	tex.create = true
}

func (tex *gl32Texture) clear() {
	gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (tex *gl32Texture) render(pixels *image.RGBA) {
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if tex.create {
		tex.create = false

		gl.BindTexture(gl.TEXTURE_2D, tex.id)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

	} else {
		gl.BindTexture(gl.TEXTURE_2D, tex.id)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}
}
