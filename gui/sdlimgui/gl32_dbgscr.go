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
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/display/shaders"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type dbgScrHelper struct {
	showCursor             int32 // uniform
	isCropped              int32 // uniform
	screenDim              int32 // uniform
	scalingX               int32 // uniform
	scalingY               int32 // uniform
	lastX                  int32 // uniform
	lastY                  int32 // uniform
	hblank                 int32 // uniform
	visibleTop             int32 // uniform
	visibleBottom          int32 // uniform
	magShow                int32 // uniform
	magXmin                int32 // uniform
	magXmax                int32 // uniform
	magYmin                int32 // uniform
	magYmax                int32 // uniform
	lastNewFrameAtScanline int32 // uniform
}

func (attr *dbgScrHelper) get(sh shader) {
	attr.isCropped = gl.GetUniformLocation(sh.handle, gl.Str("IsCropped"+"\x00"))
	attr.showCursor = gl.GetUniformLocation(sh.handle, gl.Str("ShowCursor"+"\x00"))
	attr.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	attr.scalingX = gl.GetUniformLocation(sh.handle, gl.Str("ScalingX"+"\x00"))
	attr.scalingY = gl.GetUniformLocation(sh.handle, gl.Str("ScalingY"+"\x00"))
	attr.lastX = gl.GetUniformLocation(sh.handle, gl.Str("LastX"+"\x00"))
	attr.lastY = gl.GetUniformLocation(sh.handle, gl.Str("LastY"+"\x00"))
	attr.hblank = gl.GetUniformLocation(sh.handle, gl.Str("Hblank"+"\x00"))
	attr.lastNewFrameAtScanline = gl.GetUniformLocation(sh.handle, gl.Str("LastNewFrameAtScanline"+"\x00"))
	attr.visibleTop = gl.GetUniformLocation(sh.handle, gl.Str("VisibleTop"+"\x00"))
	attr.visibleBottom = gl.GetUniformLocation(sh.handle, gl.Str("VisibleBottom"+"\x00"))
	attr.magShow = gl.GetUniformLocation(sh.handle, gl.Str("MagShow"+"\x00"))
	attr.magXmin = gl.GetUniformLocation(sh.handle, gl.Str("MagXmin"+"\x00"))
	attr.magXmax = gl.GetUniformLocation(sh.handle, gl.Str("MagXmax"+"\x00"))
	attr.magYmin = gl.GetUniformLocation(sh.handle, gl.Str("MagYmin"+"\x00"))
	attr.magYmax = gl.GetUniformLocation(sh.handle, gl.Str("MagYmax"+"\x00"))
}

