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
	"image"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

type featureRequest struct {
	request gui.FeatureReq
	args    []gui.FeatureReqData
}

// SetFeature implements gui.GUI interface.
func (img *SdlImgui) SetFeature(request gui.FeatureReq, args ...gui.FeatureReqData) error {
	img.polling.featureSet <- featureRequest{request: request, args: args}
	return <-img.polling.featureSetErr
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
	case gui.ReqSetEmulationMode:
		err = argLen(request.args, 1)
		if err == nil {
			img.setEmulationMode(request.args[0].(emulation.Mode))
		}

	case gui.ReqEnd:
		err = argLen(request.args, 0)
		if err == nil {
			img.end()
		}

	case gui.ReqMonitorSync:
		err = argLen(request.args, 1)
		if err == nil {
			img.screen.crit.section.Lock()
			img.screen.crit.monitorSync = request.args[0].(bool)
			img.screen.crit.section.Unlock()
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

	case gui.ReqFullScreen:
		err = argLen(request.args, 1)
		if err == nil {
			img.plt.setFullScreen(request.args[0].(bool))
		}

	case gui.ReqPlusROMFirstInstallation:
		err = argLen(request.args, 1)
		if err == nil {
			img.plusROMFirstInstallation = request.args[0].(*gui.PlusROMFirstInstallation)
		}

	case gui.ReqControllerChange:
		if img.isPlaymode() {
			err = argLen(request.args, 2)
			if err == nil {
				port := request.args[0].(plugging.PortID)
				switch port {
				case plugging.PortLeftPlayer:
					img.playScr.peripheralLeft.set(request.args[1].(plugging.PeripheralID))
				case plugging.PortRightPlayer:
					img.playScr.peripheralRight.set(request.args[1].(plugging.PeripheralID))
				}
			}
		}

	case gui.ReqEmulationEvent:
		if img.isPlaymode() {
			err = argLen(request.args, 1)
			if err == nil {
				img.playScr.emulationEvent.set(request.args[0].(emulation.Event))
			}
		}

	case gui.ReqCartridgeEvent:
		if img.isPlaymode() {
			err = argLen(request.args, 1)
			if err == nil {
				img.playScr.cartridgeEvent.set(request.args[0].(mapper.Event))
			}
		}

	case gui.ReqROMSelector:
		err = argLen(request.args, 0)
		if err == nil {
			img.wm.windows[winSelectROMID].setOpen(true)
		}

	case gui.ReqComparison:
		err = argLen(request.args, 2)
		if err == nil {
			img.wm.windows[winComparisonID].(*winComparison).render = request.args[0].(chan *image.RGBA)
			img.wm.windows[winComparisonID].(*winComparison).diffRender = request.args[1].(chan *image.RGBA)
			img.wm.windows[winComparisonID].(*winComparison).setOpen(true)
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
