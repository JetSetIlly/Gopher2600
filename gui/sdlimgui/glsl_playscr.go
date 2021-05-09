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
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
)

type crtSequencer struct {
	seq                  *framebuffer.Sequence
	img                  *SdlImgui
	phosphorShader       shaderProgram
	blurShader           shaderProgram
	blendShader          shaderProgram
	effectsShader        shaderProgram
	colorShader          shaderProgram
	effectsShaderFlipped shaderProgram
	colorShaderFlipped   shaderProgram
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:                  img,
		seq:                  framebuffer.NewSequence(3),
		phosphorShader:       newPhosphorShader(img),
		blurShader:           newBlurShader(),
		blendShader:          newBlendShader(),
		effectsShader:        newEffectsShader(img, false),
		colorShader:          newColorShader(false),
		effectsShaderFlipped: newEffectsShader(img, true),
		colorShaderFlipped:   newColorShader(true),
	}
	return sh
}

func (sh *crtSequencer) destroy() {
	sh.seq.Destroy()
	sh.phosphorShader.destroy()
	sh.blurShader.destroy()
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
func (sh *crtSequencer) process(env shaderEnvironment, enabled bool, moreProcessing bool) uint32 {
	// make sure our framebuffer is correct
	//
	// any changes to the framebuffer will effect how the next frame is drawn.
	// we get rid of any phosphor effects and there is no blending stage
	//
	// there is an artifact whereby the screen seems to brighten when the frame
	// is being changed. I'm not sure what's causing this but it is something
	// that should be fixed
	//
	// !!TODO: eliminate frame brightening on size change
	changed := sh.seq.Setup(env.width, env.height)

	env.useInternalProj = true
	src := env.srcTextureID

	const (
		// an accumulation of consecutive frames producing a phosphor effect
		crtPhosphorIdx = iota

		// the finalised texture after all processing. the only thing left to
		// do is to (a) present it, or (b) copy it into idxModeProcessing so it
		// can be processed further
		crtLastIdx

		// the texture used for continued processing once the function has
		// returned (ie. moreProcessing flag is true). this texture is not used
		// in the crtShader for any other purpose and so can be clobbered with
		// no consequence.
		crtMoreProcessingIdx
	)

	if enabled {
		if !changed {
			if sh.img.crtPrefs.Phosphor.Get().(bool) {
				// use blur shader to add bloom to previous phosphor
				env.srcTextureID = sh.seq.Process(crtPhosphorIdx, func() {
					env.srcTextureID = sh.seq.Texture(crtPhosphorIdx)
					phosphorBloom := sh.img.crtPrefs.PhosphorBloom.Get().(float64)
					sh.blurShader.(*blurShader).setAttributesArgs(env, float32(phosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.srcTextureID = sh.seq.Process(crtPhosphorIdx, func() {
				phosphorLatency := sh.img.crtPrefs.PhosphorLatency.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(phosphorLatency), src)
				env.draw()
			})
		}
	} else {
		if !changed {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.srcTextureID = sh.seq.Process(crtPhosphorIdx, func() {
				env.srcTextureID = sh.seq.Texture(crtPhosphorIdx)
				fade := sh.img.crtPrefs.PixelPerfectFade.Get().(float64)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(fade), src)
				env.draw()
			})
		}
	}

	if enabled {
		// blur result of phosphor a little more
		env.srcTextureID = sh.seq.Process(crtLastIdx, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
			env.draw()
		})

		if !changed {
			// blend blur with src texture
			env.srcTextureID = sh.seq.Process(crtLastIdx, func() {
				sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 0.32, src)
				env.draw()
			})
		}

		if moreProcessing {
			sh.seq.Clear(crtMoreProcessingIdx)
			env.srcTextureID = sh.seq.Process(crtMoreProcessingIdx, func() {
				sh.effectsShaderFlipped.setAttributes(env)
				env.draw()
			})
		} else {
			env.useInternalProj = false
			sh.effectsShader.setAttributes(env)
		}
	} else {
		if moreProcessing {
			sh.seq.Clear(crtMoreProcessingIdx)
			env.srcTextureID = sh.seq.Process(crtMoreProcessingIdx, func() {
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

type playscrShader struct {
	img    *SdlImgui
	crt    *crtSequencer
	camera *cameraSequencer
}

func newPlayscrShader(img *SdlImgui) shaderProgram {
	sh := &playscrShader{
		img:    img,
		crt:    newCRTSequencer(img),
		camera: newCameraSequencer(img),
	}
	return sh
}

func (sh *playscrShader) destroy() {
	sh.crt.destroy()
}

func (sh *playscrShader) scheduleScreenshot(extended bool) {
	sh.camera.startExposure(extended)
}

func (sh *playscrShader) setAttributes(env shaderEnvironment) {
	if !sh.img.isPlaymode() {
		return
	}

	sh.img.screen.crit.section.Lock()
	env.width = int32(sh.img.playScr.scaledWidth())
	env.height = int32(sh.img.playScr.scaledHeight())
	sh.img.screen.crit.section.Unlock()

	env.internalProj = env.presentationProj

	// set scissor and viewport
	gl.Viewport(int32(-sh.img.playScr.imagePosMin.X),
		int32(-sh.img.playScr.imagePosMin.Y),
		env.width+(int32(sh.img.playScr.imagePosMin.X*2)),
		env.height+(int32(sh.img.playScr.imagePosMin.Y*2)),
	)
	gl.Scissor(int32(-sh.img.playScr.imagePosMin.X),
		int32(-sh.img.playScr.imagePosMin.Y),
		env.width+(int32(sh.img.playScr.imagePosMin.X*2)),
		env.height+(int32(sh.img.playScr.imagePosMin.Y*2)),
	)

	sh.camera.process(env)

	enabled := sh.img.crtPrefs.Enabled.Get().(bool)
	sh.crt.process(env, enabled, false)
}
