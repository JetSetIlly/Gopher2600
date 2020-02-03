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

package sdlwindows

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature implements gui.GUI interface
func (wnd *SdlWindows) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request {
	case gui.ReqSetVisibleOnStable:
		if wnd.tv.IsStable() {
			wnd.platform.showWindow(true)
		} else {
			wnd.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		wnd.platform.showWindow(args[0].(bool))

	case gui.ReqToggleVisibility:
		if wnd.platform.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			wnd.platform.showWindow(true)
		} else {
			wnd.platform.showWindow(false)
		}

	case gui.ReqSetScale:
		err = wnd.screen.setScale(args[0].(float32))

	case gui.ReqSetFpsCap:
		wnd.lmtr.Active = args[0].(bool)

	case gui.ReqCaptureMouse:
		wnd.isCaptured = args[0].(bool)
		err = sdl.CaptureMouse(wnd.isCaptured)
		if err == nil {
			wnd.platform.window.SetGrab(wnd.isCaptured)
			if wnd.isCaptured {
				sdl.ShowCursor(sdl.DISABLE)
				wnd.platform.window.SetTitle(windowTitleCaptured)
			} else {
				sdl.ShowCursor(sdl.ENABLE)
				wnd.platform.window.SetTitle(windowTitle)
			}
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return err
}
