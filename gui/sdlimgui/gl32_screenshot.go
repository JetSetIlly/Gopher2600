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
	"fmt"
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type gl32Screenshot struct {
	img *SdlImgui
	crt *crtSequencer

	// when the screenshot process is finished. the channel is sent to the
	// startProcess() function
	finish chan screenshotResult

	// the description of the screenshot to be returned over the finish channel
	// as part of screenshotResult
	description string

	// the screenshot mode we're working with
	mode screenshotMode

	// the number of frames required for the screenshot processing
	frames int

	// for composited screenshots we need to sharpen the shader manually
	compositeSharpen shaderProgram

	// a framebuffer to be used during compositing
	compositeBuffer *framebuffer.Sequence

	// list of exposures. used to create a composited image
	compositeExposures []*image.RGBA

	// finalisation of sequence. the function will be called in the main
	// goroutine so this is used for the composite process
	finalise chan func(shaderEnvironment) *image.RGBA
}

// returns texture ID and the width and height of the texture
func (sh *gl32Screenshot) textureSpec() (uint32, float32, float32) {
	width, height := sh.compositeBuffer.Dimensions()
	return sh.compositeBuffer.Texture(0), float32(width), float32(height)
}

func newGl32Screenshot(img *SdlImgui) *gl32Screenshot {
	sh := &gl32Screenshot{
		img:      img,
		finalise: make(chan func(shaderEnvironment) *image.RGBA, 1),

		compositeBuffer:  framebuffer.NewSequence(1),
		compositeSharpen: newSharpenShader(true),
		crt:              newCRTSequencer(img),
	}
	return sh
}

func (sh *gl32Screenshot) destroy() {
	sh.compositeBuffer.Destroy()
	sh.compositeSharpen.destroy()
	sh.crt.destroy()
}

// filenameSuffix will be appended to the short filename of the cartridge. if
// the string is empty then the default suffix is used
func (sh *gl32Screenshot) start(mode screenshotMode, finish chan screenshotResult) {
	// begin screenshot process if possible
	if sh.finish != nil {
		finish <- screenshotResult{
			err: fmt.Errorf("previous screenshotting still in progress"),
		}
		return
	}

	// note the channel to use on screenshot completion
	sh.finish = finish

	// mode of screenshot
	sh.mode = mode

	// description of screenshot to be returned to caller over finish channel
	if sh.img.crtPrefs.Enabled.Get().(bool) {
		sh.description = fmt.Sprintf("crt_%s", sh.mode)
	} else {
		sh.description = fmt.Sprintf("pix_%s", sh.mode)
	}

	switch sh.mode {
	case modeSingle:
		// single screenshot mode requires just one working frame
		sh.frames = 1
	case modeMotion:
		// a generous working frame count is required for motion screenshots so that
		// large phosphor values have time to accumulate
		sh.frames = 20
	case modeComposite:
		// a working count of six is good because it is divisible by both two
		// and three. this means that screens with flickering elements of both
		// two and three will work well
		sh.frames = 6
	}

	sh.crt.flushPhosphor()
	sh.compositeExposures = sh.compositeExposures[:0]
}

func (sh *gl32Screenshot) process(env shaderEnvironment, scalingImage textureSpec) {
	// if there is no finish channel then there is nothing to do
	if sh.finish == nil {
		return
	}

	// once frames counter has reached zero, we need to start listening for
	// screen finalise functions
	if sh.frames <= 0 {
		select {
		case f := <-sh.finalise:
			// call finalise function and return over finish channel
			sh.finish <- screenshotResult{
				description: sh.description,
				image:       f(env),
			}

			// indicate that screenshot is completed by forgetting about the
			// finish channel
			sh.finish = nil
		default:
		}
		return
	}

	// screenshotting is still ongoing
	switch sh.mode {
	case modeComposite:
		sh.compositeProcess(env, scalingImage)
	default:
		sh.crtProcess(env, scalingImage)
	}
}

func (sh *gl32Screenshot) crtProcess(env shaderEnvironment, scalingImage textureSpec) {
	prefs := newCrtSeqPrefs(sh.img.crtPrefs)

	if sh.mode == modeMotion {
		switch sh.frames {
		case 1:
			prefs.PixelPerfectFade = 1.0
			prefs.PhosphorLatency = 1.0
			prefs.PhosphorBloom = 0.1
		default:
			prefs.Scanlines = false
			prefs.Mask = false
			prefs.Shine = false
			prefs.Interference = false
		}
	}

	textureID := sh.crt.process(env, true, sh.img.playScr.visibleScanlines, specification.ClksVisible, 0, sh.img.playScr, prefs, sh.img.screen.rotation.Load().(specification.Rotation), true)

	// reduce exposure count and return if there is still more to do
	sh.frames--
	if sh.frames > 0 {
		return
	}

	final := image.NewRGBA(image.Rect(0, 0, int(env.width), int(env.height)))
	if final == nil {
		sh.finish <- screenshotResult{
			err: fmt.Errorf("save failed: cannot allocate image data"),
		}
		sh.finish = nil
		return
	}

	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(final.Pix))

	sh.finish <- screenshotResult{
		description: sh.description,
		image:       final,
	}
	sh.finish = nil
}

