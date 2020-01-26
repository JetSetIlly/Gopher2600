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

package sdlplay

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature implements the GUI interface
//
// MUST NOT be called from the #mainthread
func (scr *SdlPlay) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request {
	case gui.ReqSetVisibleOnStable:
		if scr.IsStable() {
			scr.showWindow(true)
		} else {
			scr.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		scr.showWindow(args[0].(bool))

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.showWindow(true)
		} else {
			scr.showWindow(false)
		}

	case gui.ReqSetScale:
		err = scr.setWindowThread(args[0].(float32))

	case gui.ReqSetFpsCap:
		scr.lmtr.Active = args[0].(bool)

	case gui.ReqCaptureMouse:
		scr.isCaptured = args[0].(bool)
		err = sdl.CaptureMouse(scr.isCaptured)
		if err == nil {
			scr.window.SetGrab(scr.isCaptured)
			if scr.isCaptured {
				sdl.ShowCursor(sdl.DISABLE)
			} else {
				sdl.ShowCursor(sdl.ENABLE)
			}
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return err
}

// SetEventChannel implements the GUI interface
func (scr *SdlPlay) SetEventChannel(eventChannel chan gui.Event) {
	scr.eventChannel = eventChannel
}
