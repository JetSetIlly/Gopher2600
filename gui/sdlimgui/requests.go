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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/userinput"
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

// check length of arguments sent with feature request.
func argLen(args []gui.FeatureReqData, expectedLen int) error {
	if len(args) != expectedLen {
		return curated.Errorf("wrong number of arguments (%d instead of %d)", len(args), expectedLen)
	}
	return nil
}

// featureRequests have been handed over to the featureReq channel. we service
// any requests on that channel here.
func (img *SdlImgui) serviceSetFeature(request featureRequest) {
	var err error

	switch request.request {
	case gui.ReqSetPlaymode:
		err = argLen(request.args, 2)
		if err == nil {
			img.setDbgAndVCS(nil, request.args[0].(*hardware.VCS))
			if request.args[1] == nil {
				img.userinput = nil
			} else {
				img.userinput = request.args[1].(chan userinput.Event)
			}
		}

	case gui.ReqSetDebugmode:
		err = argLen(request.args, 2)
		if err == nil {
			img.setDbgAndVCS(request.args[0].(*debugger.Debugger), nil)
			if request.args[1] == nil {
				img.userinput = nil
			} else {
				img.userinput = request.args[1].(chan userinput.Event)
			}
		}

	case gui.ReqState:
		err = argLen(request.args, 1)
		if err == nil {
			img.setEmulationState(request.args[0].(gui.EmulationState))
		}

	case gui.ReqVSync:
		err = argLen(request.args, 1)
		if err == nil {
			img.screen.crit.section.Lock()
			img.screen.crit.vsync = request.args[0].(bool)
			img.screen.crit.section.Unlock()
		}

	case gui.ReqFullScreen:
		err = argLen(request.args, 1)
		if err == nil {
			img.plt.setFullScreen(request.args[0].(bool))
		}

	case gui.ReqSetVisibility:
		err = argLen(request.args, 1)
		if err == nil {
			if img.isPlaymode() {
				err = curated.Errorf("visibility not supported in playmode")
			} else {
				img.wm.dbgScr.setOpen(request.args[0].(bool))
			}
		}

	case gui.ReqPlusROMFirstInstallation:
		err = argLen(request.args, 1)
		if err == nil {
			img.plusROMFirstInstallation = request.args[0].(*gui.PlusROMFirstInstallation)
		}

	case gui.ReqControllerChange:
		if img.state == gui.StateInitialising {
			break
		}

		if img.isPlaymode() {
			port := request.args[0].(plugging.PortID)
			switch port {
			case plugging.LeftPlayer:
				img.playScr.controllerAlertLeft.open(request.args[1].(string))
			case plugging.RightPlayer:
				img.playScr.controllerAlertRight.open(request.args[1].(string))
			}
		}

	default:
		err = curated.Errorf(gui.UnsupportedGuiFeature, request.request)
	}

	if err == nil {
		img.polling.featureSetErr <- nil
	} else {
		img.polling.featureSetErr <- curated.Errorf("sdlimgui: %s: %v", request.request, err)
	}
}
