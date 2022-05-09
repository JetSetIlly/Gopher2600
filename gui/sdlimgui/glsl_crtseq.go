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

package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
)

type crtSequencer struct {
	seq                   *framebuffer.Sequence
	img                   *SdlImgui
	scalingShader         shaderProgram
	phosphorShader        shaderProgram
	blackCorrectionShader shaderProgram
	blurShader            shaderProgram
	ghostingShader        shaderProgram
	blendShader           shaderProgram
	effectsShader         shaderProgram
	colorShader           shaderProgram
	effectsShaderFlipped  shaderProgram
	colorShaderFlipped    shaderProgram
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:                   img,
		seq:                   framebuffer.NewSequence(5),
		scalingShader:         newScalingShader(),
		phosphorShader:        newPhosphorShader(img),
		blackCorrectionShader: newBlackCorrectionShader(),
		blurShader:            newBlurShader(),
		ghostingShader:        newGhostingShader(img),
		blendShader:           newBlendShader(),
		effectsShader:         newEffectsShader(img, false),
		colorShader:           newColorShader(false),
		effectsShaderFlipped:  newEffectsShader(img, true),
		colorShaderFlipped:    newColorShader(true),
	}
	return sh
}

func (sh *crtSequencer) destroy() {
	sh.seq.Destroy()
	sh.scalingShader.destroy()
	sh.phosphorShader.destroy()
	sh.blackCorrectionShader.destroy()
	sh.blurShader.destroy()
	sh.ghostingShader.destroy()
	sh.blendShader.destroy()
	sh.effectsShader.destroy()
	sh.colorShader.destroy()
	sh.effectsShaderFlipped.destroy()
	sh.colorShaderFlipped.destroy()
}

// moreProcessing should be true if more shaders are to be applied to the
// framebuffer before presentation
//
// returns the last textureID drawn to as part of the process(). the texture
// returned depends on the value of moreProcessing.
func (sh *crtSequencer) process(env shaderEnvironment, moreProcessing bool, numScanlines int, numClocks int) uint32 {
	const (
		// an accumulation of consecutive frames producing a phosphor effect
		phosphor = iota

		// storage for the initial processing step (ghosting filter)
		processedSrc

		// the finalised texture after all processing. the only thing left to
		// do is to (a) present it, or (b) copy it into idxModeProcessing so it
		// can be processed further
		working

		// the texture used for continued processing once the function has
		// returned (ie. moreProcessing flag is true). this texture is not used
		// in the crtShader for any other purpose and so can be clobbered with
		// no consequence.
		more
	)

	// we'll be chaining many shaders together so use internal projection
	env.useInternalProj = true

	// whether crt effects are enabled
	enabled := sh.img.crtPrefs.Enabled.Get().(bool)

	// phosphor draw
	phosphorPasses := 1

	// make sure our framebuffer is correct. if framebuffer has changed then
	// alter the phosphor/fade options
	if sh.seq.Setup(env.width, env.height) {
		phosphorPasses = 3
	}

	// scale image
	env.srcTextureID = sh.seq.Process(processedSrc, func() {
		sh.scalingShader.(*scalingShader).setAttributesArgs(env, sh.img.playScr)
		env.draw()
	})

	// apply ghosting filter to texture. this is useful for the zookeeper brick effect
	if enabled && sh.img.crtPrefs.Ghosting.Get().(bool) {
		env.srcTextureID = sh.seq.Process(processedSrc, func() {
			sh.ghostingShader.(*ghostingShader).setAttributesArgs(env, float32(sh.img.crtPrefs.GhostingAmount.Get().(float64)))
			env.draw()
		})
	}
	src := env.srcTextureID

	for i := 0; i < phosphorPasses; i++ {
		if enabled {
			if sh.img.crtPrefs.Phosphor.Get().(bool) {
				// use blur shader to add bloom to previous phosphor
				env.srcTextureID = sh.seq.Process(phosphor, func() {
					env.srcTextureID = sh.seq.Texture(phosphor)
					phosphorBloom := sh.img.crtPrefs.PhosphorBloom.Get().(float64)
					sh.blurShader.(*blurShader).setAttributesArgs(env, float32(phosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.srcTextureID = sh.seq.Process(phosphor, func() {
				phosphorLatency := sh.img.crtPrefs.PhosphorLatency.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(phosphorLatency), src)
				env.draw()
			})
		} else {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.srcTextureID = sh.seq.Process(phosphor, func() {
				env.srcTextureID = sh.seq.Texture(phosphor)
				fade := sh.img.crtPrefs.PixelPerfectFade.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(fade), src)
				env.draw()
			})
		}
	}

	if enabled {
		// video-black correction
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blackCorrectionShader.(*blackCorrectionShader).setAttributesArgs(env, float32(sh.img.crtPrefs.BlackLevel.Get().(float64)))
			env.draw()
		})

		// blur result of phosphor a little more
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, float32(sh.img.crtPrefs.Sharpness.Get().(float64)))
			env.draw()
		})

		// // blend blur with src texture
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 0.32, src)
			env.draw()
		})

		if moreProcessing {
			// always clear the "more" texture because the shape of the texture
			// (alpha pixels exposing the window background) may change. this
			// leaves pixels from a previous shader in the texture.
			sh.seq.Clear(more)
			env.srcTextureID = sh.seq.Process(more, func() {
				interference := sh.img.crtPrefs.Interference.Get().(bool)
				noise := sh.img.crtPrefs.Noise.Get().(bool)
				sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, interference, noise)
				env.draw()
			})
		} else {
			env.useInternalProj = false
			interference := sh.img.crtPrefs.Interference.Get().(bool)
			noise := sh.img.crtPrefs.Noise.Get().(bool)
			sh.effectsShader.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, interference, noise)
		}
	} else {
		if moreProcessing {
			// see comment above
			sh.seq.Clear(more)
			env.srcTextureID = sh.seq.Process(more, func() {
				sh.colorShaderFlipped.setAttributes(env)
				env.draw()
			})
		} else {
			env.useInternalProj = false
			sh.colorShader.setAttributes(env)
		}
	}

	return env.srcTextureID
}
