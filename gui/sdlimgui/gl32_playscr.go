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
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shading"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type playscrShader struct {
	img *SdlImgui
	crt *crtSequencer
}

func newPlayscrShader(img *SdlImgui) shading.Program {
	sh := &playscrShader{
		img: img,
		crt: newCRTSequencer(img),
	}
	return sh
}

func (sh *playscrShader) Destroy() {
	sh.crt.destroy()
}

func (sh *playscrShader) SetAttributes(env shading.Environment) {
	if !sh.img.isPlaymode() {
		return
	}

	env.Width = int32(sh.img.playScr.screenWidth)
	env.Height = int32(sh.img.playScr.screenHeight)

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.screenPosMin.X),
		int32(-sh.img.playScr.screenPosMin.Y),
		env.Width+(int32(sh.img.playScr.screenPosMin.X*2)),
		env.Height+(int32(sh.img.playScr.screenPosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.screenPosMin.X),
		int32(-sh.img.playScr.screenPosMin.Y),
		env.Width+(int32(sh.img.playScr.screenPosMin.X*2)),
		env.Height+(int32(sh.img.playScr.screenPosMin.Y*2)),
	)

	env.TextureID = sh.crt.process(env, sh.img.playScr.screenTexture.getID(),
		sh.img.playScr.visibleScanlines, specification.ClksVisible,
		newCrtSeqPrefs(sh.img.crt), sh.img.screen.rotation.Load().(specification.Rotation), false)

	if sh.img.playScr.usingBevel {
		env.FlipY = true

		f := sh.img.playScr.bevelHeight / sh.img.playScr.screenWidth
		f *= bevels.SolidState.Scale

		ww, wh := sh.img.plt.windowSize()
		wwr := ww / sh.img.playScr.bevelWidth
		whr := wh / sh.img.playScr.bevelHeight

		env.ProjMtx[0][0] *= f
		env.ProjMtx[1][1] *= f
		env.ProjMtx[3][0] = -f + (bevels.SolidState.OffsetX / wwr)
		env.ProjMtx[3][1] = f + (bevels.SolidState.OffsetY / whr)

		sh.crt.colorShader.SetAttributes(env)
	}
}
