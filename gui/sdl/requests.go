package sdl

import (
	"gopher2600/errors"
	"gopher2600/gui"
)

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

	case gui.ReqSetShowMetaPixels:
		tv.scr.showMetaPixels = args[0].(bool)
		tv.update()

	case gui.ReqToggleShowMetaPixels:
		tv.scr.showMetaPixels = !tv.scr.showMetaPixels
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
		return errors.NewFormattedError(errors.UnknownGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (tv *GUI) SetEventChannel(eventChannel chan gui.Event) {
	tv.eventChannel = eventChannel
}
