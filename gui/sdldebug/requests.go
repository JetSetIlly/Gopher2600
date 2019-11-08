package sdldebug

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature is used to set a television attribute
func (pxtv *SdlDebug) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	switch request {
	case gui.ReqSetVisibilityStable:
		fallthrough

	case gui.ReqSetVisibility:
		if args[0].(bool) {
			pxtv.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				pxtv.pxl.update()
			}
		} else {
			pxtv.window.Hide()
		}

	case gui.ReqToggleVisibility:
		if pxtv.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			pxtv.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				pxtv.pxl.update()
			}
		} else {
			pxtv.window.Hide()
		}

	case gui.ReqSetPause:
		pxtv.paused = args[0].(bool)
		pxtv.pxl.update()

	case gui.ReqSetMasking:
		pxtv.pxl.setMasking(args[0].(bool))
		pxtv.pxl.update()

	case gui.ReqToggleMasking:
		pxtv.pxl.setMasking(!pxtv.pxl.unmasked)
		pxtv.pxl.update()

	case gui.ReqSetAltColors:
		pxtv.pxl.useAltPixels = args[0].(bool)
		pxtv.pxl.update()

	case gui.ReqToggleAltColors:
		pxtv.pxl.useAltPixels = !pxtv.pxl.useAltPixels
		pxtv.pxl.update()

	case gui.ReqSetOverlay:
		pxtv.pxl.useMetaPixels = args[0].(bool)
		pxtv.pxl.update()

	case gui.ReqToggleOverlay:
		pxtv.pxl.useMetaPixels = !pxtv.pxl.useMetaPixels
		pxtv.pxl.update()

	case gui.ReqSetScale:
		pxtv.pxl.setScaling(args[0].(float32))
		pxtv.pxl.update()

	case gui.ReqIncScale:
		if pxtv.pxl.pixelScaleY < 4.0 {
			pxtv.pxl.setScaling(pxtv.pxl.pixelScaleY + 0.1)
			pxtv.pxl.update()
		}

	case gui.ReqDecScale:
		if pxtv.pxl.pixelScaleY > 0.5 {
			pxtv.pxl.setScaling(pxtv.pxl.pixelScaleY - 0.1)
			pxtv.pxl.update()
		}

	default:
		return errors.New(errors.UnknownGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (pxtv *SdlDebug) SetEventChannel(eventChannel chan gui.Event) {
	pxtv.eventChannel = eventChannel
}
