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
	args    []gui.FeatureReqData
}

// GetFeature implements gui.GUI interface.
func (img *SdlImgui) GetFeature(request gui.FeatureReq) (gui.FeatureReqData, error) {
	img.polling.featureGet <- featureRequest{request: request}
	return <-img.polling.featureGetData, <-img.polling.featureGetErr
}

// featureRequests have been handed over to the featureGet channel. we service
// any requests on that channel here.
func (img *SdlImgui) serviceGetFeature(request featureRequest) {
	switch request.request {
	case gui.ReqState:
		img.polling.featureGetData <- img.state
		img.polling.featureGetErr <- nil
	default:
		img.polling.featureGetData <- nil
		img.polling.featureGetErr <- curated.Errorf(gui.UnsupportedGuiFeature, request.request)
	}
}

// SetFeature implements gui.GUI interface.
func (img *SdlImgui) SetFeature(request gui.FeatureReq, args ...gui.FeatureReqData) error {
	img.polling.featureSet <- featureRequest{request: request, args: args}
	return <-img.polling.featureSetErr
}

// SetFeatureNoError implements gui.GUI interface.
func (img *SdlImgui) SetFeatureNoError(request gui.FeatureReq, args ...gui.FeatureReqData) {
	img.polling.featureSet <- featureRequest{request: request, args: args}
	go func() {
		<-img.polling.featureSetErr
	}()
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (img *SdlImgui) serviceSetFeature(request featureRequest) {
	var err error

	switch request.request {
	case gui.ReqSetVisibility:
		img.wm.dbgScr.setOpen(request.args[0].(bool))

	case gui.ReqSetPlaymode:
		err = img.setPlaymode(request.args[0].(bool))

	case gui.ReqState:
		img.setEmulationState(request.args[0].(gui.EmulationState))

	case gui.ReqVSync:
		img.screen.crit.section.Lock()
		img.screen.crit.vsync = request.args[0].(bool)
		img.screen.crit.section.Unlock()

	case gui.ReqAddVCS:
		img.vcs = request.args[0].(*hardware.VCS)

	case gui.ReqAddDebugger:
		img.lz.Dbg = request.args[0].(*debugger.Debugger)
		img.vcs = img.lz.Dbg.VCS

	case gui.ReqSetEventChan:
		img.events = request.args[0].(chan gui.Event)

	case gui.ReqSavePrefs:
		err = img.prefs.save()
		if err == nil {
			err = img.crtPrefs.Save()
		}

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
		img.polling.featureSetErr <- nil
	} else {
		img.polling.featureSetErr <- curated.Errorf("sdlimgui: %v", err)
	}
}
