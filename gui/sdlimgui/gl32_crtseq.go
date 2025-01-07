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
	"github.com/jetsetilly/gopher2600/gui/display"
	"github.com/jetsetilly/gopher2600/gui/display/bevels"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type crtSeqPrefs struct {
	PixelPerfect     bool
	PixelPerfectFade float64

	Curve                bool
	CurveAmount          float64
	RoundedCorners       bool
	RoundedCornersAmount float64
	Scanlines            bool
	ScanlinesIntensity   float64
	Mask                 bool
	MaskIntensity        float64
	RFInterference       bool
	RFNoiseLevel         float64
	RFGhostingLevel      float64
	Phosphor             bool
	PhosphorLatency      float64
	PhosphorBloom        float64

	ChromaticAberration float64
	Sharpness           float64
	BlackLevel          float64
	Shine               bool
	Gamma               float64
}

func newCrtSeqPrefs(crt *display.CRT) crtSeqPrefs {
	p := crtSeqPrefs{
		PixelPerfect:         crt.PixelPerfect.Get().(bool),
		PixelPerfectFade:     crt.PixelPerfectFade.Get().(float64),
		Curve:                crt.Curve.Get().(bool),
		CurveAmount:          crt.CurveAmount.Get().(float64),
		RoundedCorners:       crt.RoundedCorners.Get().(bool),
		RoundedCornersAmount: crt.RoundedCornersAmount.Get().(float64),
		Scanlines:            crt.Scanlines.Get().(bool),
		ScanlinesIntensity:   crt.ScanlinesIntensity.Get().(float64),
		Mask:                 crt.Mask.Get().(bool),
		MaskIntensity:        crt.MaskIntensity.Get().(float64),
		RFInterference:       crt.RFInterference.Get().(bool),
		RFNoiseLevel:         crt.RFNoiseLevel.Get().(float64),
		RFGhostingLevel:      crt.RFGhostingLevel.Get().(float64),
		Phosphor:             crt.Phosphor.Get().(bool),
		PhosphorLatency:      crt.PhosphorLatency.Get().(float64),
		PhosphorBloom:        crt.PhosphorBloom.Get().(float64),
		ChromaticAberration:  crt.ChromaticAberration.Get().(float64),
		Sharpness:            crt.Sharpness.Get().(float64),
		BlackLevel:           crt.BlackLevel.Get().(float64),
		Shine:                crt.Shine.Get().(bool),
		Gamma:                specification.ColourGen.Gamma.Get().(float64),
	}
	if crt.UseBevel.Get().(bool) {
		p.Curve = true
		p.CurveAmount = float64(bevels.SolidState.CurveAmount)
		p.RoundedCorners = true
		p.RoundedCornersAmount = float64(bevels.SolidState.RoundCornersAmount)
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
		effectsShader:  newEffectsShader(),
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
	if prefs.PixelPerfect != sh.mostRecentPixelPerfect {
		sh.phosphor.Clear()
	}

	// note the enabled flag for comparison next frame
	sh.mostRecentPixelPerfect = prefs.PixelPerfect

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
		if prefs.PixelPerfect {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.textureID = sh.phosphor.TextureID()
			env.flipY = true
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).process(env, float32(prefs.PixelPerfectFade), newFrameForPhosphor)
				env.draw()
			})
			env.flipY = false
		} else {
			if prefs.Phosphor {
				// use blur shader to add bloom to previous phosphor
				env.textureID = sh.phosphor.TextureID()
				env.textureID = sh.phosphor.Process(func() {
					sh.blurShader.(*blurShader).process(env, float32(prefs.PhosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).process(env, float32(prefs.PhosphorLatency), newFrameForPhosphor)
				env.draw()
			})
		}
	}

	if prefs.PixelPerfect {
		env.textureID = sh.sequence.Process(func() {
			sh.colorShader.setAttributes(env)
			env.draw()
		})
	} else {
		// sharpness value
		env.textureID = sh.sequence.Process(func() {
			sh.blurShader.(*blurShader).process(env, float32(prefs.Sharpness))
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
