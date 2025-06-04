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
	"github.com/jetsetilly/gopher2600/gui/display/bevels"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type crtSeqPrefs struct {
	pixelPerfect     bool
	pixelPerfectFade float64

	curve                bool
	curveAmount          float64
	roundedCorners       bool
	roundedCornersAmount float64
	scanlines            bool
	scanlinesIntensity   float64
	mask                 bool
	maskIntensity        float64
	rfInterference       bool
	rfNoiseLevel         float64
	rfGhostingLevel      float64
	phosphor             bool
	phosphorLatency      float64
	phosphorBloom        float64

	chromaticAberration float64
	sharpness           float64
	blackLevel          float64
	shine               bool
	gamma               float64
}

func newCrtSeqPrefs(crt *preferencesCRT) crtSeqPrefs {
	p := crtSeqPrefs{
		pixelPerfect:         crt.pixelPerfect.Get().(bool),
		pixelPerfectFade:     crt.pixelPerfectFade.Get().(float64),
		curve:                crt.curve.Get().(bool),
		curveAmount:          crt.curveAmount.Get().(float64),
		roundedCorners:       crt.roundedCorners.Get().(bool),
		roundedCornersAmount: crt.roundedCornersAmount.Get().(float64),
		scanlines:            crt.scanlines.Get().(bool),
		scanlinesIntensity:   crt.scanlinesIntensity.Get().(float64),
		mask:                 crt.mask.Get().(bool),
		maskIntensity:        crt.maskIntensity.Get().(float64),
		rfInterference:       crt.rfInterference.Get().(bool),
		rfNoiseLevel:         crt.rfNoiseLevel.Get().(float64),
		rfGhostingLevel:      crt.rfGhostingLevel.Get().(float64),
		phosphor:             crt.phosphor.Get().(bool),
		phosphorLatency:      crt.phosphorLatency.Get().(float64),
		phosphorBloom:        crt.phosphorBloom.Get().(float64),
		chromaticAberration:  crt.chromaticAberration.Get().(float64),
		sharpness:            crt.sharpness.Get().(float64),
		blackLevel:           crt.blackLevel.Get().(float64),
		shine:                crt.shine.Get().(bool),
		gamma:                specification.ColourGen.Gamma.Get().(float64),
	}
	if crt.useBevel.Get().(bool) {
		p.curve = true
		p.curveAmount = float64(bevels.SolidState.CurveAmount)
		p.roundedCorners = true
		p.roundedCornersAmount = float64(bevels.SolidState.RoundCornersAmount)
	}
	return p
}

type crtSequencer struct {
	img      *SdlImgui
	sequence *framebuffer.Flip
	phosphor *framebuffer.Flip

	sharpenShader  shaderProgram
	phosphorShader shaderProgram
	blurShader     shaderProgram
	effectsShader  shaderProgram
	colorShader    shaderProgram

	// the pixel pefect setting at the most recent pass of the crt sequencer. if
	// the setting changes then we clear the phosphor buffer
	mostRecentPixelPerfect bool
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:            img,
		sequence:       framebuffer.NewFlip(true),
		phosphor:       framebuffer.NewFlip(false),
		sharpenShader:  newSharpenShader(),
		phosphorShader: newPhosphorShader(),
		blurShader:     newBlurShader(),
		effectsShader:  newEffectsShader(img.rnd),
		colorShader:    newColorShader(),
	}
	return sh
}

func (sh *crtSequencer) destroy() {
	sh.sequence.Destroy()
	sh.phosphor.Destroy()
	sh.sharpenShader.destroy()
	sh.phosphorShader.destroy()
	sh.blurShader.destroy()
	sh.effectsShader.destroy()
	sh.colorShader.destroy()
}

const crtSeqPhosphor = 0

func (sh *crtSequencer) flushPhosphor() {
	sh.phosphor.Clear()
}

// windowed says that the texture being processed is inside an imgui window and
// not drawn directly onto the background. for example, the crt image in the
// debugger TV Screen window should have a windowed value of true
//
// returns the textureID of the processed image
func (sh *crtSequencer) process(env shaderEnvironment, textureID uint32,
	numScanlines int, numClocks int,
	prefs crtSeqPrefs, rotation specification.Rotation, screenshot bool) uint32 {

	// phosphor draw
	phosphorPasses := 1

	// make sure sequence fraembuffers are correct size
	sh.sequence.Setup(env.width, env.height)

	// clear phosphor is enabled state has changed
	if prefs.pixelPerfect != sh.mostRecentPixelPerfect {
		sh.phosphor.Clear()
	}

	// note the enabled flag for comparison next frame
	sh.mostRecentPixelPerfect = prefs.pixelPerfect

	// also make sure our phosphor framebuffer is correct
	if sh.phosphor.Setup(env.width, env.height) {
		// if the change in framebuffer size is significant then graphical
		// artefacts can sometimes be seen. a possible solution to this is to
		// curtail the processing and return from the function here but this
		// results in a blank frame being rendered, which is just an artefact of
		// a different type
		phosphorPasses = 3
	}

	env.textureID = textureID

	// sharpen image
	env.textureID = sh.sequence.Process(func() {
		sh.sharpenShader.(*sharpenShader).process(env, 2)
		env.draw()
	})

	// neutral shader to keep frame buffer oriented correctly
	env.textureID = sh.sequence.Process(func() {
		sh.colorShader.setAttributes(env)
		env.draw()
	})

	// the newly copied texture will be used for the phosphor blending
	newFrameForPhosphor := env.textureID

	// apply "phosphor". how this is done depends on whether the CRT effects are
	// enabled. if they are not then the treatment is slightly different
	for i := 0; i < phosphorPasses; i++ {
		if prefs.pixelPerfect {
			// this draw doesn't do anything except keep the phosphor textures
			// the correct orientation. if we don't have this then we can see
			// inverted graphical artefacts when we switch between pixelperfect
			// and CRT rendering
			env.textureID = sh.phosphor.TextureID()
			env.textureID = sh.phosphor.Process(func() {
				sh.colorShader.(*colorShader).setAttributes(env)
				env.draw()
			})

			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.textureID = sh.phosphor.TextureID()
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).process(env, float32(prefs.pixelPerfectFade), newFrameForPhosphor)
				env.draw()
			})
		} else {
			if prefs.phosphor {
				// use blur shader to add bloom to previous phosphor
				env.textureID = sh.phosphor.TextureID()
				env.textureID = sh.phosphor.Process(func() {
					sh.blurShader.(*blurShader).process(env, float32(prefs.phosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).process(env, float32(prefs.phosphorLatency), newFrameForPhosphor)
				env.draw()
			})
		}
	}

	if !prefs.pixelPerfect {
		// sharpness value
		env.textureID = sh.sequence.Process(func() {
			sh.blurShader.(*blurShader).process(env, float32(prefs.sharpness))
			env.draw()
		})

		// apply the actual crt effects shader
		env.textureID = sh.sequence.Process(func() {
			sh.effectsShader.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks,
				prefs, rotation, screenshot)
			env.draw()
		})
	}

	return env.textureID
}
