// television interface implementation - SDLTV has an embedded HeadlessTV so
// much of the interface is implementated there.

package sdltv

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/television"
)

// GetState returns the TVState object for the named state
func (tv *SDLTV) GetState(request television.StateReq) (television.TVState, error) {
	return tv.HeadlessTV.GetState(request)
}

// GetMetaState returns the TVState object for the named state
func (tv *SDLTV) GetMetaState(request television.MetaStateReq) (string, error) {
	state, err := tv.HeadlessTV.GetMetaState(request)
	switch err := err.(type) {
	case errors.GopherError:
		if err.Errno != errors.UnknownTVRequest {
			return state, err
		}
	default:
		return state, err
	}

	switch request {
	case television.ReqLastMouse:
		return fmt.Sprintf("mouse: hp=%d, sl=%d", tv.lastMouseHorizPos, tv.lastMouseScanline), nil
	case television.ReqLastMouseHorizPos:
		return fmt.Sprintf("%d", tv.lastMouseHorizPos), nil
	case television.ReqLastMouseScanline:
		return fmt.Sprintf("%d", tv.lastMouseScanline), nil
	default:
		return "", errors.NewGopherError(errors.UnknownTVRequest, request)
	}
}

// RegisterCallback implements Television interface
func (tv *SDLTV) RegisterCallback(request television.CallbackReq, channel chan func(), callback func()) error {
	// call embedded implementation and filter out UnknownCallbackRequests
	err := tv.HeadlessTV.RegisterCallback(request, channel, callback)
	switch err := err.(type) {
	case errors.GopherError:
		if err.Errno != errors.UnknownTVRequest {
			return err
		}
	default:
		return err
	}

	switch request {
	case television.ReqOnWindowClose:
		tv.onWindowClose.channel = channel
		tv.onWindowClose.function = callback
	case television.ReqOnMouseButtonLeft:
		tv.onMouseButtonLeft.channel = channel
		tv.onMouseButtonLeft.function = callback
	case television.ReqOnMouseButtonRight:
		tv.onMouseButtonRight.channel = channel
		tv.onMouseButtonRight.function = callback
	default:
		return errors.NewGopherError(errors.UnknownTVRequest, request)
	}

	return nil
}

// SetFeature is used to set a television attribute
func (tv *SDLTV) SetFeature(request television.FeatureReq, args ...interface{}) error {
	err := tv.HeadlessTV.SetFeature(request)
	switch err := err.(type) {
	case errors.GopherError:
		if err.Errno != errors.UnknownTVRequest {
			return err
		}
	default:
		return err
	}

	switch request {
	case television.ReqSetVisibilityStable:
		err = tv.scr.stabiliser.resolveSetVisibilityStable()
		if err != nil {
			return err
		}

	case television.ReqSetVisibility:
		if args[0].(bool) {
			tv.scr.window.Show()

			// default args[1] of true if not present
			if len(args) < 2 || args[1].(bool) {
				tv.update()
			}
		} else {
			tv.scr.window.Hide()
		}

	case television.ReqSetPause:
		tv.guiLoopLock.Lock()
		tv.paused = args[0].(bool)
		tv.guiLoopLock.Unlock()
		if args[0].(bool) {
			tv.update()
		}

	case television.ReqSetDebug:
		tv.guiLoopLock.Lock()
		tv.scr.setMasking(args[0].(bool))
		tv.guiLoopLock.Unlock()

	case television.ReqSetScale:
		tv.guiLoopLock.Lock()
		tv.scr.setScaling(args[0].(float32))
		tv.guiLoopLock.Unlock()

	default:
		return errors.NewGopherError(errors.UnknownTVRequest, request)
	}

	return nil
}
