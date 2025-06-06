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
)

type bevelShader struct {
	shader
	img                 *SdlImgui
	time                int32 // uniform
	rim                 int32 // uniform
	screen              int32 // uniform
	ambientTint         int32 // uniform
	ambientTintStrength int32 // uniform
}

func newBevelShader(img *SdlImgui) shaderProgram {
	sh := &bevelShader{
		img: img,
	}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTBevel))
	sh.time = gl.GetUniformLocation(sh.handle, gl.Str("Time"+"\x00"))
	sh.rim = gl.GetUniformLocation(sh.handle, gl.Str("Rim"+"\x00"))
	sh.screen = gl.GetUniformLocation(sh.handle, gl.Str("Screen"+"\x00"))
	sh.ambientTint = gl.GetUniformLocation(sh.handle, gl.Str("AmbientTint"+"\x00"))
	sh.ambientTintStrength = gl.GetUniformLocation(sh.handle, gl.Str("AmbientTintStrength"+"\x00"))
	return sh
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

	sh.shader.setAttributes(env)

	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
	if rim, ok := env.config.(bool); ok {
		gl.Uniform1i(sh.rim, boolToInt32(rim))
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, sh.img.playScr.screenTexture.getID())
		gl.Uniform1i(sh.screen, 1)
	} else {
		gl.Uniform1i(sh.rim, boolToInt32(false))
	}

	gl.Uniform1i(sh.ambientTint, boolToInt32(sh.img.crt.ambientTint.Get().(bool)))
	gl.Uniform1f(sh.ambientTintStrength, float32(sh.img.crt.ambientTintStrength.Get().(float64)))
}
