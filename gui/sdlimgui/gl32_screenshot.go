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
	"image/jpeg"
	"os"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

type screenshotFinalise func(shaderEnvironment)

type screenshotSequencer struct {
	img *SdlImgui
	crt *crtSequencer

	// the screenshot mode we're working with
	mode screenshotMode

	// we don't want to start a new process until the working channel is empty
	working chan bool

	// the number of frames required for the screenshot processing. is reduced
	// by onw every frame down to zero
	workingCt int

	// finalise function
	finalise chan screenshotFinalise

	// name to use for saved file
	savePath string

	// for composited screenshots we need to sharpen the shader manually
	compositeSharpen shaderProgram

	// a framebuffer to be used during compositing
	compositeBuffer *framebuffer.Sequence

	// list of exposures. used to create a composited image
	compositeExposures []*image.RGBA
}

// returns texture ID and the width and height of the texture
func (sh *screenshotSequencer) textureSpec() (uint32, float32, float32) {
	width, height := sh.compositeBuffer.Dimensions()
	return sh.compositeBuffer.Texture(0), float32(width), float32(height)
}

func newScreenshotSequencer(img *SdlImgui) *screenshotSequencer {
	sh := &screenshotSequencer{
		img:      img,
		working:  make(chan bool, 1),
		finalise: make(chan screenshotFinalise, 1),

		compositeBuffer:  framebuffer.NewSequence(1),
		compositeSharpen: newSharpenShader(true),
		crt:              newCRTSequencer(img),
	}
	return sh
}

func (sh *screenshotSequencer) destroy() {
	sh.compositeBuffer.Destroy()
	sh.compositeSharpen.destroy()
	sh.crt.destroy()
}

// filenameSuffix will be appended to the short filename of the cartridge. if
// the string is empty then the default suffix is used
func (sh *screenshotSequencer) startProcess(mode screenshotMode, filenameSuffix string) {
	// begin screenshot process if possible
	select {
	case sh.working <- true:
	default:
		logger.Log("screenshot", "previous screenshotting still in progress")
		return
	}

	sh.mode = mode
	switch mode {
	case modeSingle:
		// single screenshot mode requires just one working frame
		sh.workingCt = 1
	case modeMotion:
		// a generous working frame count is required for motion screenshots so that
		// large phosphor values have time to accumulate
		sh.workingCt = 20
	case modeComposite:
		// a working count of six is good because it is divisible by both two
		// and three. this means that screens with flickering elements of both
		// two and three will work well
		sh.workingCt = 6
	}

	sh.crt.flushPhosphor()
	sh.compositeExposures = sh.compositeExposures[:0]

	// prepare file path for when the image needs to be saved
	if len(filenameSuffix) == 0 {
		if sh.img.crtPrefs.Enabled.Get().(bool) {
			sh.savePath = unique.Filename(fmt.Sprintf("crt_%s", mode), sh.img.cache.VCS.Mem.Cart.ShortName)
		} else {
			sh.savePath = unique.Filename(fmt.Sprintf("pix_%s", mode), sh.img.cache.VCS.Mem.Cart.ShortName)
		}
	} else {
		sh.savePath = fmt.Sprintf("%s_%s", sh.img.cache.VCS.Mem.Cart.ShortName, filenameSuffix)
	}
	sh.savePath = fmt.Sprintf("%s.jpg", sh.savePath)
}

func (sh *screenshotSequencer) process(env shaderEnvironment, scalingImage textureSpec) {
	if sh.workingCt <= 0 {
		select {
		case f := <-sh.finalise:
			f(env)
		default:
		}
		return
	}

	// if working channel is empty then make sure exposure is zero
	if len(sh.working) == 0 {
		logger.Log("screenshot", "recovering after a failed screenshot")
		sh.workingCt = 0
	}

	switch sh.mode {
	case modeComposite:
		sh.compositeProcess(env, scalingImage)
	default:
		sh.crtProcess(env, scalingImage)
	}
}

