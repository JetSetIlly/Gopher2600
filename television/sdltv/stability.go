package sdltv

import (
	"gopher2600/television"
)

type stability struct {
	// how many frames have been observed that look like they might be stable?
	count int

	// the current number of (stable) visible scanlines. only changes once the
	// frame is considered stable
	currVisibleScanlines int

	// has a ReqSetVisibilityStable been received recently?
	showRequest bool
}

// number of frames that needs to elapse before the screen is considered "stable"
const framesForStability int = 5

// checkStability measures the consistency of the generated television frame
// and alters window sizing and visibility when necessary. this usually only
// plays a significant role during ROM startup.
func (scr *screen) checkStability() error {
	if scr.tv.VBlankOn-scr.tv.VBlankOff == scr.stability.currVisibleScanlines {
		if scr.stability.count < framesForStability {
			scr.stability.count++

		} else if scr.stability.count == framesForStability {
			scr.stability.count++

			// update play window dimenstions
			scr.setPlayHeight(int32(scr.stability.currVisibleScanlines))

			// show window if a show request has been queued up
			if scr.stability.showRequest {
				err := scr.tv.RequestSetAttr(television.ReqSetVisibility, true, false)
				if err != nil {
					return err
				}
				scr.stability.showRequest = false
			}
		}
	} else {
		// number of visible lines has changed
		scr.stability.count = 0
		scr.stability.currVisibleScanlines = scr.tv.VBlankOn - scr.tv.VBlankOff
	}

	return nil
}
