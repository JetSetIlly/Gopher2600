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
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type crtSeqPrefs struct {
	Enabled          bool
	PixelPerfectFade float64

	Curve          bool
	RoundedCorners bool
	Shine          bool
	Mask           bool
	Scanlines      bool
	Interference   bool
	Fringing       bool
	Ghosting       bool
	Phosphor       bool

	CurveAmount          float64
	RoundedCornersAmount float64
	MaskIntensity        float64
	ScanlinesIntensity   float64
	InterferenceLevel    float64
	FringingAmount       float64
	GhostingAmount       float64
	PhosphorLatency      float64
	PhosphorBloom        float64
	Sharpness            float64
	BlackLevel           float64
	Gamma                float64
}

func newCrtSeqPrefs(crt *display.CRT) crtSeqPrefs {
	return crtSeqPrefs{
		Enabled:              crt.Enabled.Get().(bool),
		PixelPerfectFade:     crt.PixelPerfectFade.Get().(float64),
		Curve:                crt.Curve.Get().(bool),
		RoundedCorners:       crt.RoundedCorners.Get().(bool),
		Shine:                crt.Shine.Get().(bool),
		Mask:                 crt.Mask.Get().(bool),
		Scanlines:            crt.Scanlines.Get().(bool),
		Interference:         crt.Interference.Get().(bool),
		Fringing:             crt.Fringing.Get().(bool),
		Ghosting:             crt.Ghosting.Get().(bool),
		Phosphor:             crt.Phosphor.Get().(bool),
		CurveAmount:          crt.CurveAmount.Get().(float64),
		RoundedCornersAmount: crt.RoundedCornersAmount.Get().(float64),
		MaskIntensity:        crt.MaskIntensity.Get().(float64),
		ScanlinesIntensity:   crt.ScanlinesIntensity.Get().(float64),
		InterferenceLevel:    crt.InterferenceLevel.Get().(float64),
		FringingAmount:       crt.FringingAmount.Get().(float64),
		GhostingAmount:       crt.GhostingAmount.Get().(float64),
		PhosphorLatency:      crt.PhosphorLatency.Get().(float64),
		PhosphorBloom:        crt.PhosphorBloom.Get().(float64),
		Sharpness:            crt.Sharpness.Get().(float64),
		BlackLevel:           crt.BlackLevel.Get().(float64),
		Gamma:                specification.ColourGen.Gamma.Get().(float64),
	}
}

type crtSequencer struct {
	img     *SdlImgui
	enabled bool

	sequence *framebuffer.Flip
	phosphor *framebuffer.Flip

	sharpenShader  shaderProgram
	phosphorShader shaderProgram
	blurShader     shaderProgram
	ghostingShader shaderProgram
	effectsShader  shaderProgram
	colorShader    shaderProgram
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:            img,
		sequence:       framebuffer.NewFlip(true),
		phosphor:       framebuffer.NewFlip(false),
		sharpenShader:  newSharpenShader(),
		phosphorShader: newPhosphorShader(),
		blurShader:     newBlurShader(),
		ghostingShader: newGhostingShader(),
		effectsShader:  newCrtSeqEffectsShader(),
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
	sh.ghostingShader.destroy()
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
	windowed bool, numScanlines int, numClocks int,
	prefs crtSeqPrefs, rotation specification.Rotation, screenshot bool) uint32 {

	// the flipY value depends on whether the texture is to be windowed
	env.flipY = windowed

	// phosphor draw
	phosphorPasses := 1

	// make sure sequence fraembuffers are correct size
	sh.sequence.Setup(env.width, env.height)

	// clear phosphor is enabled state has changed
	if prefs.Enabled != sh.enabled {
		sh.phosphor.Clear()
	}

	// note the enabled flag for comparison next frame
	sh.enabled = prefs.Enabled

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

	// apply ghosting filter to texture. this is useful for the zookeeper brick effect
	if prefs.Enabled {
		// if ghosting isn't enabled then we need to run the image through a
		// neutral shader so that the y-orientation is correct
		if prefs.Ghosting {
			env.textureID = sh.sequence.Process(func() {
				sh.ghostingShader.(*ghostingShader).process(env, float32(prefs.GhostingAmount))
				env.draw()
			})
		} else {
			env.textureID = sh.sequence.Process(func() {
				sh.colorShader.setAttributes(env)
				env.draw()
			})
		}
	} else {
		// TV color shader is applied to pixel-perfect shader too
		env.textureID = sh.sequence.Process(func() {
			sh.colorShader.setAttributes(env)
			env.draw()
		})
	}

	// the newly copied texture will be used for the phosphor blending
	newFrameForPhosphor := env.textureID

	// apply "phosphor". how this is done depends on whether the CRT effects are
	// enabled. if they are not then the treatment is slightly different
	for i := 0; i < phosphorPasses; i++ {
		if prefs.Enabled {
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
		} else {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.textureID = sh.phosphor.TextureID()
			env.flipY = true
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).process(env, float32(prefs.PixelPerfectFade), newFrameForPhosphor)
				env.draw()
			})
		}
	}

	// we've possibly altered the flipY value in the phosphor loop above, so we
	// need to restore it to equal the windowed value (ie. as the flipY was at
	// the beginning of the function)
	env.flipY = windowed

	if prefs.Enabled {
		// sharpness value
		env.textureID = sh.sequence.Process(func() {
			sh.blurShader.(*blurShader).process(env, float32(prefs.Sharpness))
			env.draw()
		})

		env.textureID = sh.sequence.Process(func() {
			sh.effectsShader.(*crtSeqEffectsShader).setAttributesArgs(env, numScanlines, numClocks,
				prefs, rotation, screenshot)
			env.draw()
		})
	} else {
		env.textureID = sh.sequence.Process(func() {
			sh.colorShader.setAttributes(env)
			env.draw()
		})
	}

	return env.textureID
}
