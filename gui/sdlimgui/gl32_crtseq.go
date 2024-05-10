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
	Bevel          bool
	Shine          bool
	Mask           bool
	Scanlines      bool
	Interference   bool
	Flicker        bool
	Fringing       bool
	Ghosting       bool
	Phosphor       bool

	CurveAmount          float64
	RoundedCornersAmount float64
	BevelSize            float64
	MaskIntensity        float64
	MaskFine             float64
	ScanlinesIntensity   float64
	ScanlinesFine        float64
	InterferenceLevel    float64
	FlickerLevel         float64
	FringingAmount       float64
	GhostingAmount       float64
	PhosphorLatency      float64
	PhosphorBloom        float64
	Sharpness            float64
	BlackLevel           float64

	Brightness float64
	Contrast   float64
	Saturation float64
	Hue        float64
}

func newCrtSeqPrefs(prefs *display.Preferences) crtSeqPrefs {
	return crtSeqPrefs{
		Enabled:              prefs.CRT.Enabled.Get().(bool),
		PixelPerfectFade:     prefs.CRT.PixelPerfectFade.Get().(float64),
		Curve:                prefs.CRT.Curve.Get().(bool),
		RoundedCorners:       prefs.CRT.RoundedCorners.Get().(bool),
		Bevel:                prefs.CRT.Bevel.Get().(bool),
		Shine:                prefs.CRT.Shine.Get().(bool),
		Mask:                 prefs.CRT.Mask.Get().(bool),
		Scanlines:            prefs.CRT.Scanlines.Get().(bool),
		Interference:         prefs.CRT.Interference.Get().(bool),
		Flicker:              prefs.CRT.Flicker.Get().(bool),
		Fringing:             prefs.CRT.Fringing.Get().(bool),
		Ghosting:             prefs.CRT.Ghosting.Get().(bool),
		Phosphor:             prefs.CRT.Phosphor.Get().(bool),
		CurveAmount:          prefs.CRT.CurveAmount.Get().(float64),
		RoundedCornersAmount: prefs.CRT.RoundedCornersAmount.Get().(float64),
		BevelSize:            prefs.CRT.BevelSize.Get().(float64),
		MaskIntensity:        prefs.CRT.MaskIntensity.Get().(float64),
		MaskFine:             prefs.CRT.MaskFine.Get().(float64),
		ScanlinesIntensity:   prefs.CRT.ScanlinesIntensity.Get().(float64),
		ScanlinesFine:        prefs.CRT.ScanlinesFine.Get().(float64),
		InterferenceLevel:    prefs.CRT.InterferenceLevel.Get().(float64),
		FlickerLevel:         prefs.CRT.FlickerLevel.Get().(float64),
		FringingAmount:       prefs.CRT.FringingAmount.Get().(float64),
		GhostingAmount:       prefs.CRT.GhostingAmount.Get().(float64),
		PhosphorLatency:      prefs.CRT.PhosphorLatency.Get().(float64),
		PhosphorBloom:        prefs.CRT.PhosphorBloom.Get().(float64),
		Sharpness:            prefs.CRT.Sharpness.Get().(float64),
		BlackLevel:           prefs.CRT.BlackLevel.Get().(float64),

		Brightness: prefs.Colour.Brightness.Get().(float64),
		Contrast:   prefs.Colour.Contrast.Get().(float64),
		Saturation: prefs.Colour.Saturation.Get().(float64),
		Hue:        prefs.Colour.Hue.Get().(float64),
	}
}

type crtSequencer struct {
	img                   *SdlImgui
	seq                   *framebuffer.Sequence
	sharpenShader         shaderProgram
	phosphorShader        shaderProgram
	tvColorShader         shaderProgram
	blackCorrectionShader shaderProgram
	screenrollShader      shaderProgram
	blurShader            shaderProgram
	ghostingShader        shaderProgram
	effectsShader         shaderProgram
	colorShader           shaderProgram
	effectsShaderFlipped  shaderProgram
	colorShaderFlipped    shaderProgram
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:                   img,
		seq:                   framebuffer.NewSequence(5),
		sharpenShader:         newSharpenShader(true),
		phosphorShader:        newPhosphorShader(),
		tvColorShader:         newTVColorShader(),
		blackCorrectionShader: newBlackCorrectionShader(),
		screenrollShader:      newScreenrollShader(),
		blurShader:            newBlurShader(),
		ghostingShader:        newGhostingShader(),
		effectsShader:         newCrtSeqEffectsShader(false),
		colorShader:           newColorShader(false),
		effectsShaderFlipped:  newCrtSeqEffectsShader(true),
		colorShaderFlipped:    newColorShader(true),
	}
	return sh
}

func (sh *crtSequencer) destroy() {
	sh.seq.Destroy()
	sh.sharpenShader.destroy()
	sh.phosphorShader.destroy()
	sh.tvColorShader.destroy()
	sh.blackCorrectionShader.destroy()
	sh.screenrollShader.destroy()
	sh.blurShader.destroy()
	sh.ghostingShader.destroy()
	sh.effectsShader.destroy()
	sh.colorShader.destroy()
	sh.effectsShaderFlipped.destroy()
	sh.colorShaderFlipped.destroy()
}

