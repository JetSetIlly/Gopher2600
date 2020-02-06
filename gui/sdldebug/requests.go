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

package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/test"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature implements the GUI interface
//
// MUST NOT be called from the #mainthread
func (scr *SdlDebug) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	test.AssertNonMainThread()

	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request {
	case gui.ReqSetVisibility:
		scr.showWindow(args[0].(bool))
		scr.update()

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.showWindow(true)
			scr.update()
		} else {
			scr.showWindow(false)
		}

	case gui.ReqSetPause:
		scr.paused = args[0].(bool)
		scr.update()

	case gui.ReqSetMasking:
		scr.masked = args[0].(bool)
		scr.setWindowFromThread(-1)
		scr.update()

	case gui.ReqToggleMasking:
		scr.masked = !scr.masked
		scr.setWindowFromThread(-1)
		scr.update()

	case gui.ReqSetAltColors:
		scr.useAltColors = args[0].(bool)
		scr.update()

	case gui.ReqToggleAltColors:
		scr.useAltColors = !scr.useAltColors
		scr.update()

	case gui.ReqSetOverlay:
		scr.useOverlay = args[0].(bool)
		scr.update()

	case gui.ReqToggleOverlay:
		scr.useOverlay = !scr.useOverlay
		scr.update()

	case gui.ReqSetScale:
		err = scr.setWindowFromThread(args[0].(float32))
		scr.update()

	case gui.ReqIncScale:
		if scr.pixelScale < 4.0 {
			err = scr.setWindowFromThread(scr.pixelScale + 0.1)
		}
		scr.update()

	case gui.ReqDecScale:
		if scr.pixelScale > 0.5 {
			err = scr.setWindowFromThread(scr.pixelScale - 0.1)
		}
		scr.update()

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return err
}

// SetEventChannel implements the GUI interface
func (scr *SdlDebug) SetEventChannel(events chan gui.Event) {
	scr.events = events
}
