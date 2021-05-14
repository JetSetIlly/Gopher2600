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
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shaders"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type dbgScreenShader struct {
	shader

	img *SdlImgui
	crt *crtSequencer

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

func newDbgScrShader(img *SdlImgui) shaderProgram {
	sh := &dbgScreenShader{
		img: img,
	}

	sh.crt = newCRTSequencer(img)

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
	sh.crt.destroy()
}

func (sh *dbgScreenShader) setAttributes(env shaderEnvironment) {
	sh.img.screen.crit.section.Lock()
	width := sh.img.wm.dbgScr.scaledWidth(sh.img.wm.dbgScr.cropped)
	height := sh.img.wm.dbgScr.scaledHeight(sh.img.wm.dbgScr.cropped)
	numScanlines := sh.img.screen.crit.bottomScanline - sh.img.screen.crit.topScanline
	sh.img.screen.crit.section.Unlock()

	env.width = int32(width)
	env.height = int32(height)

	ox := int32(sh.img.wm.dbgScr.screenOrigin.X)
	oy := int32(sh.img.wm.dbgScr.screenOrigin.Y)
	gl.Viewport(-ox, -oy, env.width+ox, env.height+oy)
	gl.Scissor(-ox, -oy, env.width+ox, env.height+oy)

	env.internalProj = [4][4]float32{
		{2.0 / (width + float32(ox)), 0.0, 0.0, 0.0},
		{0.0, 2.0 / -(height + float32(oy)), 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}

	numClocks := specification.ClksScanline
	if sh.img.wm.dbgScr.cropped {
		numClocks = specification.ClksVisible
	}

	env.srcTextureID = sh.crt.process(env, sh.img.wm.dbgScr.crt, true, numScanlines, numClocks)

	sh.shader.setAttributes(env)

	// scaling of screen
	var vertScaling float32
	var horizScaling float32
	if sh.img.isPlaymode() {
		vertScaling = sh.img.playScr.scaling
		horizScaling = sh.img.playScr.horizScaling()
	} else {
		vertScaling = sh.img.wm.dbgScr.scaling
		horizScaling = sh.img.wm.dbgScr.horizScaling()
	}

	// critical section
	sh.img.screen.crit.section.Lock()

	gl.Uniform1f(sh.scalingX, sh.img.wm.dbgScr.horizScaling())
	gl.Uniform1f(sh.scalingY, sh.img.wm.dbgScr.scaling)
	gl.Uniform2f(sh.uncroppedScreenDim, sh.img.wm.dbgScr.scaledWidth(false), sh.img.wm.dbgScr.scaledHeight(false))
	gl.Uniform2f(sh.screenDim, sh.img.wm.dbgScr.scaledWidth(true), sh.img.wm.dbgScr.scaledHeight(true))
	if sh.img.wm.dbgScr.cropped {
		gl.Uniform1i(sh.isCropped, 1)
	} else {
		gl.Uniform1i(sh.isCropped, 0)
	}

	cursorX := sh.img.screen.crit.lastX
	cursorY := sh.img.screen.crit.lastY

	if sh.img.wm.dbgScr.cropped {
		gl.Uniform1f(sh.lastX, float32(cursorX-specification.ClksHBlank)*horizScaling)
	} else {
		gl.Uniform1f(sh.lastX, float32(cursorX)*horizScaling)
	}
	gl.Uniform1f(sh.lastY, float32(cursorY)*vertScaling)

	// screen geometry
	gl.Uniform1f(sh.hblank, specification.ClksHBlank*horizScaling)
	gl.Uniform1f(sh.topScanline, float32(sh.img.screen.crit.topScanline)*vertScaling)
	gl.Uniform1f(sh.botScanline, float32(sh.img.screen.crit.bottomScanline)*vertScaling)

	sh.img.screen.crit.section.Unlock()
	// end of critical section

	// show cursor
	if sh.img.isRewindSlider {
		gl.Uniform1i(sh.showCursor, 0)
	} else {
		switch sh.img.state {
		case gui.StatePaused:
			gl.Uniform1i(sh.showCursor, 1)
		case gui.StateRunning:
			// if FPS is low enough then show screen draw even though
			// emulation is running
			if sh.img.lz.TV.ReqFPS < television.ThreshVisual {
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
	img   *SdlImgui
	alpha int32 // uniform
}

func newOverlayShader(img *SdlImgui) shaderProgram {
	sh := &overlayShader{
		img: img,
	}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.OverlayShader))
	sh.alpha = gl.GetUniformLocation(sh.handle, gl.Str("Alpha"+"\x00"))
	return sh
}

func (sh *overlayShader) setAttributes(env shaderEnvironment) {
	sh.shader.setAttributes(env)
	gl.Uniform1f(sh.alpha, sh.img.wm.dbgScr.overlayAlpha)
}

type guiShader struct {
	shader
}

func newGUIShader() shaderProgram {
	sh := &guiShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.GUIShader))
	return sh
}
