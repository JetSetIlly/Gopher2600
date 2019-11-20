package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature is used to set a television attribute
func (scr *SdlDebug) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	switch request {
	case gui.ReqSetVisibility:
		if args[0].(bool) {
			scr.window.Show()
			scr.pxl.update()
		} else {
			scr.window.Hide()
		}

	case gui.ReqToggleVisibility:
		if scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				scr.pxl.update()
			}
		} else {
			scr.window.Hide()
		}

	case gui.ReqSetPause:
		scr.paused = args[0].(bool)
		scr.pxl.update()

	case gui.ReqSetMasking:
		scr.pxl.setMasking(args[0].(bool))
		scr.pxl.update()

	case gui.ReqToggleMasking:
		scr.pxl.setMasking(!scr.pxl.unmasked)
		scr.pxl.update()

	case gui.ReqSetAltColors:
		scr.pxl.useAltPixels = args[0].(bool)
		scr.pxl.update()

	case gui.ReqToggleAltColors:
		scr.pxl.useAltPixels = !scr.pxl.useAltPixels
		scr.pxl.update()

	case gui.ReqSetOverlay:
		scr.pxl.useMetaPixels = args[0].(bool)
		scr.pxl.update()

	case gui.ReqToggleOverlay:
		scr.pxl.useMetaPixels = !scr.pxl.useMetaPixels
		scr.pxl.update()

	case gui.ReqSetScale:
		scr.pxl.setScaling(args[0].(float32))
		scr.pxl.update()

	case gui.ReqIncScale:
		if scr.pxl.pixelScaleY < 4.0 {
			scr.pxl.setScaling(scr.pxl.pixelScaleY + 0.1)
			scr.pxl.update()
		}

	case gui.ReqDecScale:
		if scr.pxl.pixelScaleY > 0.5 {
			scr.pxl.setScaling(scr.pxl.pixelScaleY - 0.1)
			scr.pxl.update()
		}

	default:
		return errors.New(errors.UnsupportedGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (scr *SdlDebug) SetEventChannel(eventChannel chan gui.Event) {
	scr.eventChannel = eventChannel
}