func (attr *dbgScrHelper) set(img *SdlImgui) {
	// dimensions of screen
	width := img.wm.dbgScr.scaledWidth
	height := img.wm.dbgScr.scaledHeight

	// scaling of screen
	yscaling := img.wm.dbgScr.yscaling
	xscaling := img.wm.dbgScr.xscaling

	// critical section
	img.screen.crit.section.Lock()

	gl.Uniform1f(attr.scalingX, img.wm.dbgScr.xscaling)
	gl.Uniform1f(attr.scalingY, img.wm.dbgScr.yscaling)
	gl.Uniform2f(attr.screenDim, width, height)

	// cursor is the coordinates of the *most recent* pixel to be drawn
	cursorX := img.screen.crit.lastX
	cursorY := img.screen.crit.lastY

	// if crt preview is enabled then force cropping
	if img.wm.dbgScr.cropped || img.wm.dbgScr.crtPreview {
		gl.Uniform1f(attr.lastX, float32(cursorX-specification.ClksHBlank)*xscaling)
		gl.Uniform1i(attr.isCropped, boolToInt32(true))
	} else {
		gl.Uniform1f(attr.lastX, float32(cursorX)*xscaling)
		gl.Uniform1i(attr.isCropped, boolToInt32(false))
	}
	gl.Uniform1f(attr.lastY, float32(cursorY)*yscaling)

	// screen geometry
	gl.Uniform1f(attr.hblank, (specification.ClksHBlank)*xscaling)
	gl.Uniform1f(attr.visibleTop, float32(img.screen.crit.frameInfo.VisibleTop)*yscaling)
	gl.Uniform1f(attr.visibleBottom, float32(img.screen.crit.frameInfo.VisibleBottom)*yscaling)
	gl.Uniform1f(attr.lastNewFrameAtScanline, float32(img.screen.crit.frameInfo.TotalScanlines)*yscaling)

	// window magnification
	var magXmin, magYmin, magXmax, magYmax float32
	if img.wm.dbgScr.cropped || img.wm.dbgScr.crtPreview {
		magXmin = float32(img.wm.dbgScr.magnifyWindow.clip.Min.X-specification.ClksHBlank) * xscaling
		magYmin = float32(img.wm.dbgScr.magnifyWindow.clip.Min.Y-img.screen.crit.frameInfo.VisibleTop) * yscaling
		magXmax = float32(img.wm.dbgScr.magnifyWindow.clip.Max.X-specification.ClksHBlank) * xscaling
		magYmax = float32(img.wm.dbgScr.magnifyWindow.clip.Max.Y-img.screen.crit.frameInfo.VisibleTop) * yscaling
	} else {
		magXmin = float32(img.wm.dbgScr.magnifyWindow.clip.Min.X) * xscaling
		magYmin = float32(img.wm.dbgScr.magnifyWindow.clip.Min.Y) * yscaling
		magXmax = float32(img.wm.dbgScr.magnifyWindow.clip.Max.X) * xscaling
		magYmax = float32(img.wm.dbgScr.magnifyWindow.clip.Max.Y) * yscaling
	}
	gl.Uniform1i(attr.magShow, boolToInt32(img.wm.dbgScr.magnifyWindow.open))
	gl.Uniform1f(attr.magXmin, magXmin)
	gl.Uniform1f(attr.magYmin, magYmin)
	gl.Uniform1f(attr.magXmax, magXmax)
	gl.Uniform1f(attr.magYmax, magYmax)

	img.screen.crit.section.Unlock()
	// end of critical section

	// show cursor
	switch img.dbg.State() {
	case govern.Paused:
		gl.Uniform1i(attr.showCursor, 1)
	case govern.Running:
		gl.Uniform1i(attr.showCursor, 0)
	case govern.Stepping:
		gl.Uniform1i(attr.showCursor, 1)
	case govern.Rewinding:
		gl.Uniform1i(attr.showCursor, 1)
	}
}

type dbgScrShader struct {
	shader
	dbgScrHelper

	img *SdlImgui
	crt *crtSequencer
}

func newDbgScrShader(img *SdlImgui) shaderProgram {
	sh := &dbgScrShader{
		img: img,
	}
	sh.crt = newCRTSequencer(img)
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.DbgScrHelpersShader), string(shaders.DbgScrShader))
	sh.dbgScrHelper.get(sh.shader)

	return sh
}

func (sh *dbgScrShader) destroy() {
	sh.crt.destroy()
}

