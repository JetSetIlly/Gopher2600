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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type crtSeqEffectsShader struct {
	shader

	screenDim            int32
	numScanlines         int32
	numClocks            int32
	curve                int32
	roundedCorners       int32
	bevel                int32
	shine                int32
	shadowMask           int32
	scanlines            int32
	interference         int32
	noise                int32
	flicker              int32
	fringing             int32
	blackLevel           int32
	curveAmount          int32
	roundedCornersAmount int32
	bevelSize            int32
	maskIntensity        int32
	scanlinesIntensity   int32
	interferenceLevel    int32
	noiseLevel           int32
	flickerLevel         int32
	fringingAmount       int32
	time                 int32
	rotation             int32
	screenshot           int32
}

func newCrtSeqEffectsShader() shaderProgram {
	sh := &crtSeqEffectsShader{}
	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTEffectsFragShader))

	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.numScanlines = gl.GetUniformLocation(sh.handle, gl.Str("NumScanlines"+"\x00"))
	sh.numClocks = gl.GetUniformLocation(sh.handle, gl.Str("NumClocks"+"\x00"))
	sh.curve = gl.GetUniformLocation(sh.handle, gl.Str("Curve"+"\x00"))
	sh.roundedCorners = gl.GetUniformLocation(sh.handle, gl.Str("RoundedCorners"+"\x00"))
	sh.bevel = gl.GetUniformLocation(sh.handle, gl.Str("Bevel"+"\x00"))
	sh.shine = gl.GetUniformLocation(sh.handle, gl.Str("Shine"+"\x00"))
	sh.shadowMask = gl.GetUniformLocation(sh.handle, gl.Str("ShadowMask"+"\x00"))
	sh.scanlines = gl.GetUniformLocation(sh.handle, gl.Str("Scanlines"+"\x00"))
	sh.interference = gl.GetUniformLocation(sh.handle, gl.Str("Interference"+"\x00"))
	sh.flicker = gl.GetUniformLocation(sh.handle, gl.Str("Flicker"+"\x00"))
	sh.fringing = gl.GetUniformLocation(sh.handle, gl.Str("Fringing"+"\x00"))
	sh.blackLevel = gl.GetUniformLocation(sh.handle, gl.Str("BlackLevel"+"\x00"))
	sh.curveAmount = gl.GetUniformLocation(sh.handle, gl.Str("CurveAmount"+"\x00"))
	sh.roundedCornersAmount = gl.GetUniformLocation(sh.handle, gl.Str("RoundedCornersAmount"+"\x00"))
	sh.bevelSize = gl.GetUniformLocation(sh.handle, gl.Str("BevelSize"+"\x00"))
	sh.maskIntensity = gl.GetUniformLocation(sh.handle, gl.Str("MaskIntensity"+"\x00"))
	sh.scanlinesIntensity = gl.GetUniformLocation(sh.handle, gl.Str("ScanlinesIntensity"+"\x00"))
	sh.interferenceLevel = gl.GetUniformLocation(sh.handle, gl.Str("InterferenceLevel"+"\x00"))
	sh.flickerLevel = gl.GetUniformLocation(sh.handle, gl.Str("FlickerLevel"+"\x00"))
	sh.fringingAmount = gl.GetUniformLocation(sh.handle, gl.Str("FringingAmount"+"\x00"))
	sh.time = gl.GetUniformLocation(sh.handle, gl.Str("Time"+"\x00"))
	sh.rotation = gl.GetUniformLocation(sh.handle, gl.Str("Rotation"+"\x00"))
	sh.screenshot = gl.GetUniformLocation(sh.handle, gl.Str("Screenshot"+"\x00"))

	return sh
}

// most shader attributes can be discerened automatically but number of
// scanlines, clocks and whether to add noise to the image is context sensitive.
func (sh *crtSeqEffectsShader) setAttributesArgs(env shaderEnvironment, numScanlines int, numClocks int,
	prefs crtSeqPrefs, rotation specification.Rotation,
	screenshot bool) {

	sh.shader.setAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1i(sh.numScanlines, int32(numScanlines))
	gl.Uniform1i(sh.numClocks, int32(numClocks))
	gl.Uniform1i(sh.curve, boolToInt32(prefs.Curve))
	gl.Uniform1i(sh.roundedCorners, boolToInt32(prefs.RoundedCorners))
	gl.Uniform1i(sh.bevel, boolToInt32(prefs.Bevel))
	gl.Uniform1i(sh.shine, boolToInt32(prefs.Shine))
	gl.Uniform1i(sh.shadowMask, boolToInt32(prefs.Mask))
	gl.Uniform1i(sh.scanlines, boolToInt32(prefs.Scanlines))
	gl.Uniform1i(sh.interference, boolToInt32(prefs.Interference))
	gl.Uniform1i(sh.flicker, boolToInt32(prefs.Flicker))
	gl.Uniform1i(sh.fringing, boolToInt32(prefs.Fringing))
	gl.Uniform1f(sh.blackLevel, float32(prefs.BlackLevel))
	gl.Uniform1f(sh.curveAmount, float32(prefs.CurveAmount))
	gl.Uniform1f(sh.roundedCornersAmount, float32(prefs.RoundedCornersAmount))
	gl.Uniform1f(sh.bevelSize, float32(prefs.BevelSize))
	gl.Uniform1f(sh.maskIntensity, float32(prefs.MaskIntensity))
	gl.Uniform1f(sh.scanlinesIntensity, float32(prefs.ScanlinesIntensity))
	gl.Uniform1f(sh.interferenceLevel, float32(prefs.InterferenceLevel))
	gl.Uniform1f(sh.flickerLevel, float32(prefs.FlickerLevel))
	gl.Uniform1f(sh.fringingAmount, float32(prefs.FringingAmount))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)
	gl.Uniform1i(sh.rotation, int32(rotation))
	gl.Uniform1i(sh.screenshot, boolToInt32(screenshot))
}
