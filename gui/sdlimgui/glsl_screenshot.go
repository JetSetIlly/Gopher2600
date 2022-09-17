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
	img *SdlImgui
	crt *crtSequencer

	mode       screenshotMode
	exposureCt int

	prefs        crtSeqPrefs
	baseFilename string
}

func newscreenshotSequencer(img *SdlImgui) *screenshotSequencer {
	sh := &screenshotSequencer{
		img: img,
		crt: newCRTSequencer(img),
	}
	return sh
}

func (sh *screenshotSequencer) destroy() {
	sh.crt.destroy()
}

func (sh *screenshotSequencer) startProcess(mode screenshotMode) {
	sh.mode = mode

	// the tag to indicate exposure time in the filename
	var exposureTag string

	// change the crt prefers as appropriate
	sh.prefs = newCrtSeqPrefs(sh.img.crtPrefs)

	switch sh.mode {
	case modeShort:
		exposureTag = "short"
		sh.exposureCt = 3
	case modeLong:
		exposureTag = "long"
		sh.exposureCt = 4
		sh.prefs.PhosphorLatency *= 1.25
		sh.prefs.PhosphorBloom *= 1.1
		sh.prefs.PixelPerfectFade *= 1.25
	case modeVeryLong:
		exposureTag = "verylong"
		sh.exposureCt = 5
		sh.prefs.PhosphorLatency *= 1.5
		sh.prefs.PhosphorBloom *= 1.2
		sh.prefs.PixelPerfectFade *= 1.5
	}

	// make sure values have not got too large
	if sh.prefs.PhosphorLatency > 1.0 {
		sh.prefs.PhosphorLatency = 1.0
	}
	if sh.prefs.PhosphorBloom > 1.0 {
		sh.prefs.PhosphorBloom = 1.0
	}
	if sh.prefs.PixelPerfectFade > 1.0 {
		sh.prefs.PixelPerfectFade = 1.0
	}

	if sh.img.crtPrefs.Enabled.Get().(bool) {
		sh.baseFilename = unique.Filename(fmt.Sprintf("crt_%s", exposureTag), sh.img.vcs.Mem.Cart.ShortName)
	} else {
		sh.baseFilename = unique.Filename(fmt.Sprintf("pix_%s", exposureTag), sh.img.vcs.Mem.Cart.ShortName)
	}
	sh.baseFilename = fmt.Sprintf("%s.jpg", sh.baseFilename)

	sh.crt.flushPhosphor()
}

func (sh *screenshotSequencer) process(env shaderEnvironment, scalingImage scalingImage) {
	if sh.exposureCt <= 0 {
		return
	}

	textureID := sh.crt.process(env, true, false,
		sh.img.playScr.visibleScanlines, specification.ClksVisible,
		sh.img.playScr, sh.prefs)

	sh.exposureCt--
	if sh.exposureCt == 0 {
		sh.SaveJPEG(textureID, sh.baseFilename, sh.img.playScr)
	}
}

// SavesJPEG writes the texture to the specified path.
func (sh *screenshotSequencer) SaveJPEG(textureID uint32, path string, scalingImage scalingImage) {
	_, width, height := scalingImage.scaledTextureSpec()
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	if img == nil {
		logger.Log("screenshot", "save failed: cannot allocate image data")
	}

	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))

	go func() {
		f, err := os.Create(path)
		if err != nil {
			logger.Logf("screenshot", "save failed: %v", err.Error())
			return
		}

		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
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
	}()
}
