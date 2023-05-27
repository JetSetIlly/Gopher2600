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
	"image"
	"image/jpeg"
	"os"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/framebuffer"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

type screenshotMode int

const (
	modeShort screenshotMode = iota
	modeLong
	modeVeryLong
)

type screenshotSequencer struct {
	img           *SdlImgui
	buffer        *framebuffer.Sequence
	sharpenShader shaderProgram
	crt           *crtSequencer

	// dimensions of buffer
	width  int32
	height int32

	// we don't want to start a new process until the working channel is empty
	working chan bool

	// composited images are returned over the composite channel
	composite chan *image.RGBA

	// list of exposures and counter
	exposures  []*image.RGBA
	exposureCt int

	// name to use for saved file
	baseFilename string
}

// returns texture ID and the width and height of the texture
func (sh *screenshotSequencer) textureSpec() (uint32, float32, float32) {
	return sh.buffer.Texture(0), float32(sh.width), float32(sh.height)
}

func newscreenshotSequencer(img *SdlImgui) *screenshotSequencer {
	sh := &screenshotSequencer{
		img:           img,
		buffer:        framebuffer.NewSequence(1),
		sharpenShader: newSharpenShader(true),
		crt:           newCRTSequencer(img),
		working:       make(chan bool, 1),
		composite:     make(chan *image.RGBA, 1),
	}
	return sh
}

func (sh *screenshotSequencer) destroy() {
	sh.buffer.Destroy()
	sh.sharpenShader.destroy()
	sh.crt.destroy()
}

// filenameSuffix will be appended to the short filename of the cartridge. if
// the string is empty then the default suffix is used
func (sh *screenshotSequencer) startProcess(mode screenshotMode, filenameSuffix string) {
	select {
	case sh.working <- true:
	default:
		logger.Log("screenshot", "previous screenshotting still in progress")
		return
	}

	// the tag to indicate exposure time in the filename
	var exposureTag string

	switch mode {
	case modeShort:
		exposureTag = "short"
		sh.exposureCt = 1
	case modeLong:
		exposureTag = "long"
		sh.exposureCt = 6
	case modeVeryLong:
		exposureTag = "verylong"
		sh.exposureCt = 12
	}
	sh.exposures = sh.exposures[:0]

	if len(filenameSuffix) == 0 {
		if sh.img.crtPrefs.Enabled.Get().(bool) {
			sh.baseFilename = unique.Filename(fmt.Sprintf("crt_%s", exposureTag), sh.img.vcs.Mem.Cart.ShortName)
		} else {
			sh.baseFilename = unique.Filename(fmt.Sprintf("pix_%s", exposureTag), sh.img.vcs.Mem.Cart.ShortName)
		}
	} else {
		sh.baseFilename = fmt.Sprintf("%s_%s", sh.img.vcs.Mem.Cart.ShortName, filenameSuffix)
	}
	sh.baseFilename = fmt.Sprintf("%s.jpg", sh.baseFilename)
}

func (sh *screenshotSequencer) process(env shaderEnvironment, scalingImage sharpenImage) {
	if sh.exposureCt <= 0 {
		// if exposure count is zero then we need to check if there is any
		// composite images to process
		select {
		case composite := <-sh.composite:
			sh.processComposite(env, composite)
			<-sh.working
		default:
		}
		return
	}

	// set up frame buffer
	sh.width = env.width
	sh.height = env.height
	sh.buffer.Setup(sh.width, sh.height)

	// sharpen image from play screen
	env.srcTextureID = sh.buffer.Process(0, func() {
		sh.sharpenShader.(*sharpenShader).setAttributesArgs(env, scalingImage, 1)
		env.draw()
	})

	// retrieve exposure
	newExposure := image.NewRGBA(image.Rect(0, 0, int(sh.width), int(sh.height)))
	if newExposure == nil {
		logger.Log("screenshot", "save failed: cannot allocate image data")

		// this is a failure state so we need to drain the working channel
		<-sh.working
		return
	}

	gl.BindTexture(gl.TEXTURE_2D, env.srcTextureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, env.srcTextureID, 0)
	gl.ReadPixels(0, 0, int32(sh.width), int32(sh.height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(newExposure.Pix))

	// add to list of exposures
	sh.exposures = append(sh.exposures, newExposure)

	// reduce exposure count and return if there is still more to do
	sh.exposureCt--
	if sh.exposureCt > 0 {
		return
	}

	// composite exposures. we can do this in a separate goroutine. send result
	// over the composite channel
	go func() {
		composite, err := sh.compositeExposures()
		if err != nil {
			logger.Logf("screenshot", err.Error())
			composite = nil
		}
		sh.composite <- composite
	}()
}

func (sh *screenshotSequencer) compositeExposures() (*image.RGBA, error) {
	rgba := image.NewRGBA(sh.exposures[0].Rect)
	if rgba == nil {
		return nil, fmt.Errorf("save failed: cannot allocate image data")
	}

	width := rgba.Rect.Max.X - rgba.Rect.Min.X
	height := rgba.Rect.Max.Y - rgba.Rect.Min.Y

	// luminance := func(a color.RGBA) float64 {
	// 	r := float64(a.R) / 255
	// 	g := float64(a.G) / 255
	// 	b := float64(a.B) / 255
	// 	return r*0.2126 + g*0.7152 + b*0.0722
	// }

	// // returns true if a is brighter than b
	// brighter := func(a color.RGBA, b color.RGBA) bool {
	// 	return luminance(a) >= luminance(b)
	// }

	ratio := 1.0 / float64(len(sh.exposures))

	// blend := func(a uint8, b uint8) uint8 {
	// 	A := float64(a)
	// 	A = A * 0.66
	// 	B := float64(b)
	// 	B = B * 0.33
	// 	return uint8(A + B)
	// }

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ep := rgba.RGBAAt(x, y)
			for _, e := range sh.exposures {
				np := e.RGBAAt(x, y)
				// if brighter(np, ep) {
				// ep.R = blend(ep.R, np.R)
				// ep.G = blend(ep.G, np.G)
				// ep.B = blend(ep.B, np.B)
				// }

				ep.R += uint8(float64(np.R) * ratio)
				ep.G += uint8(float64(np.G) * ratio)
				ep.B += uint8(float64(np.B) * ratio)
			}
			ep.A = 255
			rgba.SetRGBA(x, y, ep)
		}
	}

	return rgba, nil
}

// process composite by passing it through the CRT shaders and then saving the image
func (sh *screenshotSequencer) processComposite(env shaderEnvironment, composite *image.RGBA) {
	if composite == nil {
		return
	}

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(composite.Stride)/4)
	gl.BindTexture(gl.TEXTURE_2D, sh.buffer.Texture(0))
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, int32(composite.Bounds().Size().X), int32(composite.Bounds().Size().Y), 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(composite.Pix))
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	env.srcTextureID = sh.buffer.Texture(0)
	sh.crt.flushPhosphor()
	textureID := sh.crt.process(env, true,
		sh.img.playScr.visibleScanlines, specification.ClksVisible,
		sh, newCrtSeqPrefs(sh.img.crtPrefs))
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, int32(sh.width), int32(sh.height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(composite.Pix))

	go func() {
		sh.SaveJPEG(composite, sh.baseFilename, sh.img.playScr)
	}()
}

// SaveJPEG writes the texture to the specified path.
func (sh *screenshotSequencer) SaveJPEG(rgba *image.RGBA, path string, scalingImage sharpenImage) {
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
