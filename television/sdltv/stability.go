package sdltv

import (
	"gopher2600/television"
)

// the purpose of the stability check is to prevent the window opening and then
// resizing after the initialisation sequence of the ROM. by giving the ROM
// time to settle down and produce frames with a consistent number of
// scanlines, we prevent the window from flapping about in response to the
// changes in scanline count.

type screenStabiliser struct {
	// the screen which is being stabilzed
	scr *screen

	// how many count have been observed that look like they might be stable?
	count int

	// the current number of (stable) visible scanlines. only changes once the
	// frame is considered stable
	visibleScanlines int32

	// the scanline number of the first visible scanline. this is currently
	// defined to be the scanline at which VBlank is turned off when the image
	// first passes the stability threshold. it is used to adjust the viewport
	// for wobbly frames. see "shift viewport" comment below.
	visibleTopReference    int32
	visibleBottomReference int32

	// has a ReqSetVisibilityStable been received recently? we don't want to
	// open the window until the screen is stable
	queuedShowRequest bool

	// record of how many scanlines the viewport has been shifted
	viewportShift int32
}

func newScreenStabiliser(scr *screen) *screenStabiliser {
	stb := new(screenStabiliser)
	stb.scr = scr
	return stb
}

// number of consistent frames that needs to elapse before the screen is
// considered "stable" -- this value has been set arbitrarily. a more
// sophisticated approach may be worth investigating
const stabilityThreshold int = 6

// checkStableFrame checks to see if the screen dimensions have been stable for
// a count of "stabilityThreshold"
//
// currently: once it's been determined that the screen dimensions are stable
// then any changes are ignored
func (stb *screenStabiliser) checkStableFrame() error {
	// measures the consistency of the generated television frame and alters
	// window sizing appropriately
	if !stb.isStable() {
		if stb.visibleTopReference == int32(stb.scr.tv.VisibleTop) &&
			stb.visibleBottomReference == int32(stb.scr.tv.VisibleBottom) {

			if stb.count < stabilityThreshold {
				stb.count++
			} else if stb.count == stabilityThreshold {
				stb.count++

				// update play height (which in turn updates masking and window size)
				err := stb.scr.setPlayHeight(int32(stb.visibleScanlines), int32(stb.visibleTopReference))
				if err != nil {
					return err
				}

				// show window if a show request has been queued up
				if stb.queuedShowRequest {
					err := stb.resolveSetVisibilityStable()
					if err != nil {
						return err
					}
				}
			} else {
				// stability hasn't been reached yet so reset count
				stb.count = 0
			}
		} else {
			stb.visibleTopReference = int32(stb.scr.tv.VisibleTop)
			stb.visibleBottomReference = int32(stb.scr.tv.VisibleBottom)
			stb.visibleScanlines = int32(stb.visibleBottomReference - stb.visibleTopReference)
		}
	}

	return nil
}

// beginViewportStabilisation should be called at beginning of frame update. note that
// it should also be paired with endViewportStabilisation, called at the end of the
// frame upate
func (stb *screenStabiliser) beginViewportStabilisation() error {
	// shift viewport: this is a fix for Plaq Attack although other ROMs could
	// feasibly have the same problem. Plaq Attack has an inconsistent number
	// of VBLank lines at the start of the frame but the same number of visible
	// scanlines throughout. the following adjusts the src/dest rectangles to
	// account for the difference.
	//
	// (note that this shift will bugger up scanline reporting when using the
	// right mouse button facility. if screen is unmasked however, then the
	// reporting will be correct)

	// do nothing if VisibleTop is -1
	if stb.scr.tv.VisibleTop != -1 {
		stb.viewportShift = int32(stb.scr.tv.VisibleTop) - stb.visibleTopReference
		stb.scr.srcRect.Y += stb.viewportShift
	}

	return nil
}

// endViewportStabilsation should be called at the end of a frame update (assuming
// beginViewportStabilisation was called at the beginning of the update)
func (stb *screenStabiliser) endViewportStabilisation() error {
	// undo viewport shift
	stb.scr.srcRect.Y -= stb.viewportShift

	return nil
}

func (stb *screenStabiliser) isStable() bool {
	return stb.count > stabilityThreshold
}

func (stb *screenStabiliser) resolveSetVisibilityStable() error {
	if stb.isStable() {
		err := stb.scr.tv.SetFeature(television.ReqSetVisibility, true, true)
		if err != nil {
			return err
		}
		stb.queuedShowRequest = false
	} else {
		stb.queuedShowRequest = true
	}
	return nil
}
