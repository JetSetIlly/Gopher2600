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
	"github.com/jetsetilly/gopher2600/gui/display/shaders"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/shading"
)

type colorShader struct {
	shading.Base
}

func newColorShader() shading.Program {
	sh := &colorShader{}
	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.ColorShader))
	return sh
}

type phosphorShader struct {
	shading.Base
	newFrame int32
	latency  int32
}

func newPhosphorShader() shading.Program {
	sh := &phosphorShader{}
	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.CRTPhosphorFragShader))
	sh.newFrame = sh.GetUniformLocation("NewFrame")
	sh.latency = sh.GetUniformLocation("Latency")
	return sh
}

func (sh *phosphorShader) process(env shading.Environment, latency float32, newFrame uint32) {
	sh.Base.SetAttributes(env)
	gl.Uniform1f(sh.latency, latency)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, newFrame)
	gl.Uniform1i(sh.newFrame, 1)
}

type blurShader struct {
	shading.Base
	blur int32
}

func newBlurShader() shading.Program {
	sh := &blurShader{}
	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.CRTBlurFragShader))
	sh.blur = sh.GetUniformLocation("Blur")
	return sh
}

func (sh *blurShader) process(env shading.Environment, blur float32) {
	sh.Base.SetAttributes(env)

	// normalise blur amount depending on screen size
	blur *= float32(env.Height) / 960.0
	gl.Uniform2f(sh.blur, blur/float32(env.Width), blur/float32(env.Height))
}

type sharpenShader struct {
	shading.Base
	sharpness int32
}

func newSharpenShader() shading.Program {
	sh := &sharpenShader{}
	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.SharpenShader))
	sh.sharpness = sh.GetUniformLocation("Sharpness")
	return sh
}

func (sh *sharpenShader) process(env shading.Environment, sharpness float32) {
	sh.Base.SetAttributes(env)
	gl.Uniform1f(sh.sharpness, sharpness)
}

type guiShader struct {
	shading.Base
}

func newGUIShader() shading.Program {
	sh := &guiShader{}
	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.GUIShader))
	return sh
}
