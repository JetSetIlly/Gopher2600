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

	var err error

	visibleTop, err := stb.scr.gtv.GetState(television.ReqVisibleTop)
	if err != nil {
		return err
	}

	visibleBottom, err := stb.scr.gtv.GetState(television.ReqVisibleBottom)
	if err != nil {
		return err
	}

	// update play height (which in turn updates masking and window size)
	if stb.count < stabilityThreshold {
		if stb.visibleTopReference != int32(visibleTop) || stb.visibleBottomReference != int32(visibleBottom) {
			stb.visibleTopReference = int32(visibleTop)
			stb.visibleBottomReference = int32(visibleBottom)
			stb.count = 0
		} else {
			stb.count++
		}
	} else if stb.count == stabilityThreshold {
		stb.count++

		// calculate the play height from the top and bottom values with a
		// minimum according to the tv specification
		height := int32(stb.visibleBottomReference - stb.visibleTopReference)
		minHeight := int32(stb.scr.gtv.GetSpec().ScanlinesPerVisible)
		if height < minHeight {
			height = minHeight
		}

		err := stb.scr.setPlayHeight(height, int32(stb.visibleTopReference))
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
	}

	return nil
}

func (stb *screenStabiliser) resolveSetVisibilityStable() error {
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
