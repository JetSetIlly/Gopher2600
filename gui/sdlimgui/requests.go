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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature implements gui.GUI interface
func (img *SdlImgui) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request {
	case gui.ReqSetVisibleOnStable:
		if img.tv.IsStable() {
			img.plt.showWindow(true)
		} else {
			img.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		img.plt.showWindow(args[0].(bool))

	case gui.ReqToggleVisibility:
		if img.plt.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			img.plt.showWindow(true)
		} else {
			img.plt.showWindow(false)
		}

	case gui.ReqSetScale:
		err = img.screen.setWindowFromThread(args[0].(float32))

	case gui.ReqSetFpsCap:
		img.lmtr.Active = args[0].(bool)

	case gui.ReqCaptureMouse:
		img.isCaptured = args[0].(bool)
		err = sdl.CaptureMouse(img.isCaptured)
		if err == nil {
			img.plt.window.SetGrab(img.isCaptured)
			if img.isCaptured {
				sdl.ShowCursor(sdl.DISABLE)
				img.plt.window.SetTitle(windowTitleCaptured)
			} else {
				sdl.ShowCursor(sdl.ENABLE)
				img.plt.window.SetTitle(windowTitle)
			}
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return err
}