const (
	// an accumulation of consecutive frames producing a crtSeqPhosphor effect
	crtSeqPhosphor = iota

	// storage for the initial processing step (ghosting filter)
	crtSeqProcessedSrc

	// the finalised texture after all processing. the only thing left to
	// do is to (a) present it, or (b) copy it into idxModeProcessing so it
	// can be processed further
	crtSeqWorking

	// the texture used for continued processing once the function has
	// returned (ie. moreProcessing flag is true). this texture is not used
	// in the crtShader for any other purpose and so can be clobbered with
	// no consequence.
	crtSeqMore
)

// flush phosphor pixels
func (sh *crtSequencer) flushPhosphor() {
	sh.seq.Clear(crtSeqPhosphor)
}

// moreProcessing should be true if more shaders are to be applied to the
// framebuffer before presentation
//
// returns the last textureID drawn to as part of the process(). the texture
// returned depends on the value of moreProcessing.
//
// if effectsEnabled is turned off then phosphor accumulation and scaling still
// occurs but crt effects are not applied.
//
// integerScaling instructs the scaling shader not to perform any smoothing
func (sh *crtSequencer) process(env shaderEnvironment, moreProcessing bool, numScanlines int, numClocks int,
	screenroll float32, image textureSpec, prefs crtSeqPrefs, rotation specification.Rotation,
	screenshot bool) uint32 {

	// we'll be chaining many shaders together so use internal projection
	env.useInternalProj = true

	// phosphor draw
	phosphorPasses := 1

	// make sure our framebuffer is correct
	if sh.seq.Setup(env.width, env.height) {
		// if the change in framebuffer size is significant then graphical
		// artefacts can sometimes be seen. a possible solution to this is to
		// curtail the processing and return from the function here but this
		// results in a blank frame being rendered, which is just an artefact of
		// a different type
		phosphorPasses = 3
	}

	// sharpen image
	env.srcTextureID = sh.seq.Process(crtSeqProcessedSrc, func() {
		// any sharpen value more than on causes ugly visual artefacts. a value
		// of zero causes the default sharpen value (four) to be used
		sh.sharpenShader.(*sharpenShader).setAttributesArgs(env, image, 1)
		env.draw()
	})

	// apply ghosting filter to texture. this is useful for the zookeeper brick effect
	if prefs.Enabled && prefs.Ghosting {
		env.srcTextureID = sh.seq.Process(crtSeqProcessedSrc, func() {
			sh.ghostingShader.(*ghostingShader).setAttributesArgs(env, float32(prefs.GhostingAmount))
			env.draw()
		})
	}
	src := env.srcTextureID

	for i := 0; i < phosphorPasses; i++ {
		if prefs.Enabled {
			if prefs.Phosphor {
				// use blur shader to add bloom to previous phosphor
				env.srcTextureID = sh.seq.Process(crtSeqPhosphor, func() {
					env.srcTextureID = sh.seq.Texture(crtSeqPhosphor)
					sh.blurShader.(*blurShader).setAttributesArgs(env, float32(prefs.PhosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.srcTextureID = sh.seq.Process(crtSeqPhosphor, func() {
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(prefs.PhosphorLatency), src)
				env.draw()
			})
		} else {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.srcTextureID = sh.seq.Process(crtSeqPhosphor, func() {
				env.srcTextureID = sh.seq.Texture(crtSeqPhosphor)
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(prefs.PixelPerfectFade), src)
				env.draw()
			})
		}
	}

	// screenroll and TV color shaders are applied to pixel-perfect shader too
	env.srcTextureID = sh.seq.Process(crtSeqWorking, func() {
		sh.screenrollShader.(*screenrollShader).setAttributesArgs(env, screenroll)
		env.draw()
	})

	env.srcTextureID = sh.seq.Process(crtSeqWorking, func() {
		sh.tvColorShader.(*tvColorShader).setAttributesArgs(env, prefs)
		env.draw()
	})

	if prefs.Enabled {
		// video-black correction
		env.srcTextureID = sh.seq.Process(crtSeqWorking, func() {
			sh.blackCorrectionShader.(*blackCorrectionShader).setAttributesArgs(env, float32(prefs.BlackLevel))
			env.draw()
		})

		// blur result of phosphor a little more
		env.srcTextureID = sh.seq.Process(crtSeqWorking, func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, float32(prefs.Sharpness))
			env.draw()
		})

		if moreProcessing {
			// always clear the "more" texture because the shape of the texture
			// (alpha pixels exposing the window background) may change. this
			// leaves pixels from a previous shader in the texture.
			sh.seq.Clear(crtSeqMore)
			env.srcTextureID = sh.seq.Process(crtSeqMore, func() {
				sh.effectsShaderFlipped.(*crtSeqEffectsShader).setAttributesArgs(env, numScanlines, numClocks, screenroll,
					prefs, rotation, screenshot)
				env.draw()
			})
		} else {
			env.useInternalProj = false
			sh.effectsShader.(*crtSeqEffectsShader).setAttributesArgs(env, numScanlines, numClocks, screenroll,
				prefs, rotation, screenshot)
		}
	} else {
		if moreProcessing {
			// see comment above
			sh.seq.Clear(crtSeqMore)
			env.srcTextureID = sh.seq.Process(crtSeqMore, func() {
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
