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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
)

type featureRequest struct {
	request gui.FeatureReq
	args    []interface{}
}

// ReqFeature implements gui.GUI interface.
func (img *SdlImgui) ReqFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	img.featureReq <- featureRequest{request: request, args: args}
	return <-img.featureErr
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (img *SdlImgui) serviceFeatureRequests(request featureRequest) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	var err error

	switch request.request {
	case gui.ReqSetEventChan:
		img.events = request.args[0].(chan gui.Event)

	case gui.ReqSetVisibility:
		img.wm.dbgScr.setOpen(request.args[0].(bool))

	case gui.ReqToggleVisibility:
		img.wm.dbgScr.setOpen(!img.wm.dbgScr.isOpen())

	case gui.ReqPause:
		img.paused = request.args[0].(bool)

	case gui.ReqSetDbgColors:
		img.wm.dbgScr.debugColors = request.args[0].(bool)

	case gui.ReqToggleDbgColors:
		img.wm.dbgScr.debugColors = !img.wm.dbgScr.debugColors

	case gui.ReqSetCropping:
		img.wm.dbgScr.setCropping(request.args[0].(bool))

	case gui.ReqToggleCropping:
		img.wm.dbgScr.setCropping(!img.wm.dbgScr.cropped)

	case gui.ReqSetOverlay:
		img.wm.dbgScr.overlay = request.args[0].(bool)

	case gui.ReqToggleOverlay:
		img.wm.dbgScr.overlay = !img.wm.dbgScr.overlay

	case gui.ReqCRTeffects:
		img.wm.dbgScr.pixelPerfect = !request.args[0].(bool)

	case gui.ReqIncScale:
		img.setScale(0.1, true)

	case gui.ReqDecScale:
		img.setScale(-0.1, true)

	case gui.ReqSetScale:
		img.setScale(request.args[0].(float32), false)

	case gui.ReqAddVCS:
		img.vcs = request.args[0].(*hardware.VCS)

	case gui.ReqAddDebugger:
		img.lz.Dbg = request.args[0].(*debugger.Debugger)
		img.vcs = img.lz.Dbg.VCS

	case gui.ReqSetPlaymode:
		err = img.setPlaymode(request.args[0].(bool))

	case gui.ReqSavePrefs:
		err = img.prefs.Save()

	case gui.ReqChangingCartridge:
		// a new cartridge requires us to reset the lazy system (see the
		// lazyvalues.Reset() function commentary for why)
		img.lz.Reset(request.args[0].(bool))

	case gui.ReqPlusROMFirstInstallation:
		img.plusROMFirstInstallation = request.args[0].(*gui.PlusROMFirstInstallation)

	default:
		err = curated.Errorf(gui.UnsupportedGuiFeature, request.request)
	}

	if err == nil {
		img.featureErr <- nil
	} else {
		img.featureErr <- curated.Errorf("sdlimgui: %v", err)
	}
}
