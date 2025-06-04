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

type effectsShader struct {
	shader

	screenDim    int32
	numScanlines int32
	numClocks    int32

	curve                int32
	curveAmount          int32
	roundedCorners       int32
	roundedCornersAmount int32

	scanlines          int32
	scanlinesIntensity int32
	mask               int32
	maskIntensity      int32

	rfInterference  int32
	rfNoiseLevel    int32
	rfGhostingLevel int32

	chromaticAberration int32
	shine               int32
	blackLevel          int32
	gamma               int32

	rotation   int32
	screenshot int32
	time       int32

	isScrsht isScreenshotting
}

// used by the effects shader to determine if a screenshot is taking place. if
// it is then specific effects settings are used with the aim of improving the
// screenshot image
type isScreenshotting interface {
	isScreenshotting() bool
}

func newEffectsShader(isScrsht isScreenshotting) shaderProgram {
	sh := &effectsShader{
		isScrsht: isScrsht,
	}

	sh.createProgram(string(shaders.StraightVertexShader), string(shaders.CRTEffectsFragShader))

	sh.screenDim = gl.GetUniformLocation(sh.handle, gl.Str("ScreenDim"+"\x00"))
	sh.numScanlines = gl.GetUniformLocation(sh.handle, gl.Str("NumScanlines"+"\x00"))
	sh.numClocks = gl.GetUniformLocation(sh.handle, gl.Str("NumClocks"+"\x00"))

	sh.curve = gl.GetUniformLocation(sh.handle, gl.Str("Curve"+"\x00"))
	sh.curveAmount = gl.GetUniformLocation(sh.handle, gl.Str("CurveAmount"+"\x00"))

	sh.roundedCorners = gl.GetUniformLocation(sh.handle, gl.Str("RoundedCorners"+"\x00"))
	sh.roundedCornersAmount = gl.GetUniformLocation(sh.handle, gl.Str("RoundedCornersAmount"+"\x00"))

	sh.scanlines = gl.GetUniformLocation(sh.handle, gl.Str("Scanlines"+"\x00"))
	sh.scanlinesIntensity = gl.GetUniformLocation(sh.handle, gl.Str("ScanlinesIntensity"+"\x00"))
	sh.mask = gl.GetUniformLocation(sh.handle, gl.Str("ShadowMask"+"\x00"))
	sh.maskIntensity = gl.GetUniformLocation(sh.handle, gl.Str("MaskIntensity"+"\x00"))

	sh.rfInterference = gl.GetUniformLocation(sh.handle, gl.Str("RFInterference"+"\x00"))
	sh.rfNoiseLevel = gl.GetUniformLocation(sh.handle, gl.Str("RFNoiseLevel"+"\x00"))
	sh.rfGhostingLevel = gl.GetUniformLocation(sh.handle, gl.Str("RFGhostingLevel"+"\x00"))

	sh.chromaticAberration = gl.GetUniformLocation(sh.handle, gl.Str("ChromaticAberration"+"\x00"))
	sh.shine = gl.GetUniformLocation(sh.handle, gl.Str("Shine"+"\x00"))
	sh.blackLevel = gl.GetUniformLocation(sh.handle, gl.Str("BlackLevel"+"\x00"))
	sh.gamma = gl.GetUniformLocation(sh.handle, gl.Str("Gamma"+"\x00"))

	sh.rotation = gl.GetUniformLocation(sh.handle, gl.Str("Rotation"+"\x00"))
	sh.screenshot = gl.GetUniformLocation(sh.handle, gl.Str("Screenshot"+"\x00"))
	sh.time = gl.GetUniformLocation(sh.handle, gl.Str("Time"+"\x00"))

	return sh
}

// most shader attributes can be discerened automatically but number of
// scanlines, clocks and whether to add noise to the image is context sensitive.
func (sh *effectsShader) setAttributesArgs(env shaderEnvironment, numScanlines int, numClocks int,
	prefs crtSeqPrefs, rotation specification.Rotation,
	screenshot bool) {

	sh.shader.setAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.width), float32(env.height))
	gl.Uniform1i(sh.numScanlines, int32(numScanlines))
	gl.Uniform1i(sh.numClocks, int32(numClocks))

	gl.Uniform1i(sh.curve, boolToInt32(prefs.curve))
	gl.Uniform1f(sh.curveAmount, float32(prefs.curveAmount))
	gl.Uniform1i(sh.roundedCorners, boolToInt32(prefs.roundedCorners))
	gl.Uniform1f(sh.roundedCornersAmount, float32(prefs.roundedCornersAmount))

	gl.Uniform1i(sh.scanlines, boolToInt32(prefs.scanlines))
	gl.Uniform1f(sh.scanlinesIntensity, float32(prefs.scanlinesIntensity))
	gl.Uniform1i(sh.mask, boolToInt32(prefs.mask))
	gl.Uniform1f(sh.maskIntensity, float32(prefs.maskIntensity))

	gl.Uniform1i(sh.rfInterference, boolToInt32(prefs.rfInterference))
	gl.Uniform1f(sh.rfNoiseLevel, float32(prefs.rfNoiseLevel))
	gl.Uniform1f(sh.rfGhostingLevel, float32(prefs.rfGhostingLevel))

	gl.Uniform1f(sh.chromaticAberration, float32(prefs.chromaticAberration))
	gl.Uniform1i(sh.shine, boolToInt32(prefs.shine))
	gl.Uniform1f(sh.blackLevel, float32(prefs.blackLevel))
	gl.Uniform1f(sh.gamma, float32(prefs.gamma))

	gl.Uniform1i(sh.rotation, int32(rotation))
	gl.Uniform1i(sh.screenshot, boolToInt32(screenshot))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)

	// no noise when a screenshot is taking place
	if sh.isScrsht.isScreenshotting() {
		gl.Uniform1f(sh.rfNoiseLevel, float32(0))
	}
}
