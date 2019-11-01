package sdl

import (
	"gopher2600/errors"
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// SetFeature is used to set a television attribute
func (pxtv *PixelTV) SetFeature(request gui.FeatureReq, args ...interface{}) (returnedErr error) {
	// lazy (but clear) handling of type assertion errors
	defer func() {
		if r := recover(); r != nil {
			returnedErr = errors.New(errors.PanicError, "sdl.SetFeature()", r)
		}
	}()

	switch request {
	case gui.ReqSetVisibilityStable:
		err := pxtv.scr.stb.resolveSetVisibility()
		if err != nil {
			return err
		}

	case gui.ReqSetVisibility:
		if args[0].(bool) {
			pxtv.scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				pxtv.scr.update()
			}
		} else {
			pxtv.scr.window.Hide()
		}

	case gui.ReqToggleVisibility:
		if pxtv.scr.window.GetFlags()&sdl.WINDOW_HIDDEN == sdl.WINDOW_HIDDEN {
			pxtv.scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				pxtv.scr.update()
			}
		} else {
			pxtv.scr.window.Hide()
		}

	case gui.ReqSetAllowDebugging:
		pxtv.allowDebugging = (args[0].(bool))
		pxtv.scr.update()

	case gui.ReqSetPause:
		pxtv.paused = args[0].(bool)
		pxtv.scr.update()

	case gui.ReqSetMasking:
		pxtv.scr.setMasking(args[0].(bool))
		pxtv.scr.update()

	case gui.ReqToggleMasking:
		pxtv.scr.setMasking(!pxtv.scr.unmasked)
		pxtv.scr.update()

	case gui.ReqSetAltColors:
		pxtv.scr.useAltPixels = args[0].(bool)
		pxtv.scr.update()

	case gui.ReqToggleAltColors:
		pxtv.scr.useAltPixels = !pxtv.scr.useAltPixels
		pxtv.scr.update()

	case gui.ReqSetOverlay:
		pxtv.scr.overlayActive = args[0].(bool)
		pxtv.scr.update()

	case gui.ReqToggleOverlay:
		pxtv.scr.overlayActive = !pxtv.scr.overlayActive
		pxtv.scr.update()

	case gui.ReqSetScale:
		pxtv.scr.setScaling(args[0].(float32))
		pxtv.scr.update()

	case gui.ReqIncScale:
		if pxtv.scr.pixelScaleY < 4.0 {
			pxtv.scr.setScaling(pxtv.scr.pixelScaleY + 0.1)
			pxtv.scr.update()
		}

	case gui.ReqDecScale:
		if pxtv.scr.pixelScaleY > 0.5 {
			pxtv.scr.setScaling(pxtv.scr.pixelScaleY - 0.1)
			pxtv.scr.update()
		}

	default:
		return errors.New(errors.UnknownGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (pxtv *PixelTV) SetEventChannel(eventChannel chan gui.Event) {
	pxtv.eventChannel = eventChannel
}
