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

	tvColor tvColorShaderPrefs
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

		tvColor: tvColorShaderPrefs{
			Brightness: prefs.Colour.Brightness.Get().(float64),
			Contrast:   prefs.Colour.Contrast.Get().(float64),
			Saturation: prefs.Colour.Saturation.Get().(float64),
			Hue:        prefs.Colour.Hue.Get().(float64),
		},
	}
}

type crtSequencer struct {
	img     *SdlImgui
	enabled bool

	sequence *framebuffer.Flip
	phosphor *framebuffer.Flip

	sharpenShader         shaderProgram
	phosphorShader        shaderProgram
	tvColorShader         shaderProgram
	blackCorrectionShader shaderProgram
	blurShader            shaderProgram
	ghostingShader        shaderProgram
	effectsShader         shaderProgram
	colorShader           shaderProgram
}

func newCRTSequencer(img *SdlImgui) *crtSequencer {
	sh := &crtSequencer{
		img:                   img,
		sequence:              framebuffer.NewFlip(true),
		phosphor:              framebuffer.NewFlip(false),
		sharpenShader:         newSharpenShader(),
		phosphorShader:        newPhosphorShader(),
		tvColorShader:         newTVColorShader(),
		blackCorrectionShader: newBlackCorrectionShader(),
		blurShader:            newBlurShader(),
		ghostingShader:        newGhostingShader(),
		effectsShader:         newCrtSeqEffectsShader(),
		colorShader:           newColorShader(),
	}
	return sh
}

func (sh *crtSequencer) destroy() {
	sh.sequence.Destroy()
	sh.phosphor.Destroy()
	sh.sharpenShader.destroy()
	sh.phosphorShader.destroy()
	sh.tvColorShader.destroy()
	sh.blackCorrectionShader.destroy()
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
func (sh *crtSequencer) process(env shaderEnvironment, windowed bool, numScanlines int, numClocks int,
	image textureSpec, prefs crtSeqPrefs, rotation specification.Rotation,
	screenshot bool) uint32 {

	// the flipY value depends on whether the texture is to be windowed
	env.flipY = windowed

	// phosphor draw
	phosphorPasses := 1

	// make sure sequence framebuffer is correct
	_ = sh.sequence.Setup(env.width, env.height)

	// clear phosphor is enabled state has changed
	if prefs.Enabled != sh.enabled {
		sh.phosphor.Clear()
	}

	// note the enabled flag for comparison next frame
	sh.enabled = prefs.Enabled

	// also make sure our phosphor framebuff is correct
	if sh.phosphor.Setup(env.width, env.height) {
		// if the change in framebuffer size is significant then graphical
		// artefacts can sometimes be seen. a possible solution to this is to
		// curtail the processing and return from the function here but this
		// results in a blank frame being rendered, which is just an artefact of
		// a different type
		phosphorPasses = 3
	}

	// sharpen image
	env.textureID = sh.sequence.Process(func() {
		// any sharpen value more than on causes ugly visual artefacts. a value
		// of zero causes the default sharpen value (four) to be used
		sh.sharpenShader.(*sharpenShader).setAttributesArgs(env, image, 1)
		env.draw()
	})

	// apply ghosting filter to texture. this is useful for the zookeeper brick effect
	if prefs.Enabled && prefs.Ghosting {
		env.textureID = sh.sequence.Process(func() {
			sh.ghostingShader.(*ghostingShader).setAttributesArgs(env, float32(prefs.GhostingAmount))
			env.draw()
		})
	}

	newFrameForPhosphor := env.textureID

	for i := 0; i < phosphorPasses; i++ {
		if prefs.Enabled {
			if prefs.Phosphor {
				// use blur shader to add bloom to previous phosphor
				env.textureID = sh.phosphor.Texture()
				env.textureID = sh.phosphor.Process(func() {
					sh.blurShader.(*blurShader).setAttributesArgs(env, float32(prefs.PhosphorBloom))
					env.draw()
				})
			}

			// add new frame to phosphor buffer
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(prefs.PhosphorLatency), newFrameForPhosphor)
				env.draw()
			})
		} else {
			// add new frame to phosphor buffer (using phosphor buffer for pixel perfect fade)
			env.textureID = sh.phosphor.Process(func() {
				sh.phosphorShader.(*phosphorShader).setAttributesArgs(env, float32(prefs.PixelPerfectFade), newFrameForPhosphor)
				env.draw()
			})
		}
	}

	// TV color shader is applied to pixel-perfect shader too
	env.textureID = sh.sequence.Process(func() {
		sh.tvColorShader.(*tvColorShader).setAttributesArgs(env, prefs.tvColor)
		env.draw()
	})

	if prefs.Enabled {
		// video-black correction
		env.textureID = sh.sequence.Process(func() {
			sh.blackCorrectionShader.(*blackCorrectionShader).setAttributesArgs(env, float32(prefs.BlackLevel))
			env.draw()
		})

		// sharpness value
		env.textureID = sh.sequence.Process(func() {
			sh.blurShader.(*blurShader).setAttributesArgs(env, float32(prefs.Sharpness))
			env.draw()
		})

		if windowed {
			env.textureID = sh.sequence.Process(func() {
				sh.effectsShader.(*crtSeqEffectsShader).setAttributesArgs(env, numScanlines, numClocks,
					prefs, rotation, screenshot)
				env.draw()
			})
		} else {
			sh.effectsShader.(*crtSeqEffectsShader).setAttributesArgs(env, numScanlines, numClocks,
				prefs, rotation, screenshot)
		}
	} else {
		if windowed {
			env.textureID = sh.sequence.Process(func() {
				sh.colorShader.setAttributes(env)
				env.draw()
			})
		} else {
			env.flipY = true
			sh.colorShader.setAttributes(env)
		}
	}

	return env.textureID
}
