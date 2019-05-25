package sdl

import (
	"gopher2600/errors"
	"gopher2600/gui"
)

// SetFeature is used to set a television attribute
func (gtv *GUI) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	switch request {
	case gui.ReqSetVisibilityStable:
		err := gtv.scr.stb.resolveSetVisibility()
		if err != nil {
			return err
		}

	case gui.ReqSetVisibility:
		if args[0].(bool) {
			gtv.scr.window.Show()

			// update screen
			// -- default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				gtv.update()
			}
		} else {
			gtv.scr.window.Hide()
		}

	case gui.ReqSetAllowDebugging:
		gtv.setDebugging(args[0].(bool))
		gtv.update()

	case gui.ReqSetPause:
		gtv.paused = args[0].(bool)
		gtv.update()

	case gui.ReqSetMasking:
		gtv.scr.setMasking(args[0].(bool))
		gtv.update()

	case gui.ReqToggleMasking:
		gtv.scr.setMasking(!gtv.scr.unmasked)
		gtv.update()

	case gui.ReqSetAltColors:
		gtv.scr.useAltPixels = args[0].(bool)
		gtv.update()

	case gui.ReqToggleAltColors:
		gtv.scr.useAltPixels = !gtv.scr.useAltPixels
		gtv.update()

	case gui.ReqSetShowMetaPixels:
		gtv.scr.showMetaPixels = args[0].(bool)
		gtv.update()

	case gui.ReqToggleShowMetaPixels:
		gtv.scr.showMetaPixels = !gtv.scr.showMetaPixels
		gtv.update()

	case gui.ReqSetScale:
		gtv.scr.setScaling(args[0].(float32))
		gtv.update()

	case gui.ReqIncScale:
		if gtv.scr.pixelScale < 4.0 {
			gtv.scr.setScaling(gtv.scr.pixelScale + 0.1)
			gtv.update()
		}

	case gui.ReqDecScale:
		if gtv.scr.pixelScale > 0.5 {
			gtv.scr.setScaling(gtv.scr.pixelScale - 0.1)
			gtv.update()
		}

	default:
		return errors.NewFormattedError(errors.UnknownGUIRequest, request)
	}

	return nil
}

// SetEventChannel implements the GUI interface
func (gtv *GUI) SetEventChannel(eventChannel chan gui.Event) {
	gtv.eventChannel = eventChannel
}
