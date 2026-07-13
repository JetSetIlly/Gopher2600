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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/screenshot"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

func (img *SdlImgui) screenshot(mode screenshotMode, path string) {
	if img.mode.Load().(govern.Mode) != govern.ModePlay {
		img.screen.crit.section.Lock()
		defer img.screen.crit.section.Unlock()

		scaled := screenshot.ScaleRawPixels(img.screen.crit.cropPixels)

		// save image to file as a JPEG
		if path == "" {
			path = screenshot.GenerateFilename(img.cache.VCS.Mem.Cart.ShortName, "", "debug")
		}
		screenshot.Save(scaled, path)

		return
	}

	finish := make(chan screenshotResult, 1)
	img.rnd.screenshot(mode, finish)

	// we'll be waiting on the screenshot completion in another goroutine and we
	// don't want to be accessing the cache from there. so we make a copy of the
	// cartridge name value
	cartName := img.cache.VCS.Mem.Cart.ShortName

	go func() {
		// notify that screenshot has been made
		defer img.SetFeature(gui.ReqNotification, notifications.NotifyScreenshot)

		// wait for result and log any errors
		res := <-finish
		if res.err != nil {
			logger.Log(logger.Allow, "screenshot", res.err)
			return
		}

		// save image according to file extension
		if path == "" {
			path = screenshot.GenerateFilename(cartName, "", res.description)
		}
		screenshot.Save(res.image, fmt.Sprintf("%s.jpg", path))
	}()
}
