package sdl

import (
	"gopher2600/gui"
	"gopher2600/television"
)

// the purpose of the stability check is to prevent the window opening and then
// resizing after the initialisation sequence of the ROM. by giving the ROM
// time to settle down and produce frames with a consistent number of
// scanlines we prevent the window from flapping about too much.

type screenStabiliser struct {
	// the screen which is being stabilzed
	scr *screen

	// how many count have been observed that look like they might be stable?
	count int

	top       int
	bottom    int
	scanlines int

	// has a ReqSetVisibilityStable been received recently? we don't want to
	// open the window until the screen is stable
	queuedShowRequest bool
}

func newScreenStabiliser(scr *screen) *screenStabiliser {
	stb := new(screenStabiliser)
	stb.scr = scr
	return stb
}

// number of consistent frames that needs to elapse before the screen is
// considered "stable". the value has been set arbitrarily, a more
// sophisticated approach may be worth investigating. for now, the lower the
// value the better.
const stabilityThreshold int = 2

// restart resets the stability count to zero thereby forcing the play area to
// be reconsidered
func (stb *screenStabiliser) restart() {
	stb.count = 0
}

// stabiliseFrame checks to see if the screen dimensions have been stable for
// a count of "stabilityThreshold"
//
// currently: once it's been determined that the screen dimensions are stable
// then any changes are ignored
func (stb *screenStabiliser) stabiliseFrame() error {
	// measures the consistency of the generated television frame and alters
	// window sizing appropriately

	var err error

	top, err := stb.scr.gtv.GetState(television.ReqVisibleTop)
	if err != nil {
		return err
	}

	bottom, err := stb.scr.gtv.GetState(television.ReqVisibleBottom)
	if err != nil {
		return err
	}

	scanlines := bottom - top

	// update play height (which in turn updates masking and window size)
	if stb.count < stabilityThreshold {
		if stb.top != top || stb.bottom != bottom {
			stb.top = top
			stb.bottom = bottom
			stb.scanlines = bottom - top
			stb.count = 0
		} else {
			stb.count++
		}
	} else if stb.count == stabilityThreshold {
		stb.count++

		// calculate the play height from the top and bottom values with a
		// minimum according to the tv specification
		minScanlines := stb.scr.gtv.GetSpec().ScanlinesPerVisible
		if scanlines < minScanlines {
			scanlines = minScanlines
		}

		err := stb.scr.setPlayArea(int32(scanlines), int32(stb.top))
		if err != nil {
			return err
		}

		// show window if a show request has been queued up
		if stb.queuedShowRequest {
			err := stb.resolveSetVisibility()
			if err != nil {
				return err
			}
		}
	} else {
		// some ROMs turn VBLANK on/off at different times (no more than a
		// scanline or two I would say) but maintain the number of scanlines in
		// the visiible area. in these instances, because of how we've
		// implemented play area masking in the SDL interface, we need to
		// adjust the play area.
		//
		// ROMs affected:
		//	* Plaque Attack
		//
		// some other ROMs turn VBLANK on/off at different time but also allow
		// the number of scanlines to change. in these instances, we do not
		// make the play area adjustment.
		//
		// ROMs (not) affected:
		//  * 28c3intro
		//
		if scanlines == stb.scanlines && stb.top != top {
			stb.scr.adjustPlayArea(int32(top - stb.top))
			stb.top = top
			stb.bottom = bottom
		}
	}

	return nil
}

func (stb *screenStabiliser) resolveSetVisibility() error {
	if stb.count > stabilityThreshold {
		err := stb.scr.gtv.SetFeature(gui.ReqSetVisibility, true, true)
		if err != nil {
			return err
		}
		stb.queuedShowRequest = false
	} else {
		stb.queuedShowRequest = true
	}
	return nil
}
