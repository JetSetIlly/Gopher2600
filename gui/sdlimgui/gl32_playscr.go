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
	"github.com/jetsetilly/gopher2600/gui/display/bevels"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type playscrShader struct {
	img *SdlImgui
	crt *crtSequencer
}

func newPlayscrShader(img *SdlImgui) shaderProgram {
	sh := &playscrShader{
		img: img,
		crt: newCRTSequencer(img),
	}
	return sh
}

func (sh *playscrShader) destroy() {
	sh.crt.destroy()
}

func (sh *playscrShader) setAttributes(env shaderEnvironment) {
	if !sh.img.isPlaymode() {
		return
	}

	env.width = int32(sh.img.playScr.screenWidth)
	env.height = int32(sh.img.playScr.screenHeight)

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.screenPosMin.X),
		int32(-sh.img.playScr.screenPosMin.Y),
		env.width+(int32(sh.img.playScr.screenPosMin.X*2)),
		env.height+(int32(sh.img.playScr.screenPosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.screenPosMin.X),
		int32(-sh.img.playScr.screenPosMin.Y),
		env.width+(int32(sh.img.playScr.screenPosMin.X*2)),
		env.height+(int32(sh.img.playScr.screenPosMin.Y*2)),
	)

	env.textureID = sh.crt.process(env, sh.img.playScr.screenTexture.getID(),
		sh.img.playScr.visibleScanlines, specification.ClksVisible,
		newCrtSeqPrefs(sh.img.crt), sh.img.screen.rotation.Load().(specification.Rotation), false)

	if sh.img.playScr.usingBevel {
		env.flipY = true

		f := sh.img.playScr.bevelHeight / sh.img.playScr.screenWidth
		f *= bevels.SolidState.Scale

		ww, wh := sh.img.plt.windowSize()
		wwr := ww / sh.img.playScr.bevelWidth
		whr := wh / sh.img.playScr.bevelHeight

		env.projMtx[0][0] *= f
		env.projMtx[1][1] *= f
		env.projMtx[3][0] = -f + (bevels.SolidState.OffsetX / wwr)
		env.projMtx[3][1] = f + (bevels.SolidState.OffsetY / whr)

		sh.crt.colorShader.setAttributes(env)
	}
}
