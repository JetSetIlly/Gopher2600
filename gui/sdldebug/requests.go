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

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature is used to set a television attribute
func (scr *SdlDebug) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request {
	case gui.ReqSetVisibility:
		if args[0].(bool) {
			scr.window.Show()
			err = scr.pxl.update()
		} else {
			scr.window.Hide()
		}

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				err = scr.pxl.update()
			}
		} else {
			scr.window.Hide()
		}

	case gui.ReqSetPause:
		scr.paused = args[0].(bool)
		err = scr.pxl.update()

	case gui.ReqSetMasking:
		scr.pxl.setMasking(args[0].(bool))
		err = scr.pxl.update()

	case gui.ReqToggleMasking:
		scr.pxl.setMasking(!scr.pxl.unmasked)
		err = scr.pxl.update()

	case gui.ReqSetAltColors:
		scr.pxl.useAltPixels = args[0].(bool)
		err = scr.pxl.update()

	case gui.ReqToggleAltColors:
		scr.pxl.useAltPixels = !scr.pxl.useAltPixels
		err = scr.pxl.update()

	case gui.ReqSetOverlay:
		scr.pxl.useMetaPixels = args[0].(bool)
		err = scr.pxl.update()

	case gui.ReqToggleOverlay:
		scr.pxl.useMetaPixels = !scr.pxl.useMetaPixels
		err = scr.pxl.update()

	case gui.ReqSetScale:
		err = scr.pxl.setScaling(args[0].(float32))
		err = scr.pxl.update()

	case gui.ReqIncScale:
		if scr.pxl.pixelScaleY < 4.0 {
			err = scr.pxl.setScaling(scr.pxl.pixelScaleY + 0.1)
			err = scr.pxl.update()
		}

	case gui.ReqDecScale:
		if scr.pxl.pixelScaleY > 0.5 {
			err = scr.pxl.setScaling(scr.pxl.pixelScaleY - 0.1)
			err = scr.pxl.update()
		}

	case gui.ReqSetOverscan:
		if args[0].(bool) {
			err = scr.resizeOverscan()
		} else {
			err = scr.resizeSpec()
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return err
}

// SetEventChannel implements the GUI interface
func (scr *SdlDebug) SetEventChannel(eventChannel chan gui.Event) {
	scr.eventChannel = eventChannel
}
