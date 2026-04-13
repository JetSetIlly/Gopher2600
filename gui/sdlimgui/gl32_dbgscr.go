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
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shading"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type dbgScrHelper struct {
	showCursor     int32 // uniform
	isCropped      int32 // uniform
	screenDim      int32 // uniform
	scalingX       int32 // uniform
	scalingY       int32 // uniform
	lastX          int32 // uniform
	lastY          int32 // uniform
	hblank         int32 // uniform
	visibleTop     int32 // uniform
	visibleBottom  int32 // uniform
	magShow        int32 // uniform
	magXmin        int32 // uniform
	magXmax        int32 // uniform
	magYmin        int32 // uniform
	magYmax        int32 // uniform
	totalScanlines int32 // uniform
	topScanline    int32 // uniform
}

func (attr *dbgScrHelper) get(sh shading.Base) {
	attr.isCropped = sh.GetUniformLocation("IsCropped")
	attr.showCursor = sh.GetUniformLocation("ShowCursor")
	attr.screenDim = sh.GetUniformLocation("ScreenDim")
	attr.scalingX = sh.GetUniformLocation("ScalingX")
	attr.scalingY = sh.GetUniformLocation("ScalingY")
	attr.lastX = sh.GetUniformLocation("LastX")
	attr.lastY = sh.GetUniformLocation("LastY")
	attr.hblank = sh.GetUniformLocation("Hblank")
	attr.totalScanlines = sh.GetUniformLocation("TotalScanlines")
	attr.topScanline = sh.GetUniformLocation("TopScanline")
	attr.visibleTop = sh.GetUniformLocation("VisibleTop")
	attr.visibleBottom = sh.GetUniformLocation("VisibleBottom")
	attr.magShow = sh.GetUniformLocation("MagShow")
	attr.magXmin = sh.GetUniformLocation("MagXmin")
	attr.magXmax = sh.GetUniformLocation("MagXmax")
	attr.magYmin = sh.GetUniformLocation("MagYmin")
	attr.magYmax = sh.GetUniformLocation("MagYmax")
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
	if img.wm.dbgScr.cropped {
		gl.Uniform1f(attr.lastX, float32(cursorX-specification.ClksHBlank)*xscaling)
		gl.Uniform1i(attr.isCropped, shading.BoolToInt32(true))
	} else {
		gl.Uniform1f(attr.lastX, float32(cursorX)*xscaling)
		gl.Uniform1i(attr.isCropped, shading.BoolToInt32(false))
	}
	gl.Uniform1f(attr.lastY, float32(cursorY)*yscaling)

	// screen geometry
	gl.Uniform1f(attr.hblank, (specification.ClksHBlank)*xscaling)
	gl.Uniform1f(attr.visibleTop, float32(img.screen.crit.frameInfo.VisibleTop)*yscaling)
	gl.Uniform1f(attr.visibleBottom, float32(img.screen.crit.frameInfo.VisibleBottom)*yscaling)
	gl.Uniform1f(attr.totalScanlines, float32(img.screen.crit.frameInfo.TotalScanlines)*yscaling)
	gl.Uniform1f(attr.topScanline, float32(img.screen.crit.frameInfo.TopScanline)*yscaling)

	// window magnification
	var magXmin, magYmin, magXmax, magYmax float32
	if img.wm.dbgScr.cropped {
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
	gl.Uniform1i(attr.magShow, shading.BoolToInt32(img.wm.dbgScr.magnifyWindow.open))
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
	shading.Base
	dbgScrHelper

	img *SdlImgui

	sequence *framebuffer.Flip
	sharpen  shading.Program
}

func newDbgScrShader(img *SdlImgui) shading.Program {
	sh := &dbgScrShader{
		img:      img,
		sequence: framebuffer.NewFlip(true),
		sharpen:  newSharpenShader(),
	}
	sh.Base.CreateProgram(string(shaders.StraightVertexShader), string(shaders.DbgScrHelpersShader), string(shaders.DbgScrShader))
	sh.dbgScrHelper.get(sh.Base)

	return sh
}

func (sh *dbgScrShader) Destroy() {
	sh.sequence.Destroy()
	sh.sharpen.Destroy()
}

func (sh *dbgScrShader) SetAttributes(env shading.Environment) {
	env.Width = int32(sh.img.wm.dbgScr.scaledWidth)
	env.Height = int32(sh.img.wm.dbgScr.scaledHeight)

	if sh.img.wm.dbgScr.elements {
		env.TextureID = sh.img.wm.dbgScr.elementsTexture.getID()
	} else {
		env.TextureID = sh.img.wm.dbgScr.displayTexture.getID()
	}

	gl.Viewport(-int32(sh.img.wm.dbgScr.screenOrigin.X),
		-int32(sh.img.wm.dbgScr.screenOrigin.Y),
		env.Width+int32(sh.img.wm.dbgScr.screenOrigin.X),
		env.Height+int32(sh.img.wm.dbgScr.screenOrigin.Y),
	)
	gl.Scissor(-int32(sh.img.wm.dbgScr.screenOrigin.X),
		-int32(sh.img.wm.dbgScr.screenOrigin.Y),
		env.Width+int32(sh.img.wm.dbgScr.screenOrigin.X),
		env.Height+int32(sh.img.wm.dbgScr.screenOrigin.Y),
	)

	projMtx := env.ProjMtx
	env.ProjMtx = [4][4]float32{
		{2.0 / (sh.img.wm.dbgScr.scaledWidth + sh.img.wm.dbgScr.screenOrigin.X), 0.0, 0.0, 0.0},
		{0.0, -2.0 / (sh.img.wm.dbgScr.scaledHeight + sh.img.wm.dbgScr.screenOrigin.Y), 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	env.FlipY = true
	sh.sequence.Setup(env.Width, env.Height)
	env.TextureID = sh.sequence.Process(func() {
		sh.sharpen.(*sharpenShader).process(env, 2)
		env.Draw()
	})

	env.TextureID = sh.sequence.Process(func() {
		sh.Base.SetAttributes(env)
		sh.dbgScrHelper.set(sh.img)
		env.Draw()
	})
	env.FlipY = false

	env.ProjMtx = projMtx
	sh.sharpen.(*sharpenShader).process(env, 2)
	sh.sharpen.SetAttributes(env)
}

type dbgScrOverlayShader struct {
	shading.Base
	dbgScrHelper
	img *SdlImgui
}

func newDbgScrOverlayShader(img *SdlImgui) shading.Program {
	sh := &dbgScrOverlayShader{
		img: img,
	}

	sh.CreateProgram(
		string(shaders.StraightVertexShader),
		string(shaders.DbgScrHelpersShader),
		string(shaders.DbgScrOverlayShader),
	)
	sh.dbgScrHelper.get(sh.Base)

	return sh
}

func (sh *dbgScrOverlayShader) SetAttributes(env shading.Environment) {
	sh.Base.SetAttributes(env)
	sh.dbgScrHelper.set(sh.img)
}
