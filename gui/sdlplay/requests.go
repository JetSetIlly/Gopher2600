package sdlplay

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature is used to set a television attribute
func (scr *SdlPlay) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	switch request {
	case gui.ReqSetVisibleOnStable:
		if scr.IsStable() {
			scr.showWindow(true)
		} else {
			scr.showOnNextStable = true
		}

	case gui.ReqSetVisibility:
		scr.showWindow(args[0].(bool))

	case gui.ReqSetFPSCap:
		scr.fpsCap = args[0].(bool)

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.window.Show()
		} else {
			scr.window.Hide()
		}

	case gui.ReqSetScale:
		scr.setScaling(args[0].(float32))

	case gui.ReqIncScale:
		if scr.scaleY < 4.0 {
			scr.setScaling(scr.scaleY + 0.1)
		}

	case gui.ReqDecScale:
		if scr.scaleY > 0.5 {
			scr.setScaling(scr.scaleY - 0.1)
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (scr *SdlPlay) SetEventChannel(eventChannel chan gui.Event) {
	scr.eventChannel = eventChannel
}