func (sh *dbgScrShader) setAttributes(env shaderEnvironment) {
	env.width = int32(sh.img.wm.dbgScr.scaledWidth)
	env.height = int32(sh.img.wm.dbgScr.scaledHeight)

	gl.Viewport(-int32(sh.img.wm.dbgScr.screenOrigin.X),
		-int32(sh.img.wm.dbgScr.screenOrigin.Y),
		env.width+int32(sh.img.wm.dbgScr.screenOrigin.X),
		env.height+int32(sh.img.wm.dbgScr.screenOrigin.Y),
	)
	gl.Scissor(-int32(sh.img.wm.dbgScr.screenOrigin.X),
		-int32(sh.img.wm.dbgScr.screenOrigin.Y),
		env.width+int32(sh.img.wm.dbgScr.screenOrigin.X),
		env.height+int32(sh.img.wm.dbgScr.screenOrigin.Y),
	)

	projection := env.projMtx
	env.projMtx = [4][4]float32{
		{2.0 / (sh.img.wm.dbgScr.scaledWidth + sh.img.wm.dbgScr.screenOrigin.X), 0.0, 0.0, 0.0},
		{0.0, -2.0 / (sh.img.wm.dbgScr.scaledHeight + sh.img.wm.dbgScr.screenOrigin.Y), 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	if sh.img.wm.dbgScr.crtPreview {
		// this is a bit weird but in the case of crtPreview we need to force
		// the use of the dbgscr's normalTexture. this is because for a period
		// of one frame the value of crtPreview and the texture being used in
		// the dbgscr's image button may not agree.
		//
		// consider the sequence and interaction of dbgscr with glsl:
		//
		// 1) crtPreview is checked to decide whether to show the normalTexture
		// 2) if it is false and elements is true then the elementsTexture is
		//          selected
		// 3) the crt checkbox is shown and clicked. the crtPreview value is
		//          changed on this frame
		// 4) for one frame therefore, it is possible to reach this point
		//          (crtPreview is true) but for the elementsTexture to have
		//          been chosen
		//
		// forcing the use of the normalTexture at this point seems the least
		// obtrusive solution. another solutions could be to defer otion
		// changes to the following frame but that would involve a manager of
		// some sort.
		//
		// altenatively, we could try a more structured method of attaching a
		// texture to an imgui.Image and packaging texture specific options
		// within that structure.
		//
		// both alternative solutions seem baroque for a single use case. maybe
		// something for the future.
		env.textureID = sh.img.wm.dbgScr.displayTexture.getID()

		prefs := newCrtSeqPrefs(sh.img.displayPrefs)
		prefs.Enabled = true
		prefs.Bevel = false

		env.textureID = sh.crt.process(env, true, sh.img.wm.dbgScr.numScanlines, specification.ClksVisible,
			sh.img.wm.dbgScr, prefs, specification.NormalRotation, false)
		env.projMtx = projection
	} else {
		// if crtPreview is disabled we still go through the crt process. we do
		// this for two reasons.
		//
		// 1) to scale the image to the correct size
		//
		// 2) the phosphor is kept up to date, which is important for the
		// moment the crtPreview is enabled.
		//
		// note that we specify integer scaling for the non-CRT preview image,
		// this is so that the overlay is aligned properly with the TV image

		prefs := newCrtSeqPrefs(sh.img.displayPrefs)
		prefs.Enabled = false

		env.textureID = sh.crt.process(env, true, sh.img.wm.dbgScr.numScanlines, specification.ClksVisible,
			sh.img.wm.dbgScr, prefs, specification.NormalRotation, false)
		env.projMtx = projection
	}

	sh.shader.setAttributes(env)
	sh.dbgScrHelper.set(sh.img)
}

type dbgScrOverlayShader struct {
	shader
	dbgScrHelper

	stripe     int32 // uniform
	stripeSize int32 // uniform
	stripeFade int32 // uniform

	img *SdlImgui
}

func newDbgScrOverlayShader(img *SdlImgui) shaderProgram {
	sh := &dbgScrOverlayShader{
		img: img,
	}

	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.DbgScrHelpersShader), string(shaders.DbgScrOverlayShader))
	sh.dbgScrHelper.get(sh.shader)

	sh.stripe = gl.GetUniformLocation(sh.handle, gl.Str("Stripe"+"\x00"))
	sh.stripeSize = gl.GetUniformLocation(sh.handle, gl.Str("Stripe_Size"+"\x00"))
	sh.stripeFade = gl.GetUniformLocation(sh.handle, gl.Str("Stripe_Fade"+"\x00"))

	return sh
}

func (sh *dbgScrOverlayShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)
	sh.dbgScrHelper.set(sh.img)
	sh.img.screen.crit.section.Lock()
	gl.Uniform1i(sh.stripe, boolToInt32(sh.img.screen.crit.overlayStriped))
	sh.img.screen.crit.section.Unlock()
	gl.Uniform1f(sh.stripeSize, 3.0)
	gl.Uniform1f(sh.stripeFade, 0.7)
}

type dbgScrMagnifyShader struct {
	img           *SdlImgui
	tvColorShader shaderProgram
}

func newDbgScrMagnifyShader(img *SdlImgui) shaderProgram {
	sh := &dbgScrMagnifyShader{
		img:           img,
		tvColorShader: newTVColorShader(),
	}
	return sh
}

func (sh *dbgScrMagnifyShader) destroy() {
	sh.tvColorShader.destroy()
}

func (sh *dbgScrMagnifyShader) setAttributes(env shaderEnvironment) {
	sh.tvColorShader.(*tvColorShader).setAttributesArgs(env, tvColorShaderPrefs{
		Brightness: sh.img.displayPrefs.Colour.Brightness.Get().(float64),
		Contrast:   sh.img.displayPrefs.Colour.Contrast.Get().(float64),
		Saturation: sh.img.displayPrefs.Colour.Saturation.Get().(float64),
		Hue:        sh.img.displayPrefs.Colour.Hue.Get().(float64),
	})
}
