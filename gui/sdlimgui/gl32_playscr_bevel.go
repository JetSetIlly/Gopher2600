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
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui/display/shaders"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shading"
)

type bevelShader struct {
	shading.Base
	img                 *SdlImgui
	time                int32 // uniform
	rim                 int32 // uniform
	screen              int32 // uniform
	ambientTint         int32 // uniform
	ambientTintStrength int32 // uniform
}

func newBevelShader(img *SdlImgui) shading.Program {
	sh := &bevelShader{
		img: img,
	}
	sh.CreateProgram(string(shaders.CRTBevel))
	sh.time = sh.GetUniformLocation("Time")
	sh.rim = sh.GetUniformLocation("Rim")
	sh.screen = sh.GetUniformLocation("Screen")
	sh.ambientTint = sh.GetUniformLocation("AmbientTint")
	sh.ambientTintStrength = sh.GetUniformLocation("AmbientTintStrength")
	return sh
}

func (sh *bevelShader) setAttributes(env shading.Environment) {
	if !sh.img.isPlaymode() {
		return
	}

	if !sh.img.playScr.usingBevel {
		return
	}

	env.Width = int32(sh.img.playScr.bevelPosMax.X - sh.img.playScr.bevelPosMin.X)
	env.Height = int32(sh.img.playScr.bevelPosMax.Y - sh.img.playScr.bevelPosMin.Y)

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.bevelPosMin.X),
		int32(-sh.img.playScr.bevelPosMin.Y),
		env.Width+(int32(sh.img.playScr.bevelPosMin.X*2)),
		env.Height+(int32(sh.img.playScr.bevelPosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.bevelPosMin.X),
		int32(-sh.img.playScr.bevelPosMin.Y),
		env.Width+(int32(sh.img.playScr.bevelPosMin.X*2)),
		env.Height+(int32(sh.img.playScr.bevelPosMin.Y*2)),
	)

	sh.Base.SetAttributes(env)

	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
	if rim, ok := env.Config.(bool); ok {
		gl.Uniform1i(sh.rim, shading.BoolToInt32(rim))
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, sh.img.playScr.screenTexture.getID())
		gl.Uniform1i(sh.screen, 1)
	} else {
		gl.Uniform1i(sh.rim, shading.BoolToInt32(false))
	}

	gl.Uniform1i(sh.ambientTint, shading.BoolToInt32(sh.img.crt.ambientTint.Get().(bool)))
	gl.Uniform1f(sh.ambientTintStrength, float32(sh.img.crt.ambientTintStrength.Get().(float64)))
}
