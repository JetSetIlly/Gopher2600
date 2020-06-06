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
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

type featureRequest struct {
	request gui.FeatureReq
	args    []interface{}
}

// ReqFeature implements the GUI interface
func (scr *SdlDebug) ReqFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	scr.featureReq <- featureRequest{request: request, args: args}
	err := <-scr.featureErr
	return err
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (scr *SdlDebug) serviceFeatureRequests(request featureRequest) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			scr.featureErr <- errors.New(errors.PanicError, "sdl.ReqFeature()", r)
		}
	}()

	var err error

	switch request.request {
	case gui.ReqSetEventChan:
		scr.events = request.args[0].(chan gui.Event)

	case gui.ReqSetVisibility:
		scr.showWindow(request.args[0].(bool))
		scr.update()

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.showWindow(true)
			scr.update()
		} else {
			scr.showWindow(false)
		}

	case gui.ReqSetPause:
		scr.paused = request.args[0].(bool)
		scr.update()

	case gui.ReqSetCropping:
		scr.cropped = request.args[0].(bool)
		scr.setWindow(-1)
		scr.update()

	case gui.ReqToggleCropping:
		scr.cropped = !scr.cropped
		scr.setWindow(-1)
		scr.update()

	case gui.ReqSetDbgColors:
		scr.useDbgColors = request.args[0].(bool)
		scr.update()

	case gui.ReqToggleDbgColors:
		scr.useDbgColors = !scr.useDbgColors
		scr.update()

	case gui.ReqSetOverlay:
		scr.useOverlay = request.args[0].(bool)
		scr.update()

	case gui.ReqToggleOverlay:
		scr.useOverlay = !scr.useOverlay
		scr.update()

	case gui.ReqSetScale:
		err = scr.setWindow(request.args[0].(float32))
		scr.update()

	case gui.ReqIncScale:
		if scr.pixelScale < 4.0 {
			err = scr.setWindow(scr.pixelScale + 0.1)
		}
		scr.update()

	case gui.ReqDecScale:
		if scr.pixelScale > 0.5 {
			err = scr.setWindow(scr.pixelScale - 0.1)
		}
		scr.update()

	case gui.ReqSavePrefs:
		// no gui related prefs to save

	case gui.ReqChangingCartridge:
		// gui doesn't need to know when the cartridge is being changed

	default:
		err = errors.New(errors.UnsupportedGUIRequest, request.request)
	}

	scr.featureErr <- err
}
