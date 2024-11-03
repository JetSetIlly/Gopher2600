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

//go:build gl21

package sdlimgui

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/logger"
)

type gl21Texture struct {
	id     uint32
	create bool
}

type gl21 struct {
	img      *SdlImgui
	textures map[uint32]gl21Texture
	font     gl21Texture
}

func newRenderer(img *SdlImgui) renderer {
	return &gl21{
		img:      img,
		textures: make(map[uint32]gl21Texture),
	}
}

func (rnd *gl21) requires() requirement {
	return requiresOpenGL21
}

func (rnd *gl21) supportsCRT() bool {
	return false
}

func (rnd *gl21) start() error {
	err := gl.Init()
	if err != nil {
		return fmt.Errorf("gl21: %w", err)
	}

	// log GPU vendor information
	logger.Logf(logger.Allow, "glsl", "vendor: %s", gl.GoStr(gl.GetString(gl.VENDOR)))
	logger.Logf(logger.Allow, "glsl", "renderer: %s", gl.GoStr(gl.GetString(gl.RENDERER)))
	logger.Logf(logger.Allow, "glsl", "driver: %s", gl.GoStr(gl.GetString(gl.VERSION)))

	return nil
}

func (rnd *gl21) destroy() {
	for _, tex := range rnd.textures {
		gl.DeleteTextures(1, &tex.id)
	}
	clear(rnd.textures)
}

func (rnd *gl21) preRender() {
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (rnd *gl21) render() {
	winw, winh := rnd.img.plt.windowSize()
	fbw, fbh := rnd.img.plt.framebufferSize()
	drawData := imgui.RenderedDrawData()

	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	if (fbw <= 0) || (fbh <= 0) {
		return
	}
	drawData.ScaleClipRects(imgui.Vec2{
		X: fbw / winw,
		Y: fbh / winh,
	})

	// Setup render state: alpha-blending enabled, no face culling, no depth testing, scissor enabled, vertex/texcoord/color pointers, polygon fill.
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	var lastPolygonMode [2]int32
	gl.GetIntegerv(gl.POLYGON_MODE, &lastPolygonMode[0])
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])
	var lastScissorBox [4]int32
	gl.GetIntegerv(gl.SCISSOR_BOX, &lastScissorBox[0])
	gl.PushAttrib(gl.ENABLE_BIT | gl.COLOR_BUFFER_BIT | gl.TRANSFORM_BIT)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.LIGHTING)
	gl.Disable(gl.COLOR_MATERIAL)
	gl.Enable(gl.SCISSOR_TEST)
	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	gl.EnableClientState(gl.COLOR_ARRAY)
	gl.Enable(gl.TEXTURE_2D)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// You may want this if using this code in an OpenGL 3+ context where shaders may be bound
	// gl.UseProgram(0)

	// Setup viewport, orthographic projection matrix
	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.
	gl.Viewport(0, 0, int32(fbw), int32(fbh))
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0, float64(winw), float64(winh), 0, -1, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	indexSize := imgui.IndexBufferLayout()

	drawType := gl.UNSIGNED_SHORT
	const bytesPerUint32 = 4
	if indexSize == bytesPerUint32 {
		drawType = gl.UNSIGNED_INT
	}

	// Render command lists
	for _, commandList := range drawData.CommandLists() {
		vertexBuffer, _ := commandList.VertexBuffer()
		indexBuffer, _ := commandList.IndexBuffer()
		indexBufferOffset := uintptr(indexBuffer)

		gl.VertexPointer(2, gl.FLOAT, int32(vertexSize), unsafe.Pointer(uintptr(vertexBuffer)+uintptr(vertexOffsetPos)))
		gl.TexCoordPointer(2, gl.FLOAT, int32(vertexSize), unsafe.Pointer(uintptr(vertexBuffer)+uintptr(vertexOffsetUv)))
		gl.ColorPointer(4, gl.UNSIGNED_BYTE, int32(vertexSize), unsafe.Pointer(uintptr(vertexBuffer)+uintptr(vertexOffsetCol)))

		for _, command := range commandList.Commands() {
			if command.HasUserCallback() {
				command.CallUserCallback(commandList)
			} else {
				clipRect := command.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbh)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))
				gl.BindTexture(gl.TEXTURE_2D, uint32(command.TextureID()))
				gl.DrawElementsWithOffset(gl.TRIANGLES, int32(command.ElementCount()), uint32(drawType), indexBufferOffset)
			}

			indexBufferOffset += uintptr(command.ElementCount() * indexSize)
		}
	}

	// Restore modified state
	gl.DisableClientState(gl.COLOR_ARRAY)
	gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)
	gl.DisableClientState(gl.VERTEX_ARRAY)
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.MatrixMode(gl.MODELVIEW)
	gl.PopMatrix()
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()
	gl.PopAttrib()
	gl.PolygonMode(gl.FRONT, uint32(lastPolygonMode[0]))
	gl.PolygonMode(gl.BACK, uint32(lastPolygonMode[1]))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.Scissor(lastScissorBox[0], lastScissorBox[1], lastScissorBox[2], lastScissorBox[3])
}

func (rnd *gl21) screenshot(mode screenshotMode, finish chan screenshotResult) {
	finish <- screenshotResult{
		err: fmt.Errorf("gl21: renderer does not support screenshotting"),
	}
}

func (rnd *gl21) addTexture(_ textureType, linear bool, clamp bool) texture {
	tex := gl21Texture{
		create: true,
	}

	gl.GenTextures(1, &tex.id)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
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

func (rnd *gl21) addFontTexture(fnts imgui.FontAtlas) texture {
	tex := rnd.addTexture(textureGUI, true, false)
	image := fnts.TextureDataRGBA32()

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.BindTexture(gl.TEXTURE_2D, tex.getID())
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(image.Width), int32(image.Height), 0, gl.RGBA, gl.UNSIGNED_BYTE, image.Pixels)

	return tex
}

func (tex *gl21Texture) getID() uint32 {
	return tex.id
}

func (tex *gl21Texture) markForCreation() {
	tex.create = true
}

func (tex *gl21Texture) clear() {
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, 1, 1, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr([]uint8{0}))
}

func (tex *gl21Texture) render(pixels *image.RGBA) {
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
