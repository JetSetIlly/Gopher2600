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
)

type bevelShader struct {
	img  *SdlImgui
	dust shaderProgram
}

func newBevelShader(img *SdlImgui) shaderProgram {
	sh := &bevelShader{
		img:  img,
		dust: newDustShader(),
	}
	return sh
}

func (sh *bevelShader) destroy() {
	sh.dust.destroy()
}

func (sh *bevelShader) setAttributes(env shaderEnvironment) {
	if !sh.img.isPlaymode() {
		return
	}

	if !sh.img.playScr.usingBevel {
		return
	}

	env.width = int32(sh.img.playScr.bevelPosMax.X - sh.img.playScr.bevelPosMin.X)
	env.height = int32(sh.img.playScr.bevelPosMax.Y - sh.img.playScr.bevelPosMin.Y)

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.bevelPosMin.X),
		int32(-sh.img.playScr.bevelPosMin.Y),
		env.width+(int32(sh.img.playScr.bevelPosMin.X*2)),
		env.height+(int32(sh.img.playScr.bevelPosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.bevelPosMin.X),
		int32(-sh.img.playScr.bevelPosMin.Y),
		env.width+(int32(sh.img.playScr.bevelPosMin.X*2)),
		env.height+(int32(sh.img.playScr.bevelPosMin.Y*2)),
	)

	sh.dust.setAttributes(env)
}
