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
	"fmt"

	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
)

type cameraSequencer struct {
	seq *framebuffer.Sequence
	img *SdlImgui

	phosphorShader       shaderProgram
	blurShader           shaderProgram
	blendShader          shaderProgram
	effectsShaderFlipped shaderProgram
	colorShaderFlipped   shaderProgram

	extended bool
	frames   int
}

func newCameraSequencer(img *SdlImgui) *cameraSequencer {
	sh := &cameraSequencer{
		img:                  img,
		seq:                  framebuffer.NewSequence(5),
		phosphorShader:       newPhosphorShader(img),
		blurShader:           newBlurShader(),
		blendShader:          newBlendShader(),
		effectsShaderFlipped: newEffectsShader(img, true),
		colorShaderFlipped:   newColorShader(true),
	}
	return sh
}

func (sh *cameraSequencer) destroy() {
	sh.seq.Destroy()
	sh.phosphorShader.destroy()
	sh.blurShader.destroy()
	sh.blendShader.destroy()
	sh.effectsShaderFlipped.destroy()
	sh.colorShaderFlipped.destroy()
}

func (sh *cameraSequencer) startExposure(extended bool) {
	sh.extended = extended
	if extended {
		sh.frames = 5
	} else {
		sh.frames = 1
	}

	for i := 1; i < sh.seq.Len(); i++ {
		sh.seq.Clear(i)
	}
}

func (sh *cameraSequencer) process(env shaderEnvironment) {
	const (
		// an accumulation of consecutive frames producing a phosphor effect
		phosphor = iota

		// the working camera texture
		camera

		// the final camera image
		final

		// copy of two previous frames raw data (ie. the data before any processing)
		rawA
		rawB
	)

	_ = sh.seq.Setup(env.width, env.height)

	env.useInternalProj = true
	src := env.srcTextureID

	// use blur shader to add bloom to previous phosphor
	env.srcTextureID = sh.seq.Process(phosphor, func() {
		env.srcTextureID = sh.seq.Texture(phosphor)
		phosphorBloom := sh.img.crtPrefs.PhosphorBloom.Get().(float64)
		sh.blurShader.(*blurShader).setAttributesArgs(env, float32(phosphorBloom))
		env.draw()
	})

	// add new frame to phosphor buffer
	env.srcTextureID = sh.seq.Process(phosphor, func() {
		phosphorLatency := sh.img.crtPrefs.PhosphorLatency.Get().(float64)
		sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(phosphorLatency), src)
		env.draw()
	})

	// nothing to do
	if sh.frames == 0 {
		return
	}

	// number of scanlines must be retrieved in screen's critical section
	sh.img.screen.crit.section.Lock()
	numScanlines := sh.img.screen.crit.bottomScanline - sh.img.screen.crit.topScanline
	sh.img.screen.crit.section.Unlock()

	// number of clocks for the camera is ClksVisible
	numClocks := specification.ClksVisible

	// if this is a non-extended screenshot then perform a basic process
	if !sh.extended {
		env.srcTextureID = sh.seq.Process(camera, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, src)
			env.draw()
		})
		// blur result of blended frames a little more
		env.srcTextureID = sh.seq.Process(camera, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
			env.draw()
		})

		// create final camera
		sh.seq.Clear(final)
		sh.seq.Process(final, func() {
			sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, false)
			env.draw()
		})

		// save this frame's results
		filename := fmt.Sprintf("%s.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))
		err := sh.seq.SaveJPEG(final, filename)
		if err != nil {
			logger.Log("camera", err.Error())
		} else {
			logger.Logf("camera", "saved to %s", filename)
		}

		// end non-extended screenshot early
		sh.frames = 0
		return
	}

	// blend current phosphor with current/previous frames
	switch sh.frames {
	case 1:
		env.srcTextureID = sh.seq.Process(camera, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, sh.seq.Texture(rawB))
			env.draw()
		})
		fallthrough
	case 2:
		env.srcTextureID = sh.seq.Process(camera, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, sh.seq.Texture(rawA))
			env.draw()
		})
		fallthrough
	case 3:
		env.srcTextureID = sh.seq.Process(camera, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, src)
			env.draw()
		})
	}

	// blur result of blended frames a little more
	env.srcTextureID = sh.seq.Process(camera, func() {
		sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
		env.draw()
	})

	// create final camera
	sh.seq.Clear(final)
	env.srcTextureID = sh.seq.Process(final, func() {
		sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, false)
		env.draw()
	})

	// filename to save JPEG to
	var filename string

	// make copies of raw and phosphor data for this frame
	switch sh.frames {
	case 3:
		env.srcTextureID = sh.seq.Process(rawA, func() {
			env.srcTextureID = src
			sh.colorShaderFlipped.setAttributes(env)
			env.draw()
		})
		filename = fmt.Sprintf("%s_A.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))
	case 2:
		env.srcTextureID = sh.seq.Process(rawB, func() {
			env.srcTextureID = src
			sh.colorShaderFlipped.setAttributes(env)
			env.draw()
		})
		filename = fmt.Sprintf("%s_AB.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))
	case 1:
		filename = fmt.Sprintf("%s_ABC.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))
	}

	// save this frame's results
	err := sh.seq.SaveJPEG(final, filename)
	if err != nil {
		logger.Log("camera", err.Error())
	} else {
		logger.Logf("camera", "saved to %s", filename)
	}

	// ready for next frame
	sh.frames--

	// return immediately unless we've just reached the end of the frames counter
	if sh.frames != 0 {
		return
	}

	// produce the remaining combinations

	// frame A & C (with phosphor C)
	env.srcTextureID = sh.seq.Process(camera, func() {
		env.srcTextureID = sh.seq.Texture(phosphor)
		sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, sh.seq.Texture(rawA))
		env.draw()
	})
	env.srcTextureID = sh.seq.Process(camera, func() {
		sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, src)
		env.draw()
	})
	env.srcTextureID = sh.seq.Process(camera, func() {
		sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
		env.draw()
	})
	sh.seq.Clear(final)
	env.srcTextureID = sh.seq.Process(final, func() {
		sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, false)
		env.draw()
	})

	filename = fmt.Sprintf("%s_AC.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))

	err = sh.seq.SaveJPEG(final, filename)
	if err != nil {
		logger.Log("camera", err.Error())
	} else {
		logger.Logf("camera", "saved to %s", filename)
	}

	// frame B & C (with phosphor C)
	env.srcTextureID = sh.seq.Process(camera, func() {
		env.srcTextureID = sh.seq.Texture(phosphor)
		sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, sh.seq.Texture(rawB))
		env.draw()
	})
	env.srcTextureID = sh.seq.Process(camera, func() {
		sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, src)
		env.draw()
	})
	env.srcTextureID = sh.seq.Process(camera, func() {
		sh.blurShader.(*blurShader).setAttributesArgs(env, 0.17)
		env.draw()
	})
	sh.seq.Clear(final)
	env.srcTextureID = sh.seq.Process(final, func() {
		sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, false)
		env.draw()
	})

	filename = fmt.Sprintf("%s_BC.jpg", paths.UniqueFilename("camera", sh.img.vcs.Mem.Cart.ShortName))

	err = sh.seq.SaveJPEG(final, filename)
	if err != nil {
		logger.Log("camera", err.Error())
	} else {
		logger.Logf("camera", "saved to %s", filename)
	}

	return
}
