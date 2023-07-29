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
	"fmt"
	"image"

	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/notifications"
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
		return fmt.Errorf("wrong number of arguments (%d instead of %d)", len(args), expectedLen)
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
			img.setEmulationMode(request.args[0].(govern.Mode))
		}

	case gui.ReqEnd:
		err = argLen(request.args, 0)
		if err == nil {
			img.end()
		}

	case gui.ReqFullScreen:
		err = argLen(request.args, 1)
		if err == nil {
			img.plt.setFullScreen(request.args[0].(bool))
		}

	case gui.ReqPeripheralNotify:
		err = argLen(request.args, 2)
		if err == nil {
			port := request.args[0].(plugging.PortID)
			switch port {
			case plugging.PortLeft:
				img.playScr.peripheralLeft.set(request.args[1].(plugging.PeripheralID))
			case plugging.PortRight:
				img.playScr.peripheralRight.set(request.args[1].(plugging.PeripheralID))
			}
		}

	case gui.ReqEmulationNotify:
		if img.isPlaymode() {
			err = argLen(request.args, 1)
			if err == nil {
				img.playScr.emulationNotice.set(request.args[0].(notifications.Notify))
			}
		}

	case gui.ReqCartridgeNotify:
		err = argLen(request.args, 1)
		if err == nil {
			notice := request.args[0].(notifications.Notify)

			switch notice {
			case notifications.NotifyPlusROMNewInstallation:
				img.modal = modalPlusROMFirstInstallation
			case notifications.NotifyUnsupportedDWARF:
				img.modal = modalUnsupportedDWARF
			default:
				img.playScr.cartridgeNotice.set(notice)
			}
		}

	case gui.ReqROMSelector:
		err = argLen(request.args, 0)
		if err == nil {
			if img.isPlaymode() {
				img.wm.playmodeWindows[winSelectROMID].playmodeSetOpen(true)
			} else {
				img.wm.debuggerWindows[winSelectROMID].debuggerSetOpen(true)
			}
		}

	case gui.ReqComparison:
		err = argLen(request.args, 2)
		if err == nil {
			open := false
			if request.args[0] != nil {
				img.wm.playmodeWindows[winComparisonID].(*winComparison).render = request.args[0].(chan *image.RGBA)
				open = true
			}
			if request.args[1] != nil {
				img.wm.playmodeWindows[winComparisonID].(*winComparison).diffRender = request.args[1].(chan *image.RGBA)
				open = true
			}
			img.wm.playmodeWindows[winComparisonID].(*winComparison).playmodeSetOpen(open)
		}

	case gui.ReqBotFeedback:
		err = argLen(request.args, 1)
		if err == nil {
			f := request.args[0].(*bots.Feedback)
			img.wm.playmodeWindows[winBotID].(*winBot).startBotSession(f)
		}

	case gui.ReqCoProcSourceLine:
		err = argLen(request.args, 1)
		if err == nil {
			ln := request.args[0].(*dwarf.SourceLine)
			srcWin := img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
			srcWin.gotoSourceLine(ln)
		}

	case gui.ReqScreenshot:
		switch len(request.args) {
		case 0:
			img.glsl.shaders[playscrShaderID].(*playscrShader).screenshot.startProcess(modeSingle, "")
		case 1:
			img.glsl.shaders[playscrShaderID].(*playscrShader).screenshot.startProcess(modeSingle, request.args[0].(string))
		default:
			err = fmt.Errorf("wrong number of arguments (%d instead of 1 or zero)", len(request.args))
		}

	default:
		err = fmt.Errorf("sdlimgui: unsupport feature request (%s)", request.request)
	}

	if err == nil {
		img.polling.featureSetErr <- nil
	} else {
		img.polling.featureSetErr <- fmt.Errorf("sdlimgui: %s: %w", request.request, err)
	}
}
