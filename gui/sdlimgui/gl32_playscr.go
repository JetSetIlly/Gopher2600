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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type playscrShader struct {
	img        *SdlImgui
	crt        *crtSequencer
	screenshot *gl32Screenshot
}

func newPlayscrShader(img *SdlImgui) shaderProgram {
	sh := &playscrShader{
		img:        img,
		crt:        newCRTSequencer(img),
		screenshot: newGl32Screenshot(img),
	}
	return sh
}

func (sh *playscrShader) destroy() {
	sh.crt.destroy()
	sh.screenshot.destroy()
}

func (sh *playscrShader) setAttributes(env shaderEnvironment) {
	if !sh.img.isPlaymode() {
		return
	}

	env.width = int32(sh.img.playScr.scaledWidth)
	env.height = int32(sh.img.playScr.scaledHeight)

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.imagePosMin.X),
		int32(-sh.img.playScr.imagePosMin.Y),
		env.width+(int32(sh.img.playScr.imagePosMin.X*2)),
		env.height+(int32(sh.img.playScr.imagePosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.imagePosMin.X),
		int32(-sh.img.playScr.imagePosMin.Y),
		env.width+(int32(sh.img.playScr.imagePosMin.X*2)),
		env.height+(int32(sh.img.playScr.imagePosMin.Y*2)),
	)

	sh.screenshot.process(env, sh.img.playScr)

	sh.crt.process(env, false, sh.img.playScr.visibleScanlines, specification.ClksVisible, sh.img.playScr, newCrtSeqPrefs(sh.img.displayPrefs), sh.img.screen.rotation.Load().(specification.Rotation), false)
}
