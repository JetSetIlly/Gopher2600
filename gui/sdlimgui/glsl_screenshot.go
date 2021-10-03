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
	"github.com/jetsetilly/gopher2600/resources/unique"
)

type screenshotMode int

const (
	modeSingle screenshotMode = iota
	modeDouble
	modeTriple
)

type screenshotSequencer struct {
	seq *framebuffer.Sequence
	img *SdlImgui

	phosphorShader        shaderProgram
	blackCorrectionShader shaderProgram
	blurShader            shaderProgram
	ghostingShader        shaderProgram
	blendShader           shaderProgram
	effectsShaderFlipped  shaderProgram
	colorShaderFlipped    shaderProgram

	mode screenshotMode

	exposureLength int
	exposureCt     int
	crtProcessing  bool
	baseFilename   string
}

func newscreenshotSequencer(img *SdlImgui) *screenshotSequencer {
	sh := &screenshotSequencer{
		img:                   img,
		seq:                   framebuffer.NewSequence(6),
		phosphorShader:        newPhosphorShader(img),
		blackCorrectionShader: newBlackCorrectionShader(),
		blurShader:            newBlurShader(),
		ghostingShader:        newGhostingShader(img),
		blendShader:           newBlendShader(),
		effectsShaderFlipped:  newEffectsShader(img, true),
		colorShaderFlipped:    newColorShader(true),
	}
	return sh
}

func (sh *screenshotSequencer) destroy() {
	sh.seq.Destroy()
	sh.phosphorShader.destroy()
	sh.blackCorrectionShader.destroy()
	sh.blurShader.destroy()
	sh.ghostingShader.destroy()
	sh.blendShader.destroy()
	sh.effectsShaderFlipped.destroy()
	sh.colorShaderFlipped.destroy()
}

func (sh *screenshotSequencer) startProcess(mode screenshotMode) {
	sh.mode = mode

	// give the process() function some time to accumulate a suitable phosphor
	// texture etc.
	sh.exposureCt = -3

	switch sh.mode {
	case modeSingle:
		sh.exposureLength = 1
	case modeDouble:
		sh.exposureLength = 5
	case modeTriple:
		sh.exposureLength = 5
	}

	sh.crtProcessing = sh.img.crtPrefs.Enabled.Get().(bool)
	if sh.crtProcessing {
		sh.baseFilename = unique.Filename("crt", sh.img.vcs.Mem.Cart.ShortName)
	} else {
		sh.baseFilename = unique.Filename("pix", sh.img.vcs.Mem.Cart.ShortName)
	}
}

func (sh *screenshotSequencer) process(env shaderEnvironment) {
	const (
		// an accumulation of consecutive frames producing a phosphor effect
		phosphor = iota

		// storage for the initial processing step (ghosting filter)
		processedSrc

		// the working screenshot texture
		working

		// the final screenshot image
		final

		// start of raw pixels from previous frames
		prev
	)

	// nothing to do
	if sh.exposureCt >= sh.exposureLength {
		return
	}

	_ = sh.seq.Setup(env.width, env.height)

	env.useInternalProj = true

	// apply ghosting filter to texture. this is useful for the zookeeper brick effect
	if sh.crtProcessing && sh.img.crtPrefs.Ghosting.Get().(bool) {
		env.srcTextureID = sh.seq.Process(processedSrc, func() {
			sh.ghostingShader.(*ghostingShader).setAttributesArgs(env, float32(sh.img.crtPrefs.GhostingAmount.Get().(float64)))
			env.draw()
		})
	}
	src := env.srcTextureID

	if sh.crtProcessing {
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

		// video black correction
		if sh.img.crtPrefs.Curve.Get().(bool) {
			env.srcTextureID = sh.seq.Process(working, func() {
				sh.blackCorrectionShader.(*blackCorrectionShader).setAttributes(env)
				env.draw()
			})
		}
	} else {
		// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
		env.srcTextureID = sh.seq.Process(phosphor, func() {
			env.srcTextureID = sh.seq.Texture(phosphor)
			fade := sh.img.crtPrefs.PixelPerfectFade.Get().(float64)
			sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(fade), src)
			env.draw()
		})
	}

	// filename for saved file
	var filename string

	switch sh.mode {
	case modeSingle:
		sh.exposureCt++
		filename = fmt.Sprintf("%s.jpg", sh.baseFilename)

	case modeDouble:
		// blend current phosphor with current and previous frames
		env.srcTextureID = sh.seq.Process(working, func() {
			t := sh.seq.Texture(prev + (sh.exposureCt % 2))
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, t)
			env.draw()
		})

		sh.exposureCt++

		filename = fmt.Sprintf("%s_double_%d.jpg", sh.baseFilename, sh.exposureCt)

	case modeTriple:
		sh.exposureCt++

		// blend two previous frames to create long exposure
		t := sh.seq.Texture(prev)
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, t)
			env.draw()
		})
		t = sh.seq.Texture(prev + 1)
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, t)
			env.draw()
		})

		// set filename
		filename = fmt.Sprintf("%s_triple_%d.jpg", sh.baseFilename, sh.exposureCt)
	}

	// blend current frame
	env.srcTextureID = sh.seq.Process(working, func() {
		sh.blendShader.(*blendShader).setAttributesArgs(env, 1.0, 1.0, src)
		env.draw()
	})

	if sh.crtProcessing {
		// blur result of blended frames a little more
		env.srcTextureID = sh.seq.Process(working, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, float32(sh.img.crtPrefs.Sharpness.Get().(float64)))
			env.draw()
		})

		// create final screenshot
		env.srcTextureID = sh.seq.Process(final, func() {
			numScanlines := sh.img.wm.dbgScr.numScanlines
			numClocks := specification.ClksVisible
			sh.effectsShaderFlipped.(*effectsShader).setAttributesArgs(env, numScanlines, numClocks, false)
			env.draw()
		})
	} else {
		// create final screenshot
		env.srcTextureID = sh.seq.Process(final, func() {
			sh.colorShaderFlipped.setAttributes(env)
			env.draw()
		})
	}

	// save final texture if exposure count is one or more
	if sh.exposureCt >= 1 {
		sh.seq.SaveJPEG(final, filename, "screenshot")
	}

	// create copy of raw frame. we've bumped the frames counter already but
	// that's okay because it simplfies the prev modulo addition next frame for
	// double exposure
	if sh.exposureCt <= sh.exposureLength {
		d := sh.exposureCt % 2
		if d < 0 {
			d *= -1
		}

		_ = sh.seq.Process(prev+d, func() {
			env.srcTextureID = src
			sh.colorShaderFlipped.setAttributes(env)
			env.draw()
		})
	}
}
