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
}

func (attr *dbgScrHelper) set(img *SdlImgui, view *winDbgScrView) {
	// critical section
	img.screen.crit.section.Lock()

	gl.Uniform1f(attr.scalingX, view.xscaling)
	gl.Uniform1f(attr.scalingY, view.yscaling)
	gl.Uniform2f(attr.screenDim, view.scaledWidth, view.scaledHeight)

	// cursor is the coordinates of the *most recent* pixel to be drawn
	cursorX := img.screen.crit.lastX
	cursorY := img.screen.crit.lastY

	// if crt preview is enabled then force cropping
	if view.cropped {
		gl.Uniform1f(attr.lastX, float32(cursorX-specification.ClksHBlank)*view.xscaling)
		gl.Uniform1i(attr.isCropped, shading.BoolToInt32(true))
	} else {
		gl.Uniform1f(attr.lastX, float32(cursorX)*view.xscaling)
		gl.Uniform1i(attr.isCropped, shading.BoolToInt32(false))
	}
	gl.Uniform1f(attr.lastY, float32(cursorY)*view.yscaling)

	// screen geometry
	gl.Uniform1f(attr.hblank, (specification.ClksHBlank)*view.xscaling)
	gl.Uniform1f(attr.visibleTop, float32(img.screen.crit.frameInfo.VisibleTop)*view.yscaling)
	gl.Uniform1f(attr.visibleBottom, float32(img.screen.crit.frameInfo.VisibleBottom)*view.yscaling)
	gl.Uniform1f(attr.totalScanlines, float32(img.screen.crit.frameInfo.TotalScanlines)*view.yscaling)
	gl.Uniform1f(attr.topScanline, float32(img.screen.crit.frameInfo.TopScanline)*view.yscaling)

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
	sh.Base.CreateProgram(string(shaders.DbgScrHelpersShader), string(shaders.DbgScrShader))
	sh.dbgScrHelper.get(sh.Base)

	return sh
}

func (sh *dbgScrShader) Destroy() {
	sh.sequence.Destroy()
	sh.sharpen.Destroy()
}

func (sh *dbgScrShader) SetAttributes(env shading.Environment) {
	view := env.Config.(*winDbgScrView)

	env.Width = int32(view.scaledWidth)
	env.Height = int32(view.scaledHeight)

	gl.Viewport(-int32(view.screenOrigin.X),
		-int32(view.screenOrigin.Y),
		env.Width+int32(view.screenOrigin.X),
		env.Height+int32(view.screenOrigin.Y),
	)
	gl.Scissor(-int32(view.screenOrigin.X),
		-int32(view.screenOrigin.Y),
		env.Width+int32(view.screenOrigin.X),
		env.Height+int32(view.screenOrigin.Y),
	)

	projMtx := env.ProjMtx
	env.ProjMtx = [4][4]float32{
		{2.0 / (view.scaledWidth + view.screenOrigin.X), 0.0, 0.0, 0.0},
		{0.0, -2.0 / (view.scaledHeight + view.screenOrigin.Y), 0.0, 0.0},
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
		sh.dbgScrHelper.set(sh.img, view)
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

	sh.CreateProgram(string(shaders.DbgScrHelpersShader), string(shaders.DbgScrOverlayShader))
	sh.dbgScrHelper.get(sh.Base)

	return sh
}

func (sh *dbgScrOverlayShader) SetAttributes(env shading.Environment) {
	view := env.Config.(*winDbgScrView)
	sh.Base.SetAttributes(env)
	sh.dbgScrHelper.set(sh.img, view)
}
