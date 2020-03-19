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
	"gopher2600/debugger"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/hardware"
)

type featureRequest struct {
	request gui.FeatureReq
	args    []interface{}
}

// SetFeature implements gui.GUI interface
func (img *SdlImgui) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	img.featureReq <- featureRequest{request: request, args: args}
	return <-img.featureErr
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (img *SdlImgui) serviceFeatureRequests(request featureRequest) {
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

	case gui.ReqSetVisibility:

	case gui.ReqToggleVisibility:

	case gui.ReqSetAltColors:
		img.screen.useAltPixels = request.args[0].(bool)

	case gui.ReqToggleAltColors:
		img.screen.useAltPixels = !img.screen.useAltPixels

	case gui.ReqSetScale:
		err = img.screen.setWindowFromThread(request.args[0].(float32))

	case gui.ReqSetPause:
		img.pause(request.args[0].(bool))

	case gui.ReqAddDebugger:
		img.lazy.Dbg = request.args[0].(*debugger.Debugger)

	case gui.ReqAddVCS:
		img.lazy.VCS = request.args[0].(*hardware.VCS)

	case gui.ReqAddDisasm:
		img.dsm = request.args[0].(*disassembly.Disassembly)

	default:
		err = errors.New(errors.UnsupportedGUIRequest, request)
	}

	img.featureErr <- err
}