func (sh *gl32Screenshot) compositeProcess(env shaderEnvironment, scalingImage textureSpec) {
	// set up composite frame buffer. we don't care if the dimensions have
	// changed (Setup() function returns true)
	_ = sh.compositeBuffer.Setup(env.width, env.height)

	// sharpen image from play screen
	env.srcTextureID = sh.compositeBuffer.Process(0, func() {
		sh.compositeSharpen.(*sharpenShader).setAttributesArgs(env, scalingImage, 1)
		env.draw()
	})

	// retrieve exposure
	newExposure := image.NewRGBA(image.Rect(0, 0, int(env.width), int(env.height)))
	if newExposure == nil {
		sh.finish <- screenshotResult{
			err: fmt.Errorf("save failed: cannot allocate image data"),
		}
		sh.finish = nil
		return
	}

	gl.BindTexture(gl.TEXTURE_2D, env.srcTextureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, env.srcTextureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(newExposure.Pix))

	// add to list of exposures
	sh.compositeExposures = append(sh.compositeExposures, newExposure)

	// reduce exposure count and return if there is still more to do
	sh.frames--
	if sh.frames > 0 {
		return
	}

	// composite exposures. we can do this in a separate goroutine. doing it in
	// the main thread causes a noticeable pause in the emulation
	//
	// note however that the final composition step must be conducting in the
	// main goroutine so we make use of the finalise channel
	go func() {
		composite, err := sh.compositeAssemble()

		if err != nil {
			sh.finish <- screenshotResult{
				err: fmt.Errorf("save failed: %w", err),
			}
			sh.finish = nil
			return
		}

		sh.finalise <- func(env shaderEnvironment) *image.RGBA {
			return sh.compositeFinalise(env, composite)
		}
	}()
}

func (sh *gl32Screenshot) compositeAssemble() (*image.RGBA, error) {
	switch len(sh.compositeExposures) {
	case 0:
		return nil, fmt.Errorf("composition: exposure list is empty")
	case 1:
		// if there is only one exposure then the composition is by
		// defintion already completed
		return sh.compositeExposures[0], nil
	}

	rgba := image.NewRGBA(sh.compositeExposures[0].Rect)
	if rgba == nil {
		return nil, fmt.Errorf("composition: cannot allocate image data")
	}

	width, height := sh.compositeBuffer.Dimensions()

	luminance := func(a color.RGBA) float64 {
		r := float64(a.R) / 255
		g := float64(a.G) / 255
		b := float64(a.B) / 255
		return r*0.2126 + g*0.7152 + b*0.0722
	}

	// returns true if a is brighter than b
	brighter := func(a color.RGBA, b color.RGBA) bool {
		return luminance(a) >= luminance(b)
	}

	blend := func(a uint8, b uint8) uint8 {
		A := float64(a)
		A = A * 0.67
		B := float64(b)
		B = B * 0.33
		return uint8(A + B)
	}

	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			ep := rgba.RGBAAt(x, y)
			for _, e := range sh.compositeExposures {
				np := e.RGBAAt(x, y)

				if brighter(np, ep) {
					ep.R = blend(ep.R, np.R)
					ep.G = blend(ep.G, np.G)
					ep.B = blend(ep.B, np.B)
				}

				// ep.R = accumulate(ep.R, np.R)
				// ep.G = accumulate(ep.G, np.G)
				// ep.B = accumulate(ep.B, np.B)
			}
			ep.A = 255
			rgba.SetRGBA(x, y, ep)
		}
	}

	return rgba, nil
}

// finalise composite by passing it through the CRT shaders and then saving the image
func (sh *gl32Screenshot) compositeFinalise(env shaderEnvironment, composite *image.RGBA) *image.RGBA {
	// copy composite pixels to framebuffer texture
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(composite.Stride)/4)
	gl.BindTexture(gl.TEXTURE_2D, sh.compositeBuffer.Texture(0))
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, int32(composite.Bounds().Size().X), int32(composite.Bounds().Size().Y), 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(composite.Pix))
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	env.srcTextureID = sh.compositeBuffer.Texture(0)

	// pass composite image through CRT shaders
	textureID := sh.crt.process(env, true, sh.img.playScr.visibleScanlines, specification.ClksVisible, 0, sh, newCrtSeqPrefs(sh.img.crtPrefs), sh.img.screen.rotation.Load().(specification.Rotation), true)

	gl.BindTexture(gl.TEXTURE_2D, textureID)

	// copy processed pixels back into composite image
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(composite.Pix))

	return composite
}
