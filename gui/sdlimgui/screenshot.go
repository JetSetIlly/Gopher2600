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

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

func (img *SdlImgui) screenshot(mode screenshotMode, filenameSuffix string) {
	finish := make(chan screenshotResult, 1)
	img.rnd.screenshot(mode, finish)

	// we'll be waiting on the screenshot completion in another goroutine and we
	// don't want to be accessing the cache from there. so we make a copy of the
	// cartridge name value
	cartName := img.cache.VCS.Mem.Cart.ShortName

	go func() {
		// wait for result and log any errors
		res := <-finish
		if res.err != nil {
			logger.Log(logger.Allow, "screenshot", res.err)
			return
		}

		// prepare file path for when the image needs to be saved
		var path string

		if len(filenameSuffix) == 0 {
			path = unique.Filename(res.description, cartName)
		} else {
			path = fmt.Sprintf("%s_%s", cartName, filenameSuffix)
		}
		path = fmt.Sprintf("%s.jpg", path)

		// save image to file as a JPEG
		saveJPEG(res.image, path)
	}()
}

// saveJPEG writes the texture to the specified path.
func saveJPEG(rgba *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "save failed: %v", err)
		return
	}

	err = jpeg.Encode(f, rgba, &jpeg.Options{Quality: 100})
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "save failed: %v", err)
		_ = f.Close()
		return
	}

	err = f.Close()
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "save failed: %v", err)
		return
	}

	// indicate success
	logger.Logf(logger.Allow, "screenshot", "saved: %s", path)
}
