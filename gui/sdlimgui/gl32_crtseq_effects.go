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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type effectsShader struct {
	shading.Base

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

func newEffectsShader(isScrsht isScreenshotting) shading.Program {
	sh := &effectsShader{
		isScrsht: isScrsht,
	}

	sh.CreateProgram(string(shaders.StraightVertexShader), string(shaders.CRTEffectsFragShader))

	sh.screenDim = sh.GetUniformLocation("ScreenDim")
	sh.numScanlines = sh.GetUniformLocation("NumScanlines")
	sh.numClocks = sh.GetUniformLocation("NumClocks")

	sh.curve = sh.GetUniformLocation("Curve")
	sh.curveAmount = sh.GetUniformLocation("CurveAmount")

	sh.roundedCorners = sh.GetUniformLocation("RoundedCorners")
	sh.roundedCornersAmount = sh.GetUniformLocation("RoundedCornersAmount")

	sh.scanlines = sh.GetUniformLocation("Scanlines")
	sh.scanlinesIntensity = sh.GetUniformLocation("ScanlinesIntensity")
	sh.mask = sh.GetUniformLocation("ShadowMask")
	sh.maskIntensity = sh.GetUniformLocation("MaskIntensity")

	sh.rfInterference = sh.GetUniformLocation("RFInterference")
	sh.rfNoiseLevel = sh.GetUniformLocation("RFNoiseLevel")
	sh.rfGhostingLevel = sh.GetUniformLocation("RFGhostingLevel")

	sh.chromaticAberration = sh.GetUniformLocation("ChromaticAberration")
	sh.shine = sh.GetUniformLocation("Shine")
	sh.blackLevel = sh.GetUniformLocation("BlackLevel")
	sh.gamma = sh.GetUniformLocation("Gamma")

	sh.rotation = sh.GetUniformLocation("Rotation")
	sh.screenshot = sh.GetUniformLocation("Screenshot")
	sh.time = sh.GetUniformLocation("Time")

	return sh
}

// most shader attributes can be discerened automatically but number of
// scanlines, clocks and whether to add noise to the image is context sensitive.
func (sh *effectsShader) setAttributesArgs(env shading.Environment, numScanlines int, numClocks int,
	prefs crtSeqPrefs, rotation specification.Rotation,
	screenshot bool) {

	sh.Base.SetAttributes(env)

	gl.Uniform2f(sh.screenDim, float32(env.Width), float32(env.Height))
	gl.Uniform1i(sh.numScanlines, int32(numScanlines))
	gl.Uniform1i(sh.numClocks, int32(numClocks))

	gl.Uniform1i(sh.curve, shading.BoolToInt32(prefs.curve))
	gl.Uniform1f(sh.curveAmount, float32(prefs.curveAmount))
	gl.Uniform1i(sh.roundedCorners, shading.BoolToInt32(prefs.roundedCorners))
	gl.Uniform1f(sh.roundedCornersAmount, float32(prefs.roundedCornersAmount))

	gl.Uniform1i(sh.scanlines, shading.BoolToInt32(prefs.scanlines))
	gl.Uniform1f(sh.scanlinesIntensity, float32(prefs.scanlinesIntensity))
	gl.Uniform1i(sh.mask, shading.BoolToInt32(prefs.mask))
	gl.Uniform1f(sh.maskIntensity, float32(prefs.maskIntensity))

	gl.Uniform1i(sh.rfInterference, shading.BoolToInt32(prefs.rfInterference))
	gl.Uniform1f(sh.rfNoiseLevel, float32(prefs.rfNoiseLevel))
	gl.Uniform1f(sh.rfGhostingLevel, float32(prefs.rfGhostingLevel))

	gl.Uniform1f(sh.chromaticAberration, float32(prefs.chromaticAberration))
	gl.Uniform1i(sh.shine, shading.BoolToInt32(prefs.shine))
	gl.Uniform1f(sh.blackLevel, float32(prefs.blackLevel))
	gl.Uniform1f(sh.gamma, float32(prefs.gamma))

	gl.Uniform1i(sh.rotation, int32(rotation))
	gl.Uniform1i(sh.screenshot, shading.BoolToInt32(screenshot))
	gl.Uniform1f(sh.time, float32(time.Now().Nanosecond())/100000000.0)

	// no noise when a screenshot is taking place
	if sh.isScrsht.isScreenshotting() {
		gl.Uniform1f(sh.rfNoiseLevel, float32(0))
	}
}
