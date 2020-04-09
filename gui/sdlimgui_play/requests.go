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

package sdlimgui_play

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/veandco/go-sdl2/sdl"
)

type featureRequest struct {
	request gui.FeatureReq
	args    []interface{}
}

// SetFeature implements gui.GUI interface
func (img *SdlImguiPlay) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	img.featureReq <- featureRequest{request: request, args: args}
	return <-img.featureErr
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (img *SdlImguiPlay) serviceFeatureRequests(request featureRequest) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			img.featureErr <- errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	var err error

	switch request.request {
	case gui.ReqSetEventChan:
		img.events = request.args[0].(chan gui.Event)

	case gui.ReqSetVisibleOnStable:
		if img.tv.IsStable() {
			img.plt.window.Show()
		} else {
			img.plt.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		if request.args[0].(bool) {
			img.plt.window.Show()
		} else {
			img.plt.window.Hide()
		}

	case gui.ReqToggleVisibility:
		if img.plt.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			img.plt.window.Show()
		} else {
			img.plt.window.Hide()
		}

	case gui.ReqSetAltColors:

	case gui.ReqToggleAltColors:

	case gui.ReqSetCropping:

	case gui.ReqToggleCropping:

	case gui.ReqSetOverlay:

	case gui.ReqToggleOverlay:

	case gui.ReqIncScale:
		if img.screen.scaling < 4.0 {
			img.screen.scaling += 0.1
		}
		img.plt.fitDisplaySize()

	case gui.ReqDecScale:
		if img.screen.scaling > 0.5 {
			img.screen.scaling -= 0.1
		}
		img.plt.fitDisplaySize()

	case gui.ReqSetScale:
		img.screen.scaling = request.args[0].(float32)
		img.plt.fitDisplaySize()

	default:
		err = errors.New(errors.UnsupportedGUIRequest, request)
	}

	img.featureErr <- err
}