func (sh *screenshotSequencer) crtProcess(env shaderEnvironment, scalingImage textureSpec) {
	prefs := newCrtSeqPrefs(sh.img.crtPrefs)

	if sh.mode == modeMotion {
		switch sh.workingCt {
		case 1:
			prefs.PixelPerfectFade = 1.0
			prefs.PhosphorLatency = 1.0
			prefs.PhosphorBloom = 0.1
		default:
			prefs.Scanlines = false
			prefs.Mask = false
			prefs.Shine = false
			prefs.Noise = false
			prefs.Interference = false
		}
	}

	textureID := sh.crt.process(env, true,
		sh.img.playScr.visibleScanlines, specification.ClksVisible, sh.img.screen.rotation.Load().(specification.Rotation),
		sh.img.playScr, prefs)

	// reduce exposure count and return if there is still more to do
	sh.workingCt--
	if sh.workingCt > 0 {
		return
	}

	final := image.NewRGBA(image.Rect(0, 0, int(env.width), int(env.height)))
	if final == nil {
		logger.Log("screenshot", "save failed: cannot allocate image data")
		<-sh.working
		return
	}

	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(final.Pix))

	sh.finalise <- func(_ shaderEnvironment) {
		saveJPEG(final, sh.savePath)
		<-sh.working
	}
}

func (sh *screenshotSequencer) compositeProcess(env shaderEnvironment, scalingImage textureSpec) {
	// set up frame buffer. if dimensions have changed refuse to continue with
	// screenshot processing
	if sh.compositeBuffer.Setup(env.width, env.height) {
		logger.Log("screenshot", "save failed: emulation window has changed dimensions")
		<-sh.working
		return
	}

	// sharpen image from play screen
	env.srcTextureID = sh.compositeBuffer.Process(0, func() {
		sh.compositeSharpen.(*sharpenShader).setAttributesArgs(env, scalingImage, 1)
		env.draw()
	})

	// retrieve exposure
	newExposure := image.NewRGBA(image.Rect(0, 0, int(env.width), int(env.height)))
	if newExposure == nil {
		logger.Log("screenshot", "save failed: cannot allocate image data")
		<-sh.working
		return
	}

	gl.BindTexture(gl.TEXTURE_2D, env.srcTextureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, env.srcTextureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(newExposure.Pix))

	// add to list of exposures
	sh.compositeExposures = append(sh.compositeExposures, newExposure)

	// reduce exposure count and return if there is still more to do
	sh.workingCt--
	if sh.workingCt > 0 {
		return
	}

	// composite exposures. we can do this in a separate goroutine. send result
	// over the composite channel
	go func() {
		var composite *image.RGBA

		switch len(sh.compositeExposures) {
		case 0:
			logger.Logf("screenshot", "composition: exposure list is empty")
		case 1:
			// if there is only one exposure then the composition is by
			// defintion already completed
			composite = sh.compositeExposures[0]
		default:
			var err error
			composite, err = sh.compositeAssemble()
			if err != nil {
				logger.Logf("screenshot", err.Error())
				composite = nil
			}
		}

		if composite == nil {
			<-sh.working
			return
		}

		sh.finalise <- func(env shaderEnvironment) {
			sh.compositeFinalise(env, composite)
		}
	}()
}

func (sh *screenshotSequencer) compositeAssemble() (*image.RGBA, error) {
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

	// const brightnessAdjust = 1.20
	// ratio := brightnessAdjust / float64(len(sh.compositeExposures))
	// accumulate := func(a uint8, b uint8) uint8 {
	// 	// being careful not to overflow the uint8
	// 	B := int(float64(b) * ratio)
	// 	C := int(a) + B
	// 	if C > 255 {
	// 		return 255
	// 	}
	// 	return uint8(C)
	// }

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
func (sh *screenshotSequencer) compositeFinalise(env shaderEnvironment, composite *image.RGBA) {
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
	textureID := sh.crt.process(env, true,
		sh.img.playScr.visibleScanlines, specification.ClksVisible, sh.img.screen.rotation.Load().(specification.Rotation),
		sh, newCrtSeqPrefs(sh.img.crtPrefs))
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	// copy processed pixels back into composite image
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, env.width, env.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(composite.Pix))

	// save composite image to file
	go func() {
		saveJPEG(composite, sh.savePath)
		<-sh.working
	}()
}

// saveJPEG writes the texture to the specified path.
func saveJPEG(rgba *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		logger.Logf("screenshot", "save failed: %v", err.Error())
		return
	}

	err = jpeg.Encode(f, rgba, &jpeg.Options{Quality: 100})
	if err != nil {
		logger.Logf("screenshot", "save failed: %v", err.Error())
		_ = f.Close()
		return
	}

	err = f.Close()
	if err != nil {
		logger.Logf("screenshot", "save failed: %v", err.Error())
		return
	}

	// indicate success
	logger.Logf("screenshot", "saved: %s", path)
}
