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

type featureRequest struct {
	request gui.FeatureReq
	args    []interface{}
}

// SetFeature implements the GUI interface
func (scr *SdlPlay) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	scr.featureReq <- featureRequest{request: request, args: args}
	return <-scr.featureErr
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (scr *SdlPlay) serviceFeatureRequests(request featureRequest) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			scr.featureErr <- errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request.request {
	case gui.ReqSetEventChan:
		scr.events = request.args[0].(chan gui.Event)

	case gui.ReqSetVisibleOnStable:
		if scr.IsStable() {
			scr.showWindow(true)
		} else {
			scr.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		scr.showWindow(request.args[0].(bool))

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.showWindow(true)
		} else {
			scr.showWindow(false)
		}

	case gui.ReqSetScale:
		err = scr.setWindow(request.args[0].(float32))

	default:
		err = errors.New(errors.UnsupportedGUIRequest, request.request)
	}

	scr.featureErr <- err
}
