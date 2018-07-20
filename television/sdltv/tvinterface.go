// television interface implementation - SDLTV has an embedded HeadlessTV so
// much of the interface is implementated there.

package sdltv

import (
	"gopher2600/errors"
	"gopher2600/television"
)

// list of addtiinal callback register requests for SDL television
// TODO: gui related callback requests should be standardised accross all gui
// implementations.
const (
	ReqOnWindowClose television.CallbackReq = "ONWINDOWCLOSE"
)

// Signal is the principle method of communication between the VCS and
// televsion. note that most of the work is done in the embedded HeadlessTV
// instance
func (tv *SDLTV) Signal(attr television.SignalAttributes) {
	tv.HeadlessTV.Signal(attr)

	// *CRITICAL SECTION*
	// (R) tv.scr, tv.dbgScr
	tv.guiLoopLock.Lock()
	defer tv.guiLoopLock.Unlock()

	guiDbgScr := tv.scr == tv.dbgScr

	if tv.Phosphor || guiDbgScr {
		// decode color
		r, g, b := byte(0), byte(0), byte(0)
		if attr.Pixel <= 256 {
			col := tv.Spec.Colors[attr.Pixel]
			r, g, b = byte((col&0xff0000)>>16), byte((col&0xff00)>>8), byte(col&0xff)
		}
		tv.setPixel(int32(tv.PixelX(!guiDbgScr)), int32(tv.PixelY(!guiDbgScr)), r, g, b, tv.scr.pixels)
	}
}

// SetVisibility toggles the visiblity of the SDLTV window
func (tv *SDLTV) SetVisibility(visible bool) error {
	// *NON-CRITICAL SECTION* called from guiLoop but SDL handles its own
	// concurrency conflicts

	if visible {
		tv.window.Show()
	} else {
		tv.window.Hide()
	}
	return nil
}

// SetPause toggles whether the tv is currently being updated. we can use this
// when we pause the emulation to make sure we aren't left with a blank screen
func (tv *SDLTV) SetPause(pause bool) error {
	if pause {
		tv.paused = true
		tv.update()
	} else {
		tv.paused = false
	}
	return nil
}

// RegisterCallback implements Television interface
func (tv *SDLTV) RegisterCallback(request television.CallbackReq, callback func()) error {
	// call embedded implementation and filter out UnknownCallbackRequests
	err := tv.HeadlessTV.RegisterCallback(request, callback)
	switch err := err.(type) {
	case errors.GopherError:
		if err.Errno != errors.UnknownCallbackRequest {
			return err
		}
	default:
		return err
	}

	switch request {
	case ReqOnWindowClose:
		// * CRITICAL SEECTION*
		// (W) tv.onWindowClose
		tv.guiLoopLock.Lock()
		tv.onWindowClose = callback
		tv.guiLoopLock.Unlock()
	default:
		return errors.GopherError{Errno: errors.UnknownCallbackRequest, Values: errors.Values{request}}
	}

	return nil
}
