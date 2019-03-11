package sdl

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/gui"
)

// GetMetaState returns the TVState object for the named state
func (tv *GUI) GetMetaState(request gui.MetaStateReq) (interface{}, error) {
	tv.crit.guiMutex.Lock()
	defer tv.crit.guiMutex.Unlock()

	switch request {
	case gui.ReqLastKeyboard:
		return fmt.Sprintf("%c", tv.crit.keypress), nil
	case gui.ReqLastMouse:
		return fmt.Sprintf("mouse: hp=%d, sl=%d", tv.crit.lastMouseHorizPos, tv.crit.lastMouseScanline), nil
	case gui.ReqLastMouseHorizPos:
		return tv.crit.lastMouseHorizPos, nil
	case gui.ReqLastMouseScanline:
		return tv.crit.lastMouseScanline, nil
	default:
		return nil, errors.NewGopherError(errors.UnknownGUIRequest, request)
	}
}

// RegisterCallback setups up communication between a GUI goroutine and the
// main goroutine
func (tv *GUI) RegisterCallback(request gui.CallbackReq, channel chan func(), callback func()) error {
	// call embedded implementation and filter out UnknownCallbackRequests
	switch request {
	case gui.ReqOnWindowClose:
		tv.onWindowClose.channel = channel
		tv.onWindowClose.function = callback
	case gui.ReqOnKeyboard:
		tv.onKeyboard.channel = channel
		tv.onKeyboard.function = callback
	case gui.ReqOnMouseButtonLeft:
		tv.onMouseButtonLeft.channel = channel
		tv.onMouseButtonLeft.function = callback
	case gui.ReqOnMouseButtonRight:
		tv.onMouseButtonRight.channel = channel
		tv.onMouseButtonRight.function = callback
	default:
		return errors.NewGopherError(errors.UnknownGUIRequest, request)
	}

	return nil
}

// SetFeature is used to set a television attribute
func (tv *GUI) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	switch request {
	case gui.ReqSetVisibilityStable:
		err := tv.scr.stb.resolveSetVisibilityStable()
		if err != nil {
			return err
		}

	case gui.ReqSetVisibility:
		if args[0].(bool) {
			tv.scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				tv.update()
			}
		} else {
			tv.scr.window.Hide()
		}

	case gui.ReqSetAllowDebugging:
		tv.setDebugging(args[0].(bool))
		tv.update()

	case gui.ReqSetPause:
		tv.paused = args[0].(bool)
		tv.update()

	case gui.ReqSetMasking:
		tv.scr.setMasking(args[0].(bool))
		tv.update()

	case gui.ReqToggleMasking:
		tv.scr.setMasking(!tv.scr.unmasked)
		tv.update()

	case gui.ReqSetAltColors:
		tv.scr.useAltPixels = args[0].(bool)
		tv.update()

	case gui.ReqToggleAltColors:
		tv.scr.useAltPixels = !tv.scr.useAltPixels
		tv.update()

	case gui.ReqSetShowSystemState:
		tv.scr.showSystemState = args[0].(bool)
		tv.update()

	case gui.ReqToggleShowSystemState:
		tv.scr.showSystemState = !tv.scr.showSystemState
		tv.update()

	case gui.ReqSetScale:
		tv.scr.setScaling(args[0].(float32))
		tv.update()

	case gui.ReqIncScale:
		if tv.scr.pixelScale < 4.0 {
			tv.scr.setScaling(tv.scr.pixelScale + 0.1)
			tv.update()
		}

	case gui.ReqDecScale:
		if tv.scr.pixelScale > 0.5 {
			tv.scr.setScaling(tv.scr.pixelScale - 0.1)
			tv.update()
		}

	default:
		return errors.NewGopherError(errors.UnknownGUIRequest, request)
	}

	return nil
}
