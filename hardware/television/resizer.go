// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package television

import (
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// resizer handles the expansion of the visible area of the TV screen
//
// ROMs used to test resizing:
//  * good base cases
//		- Pitfall
//		- Hero
//
//  * needs an additional (or two) scanlines to accommodate full screen
//		- Ladybug
//		- Man Goes Down
//
//  * frame that needs to be resized after startup period
//		- Hack Em Hanglyman (pre-release)
//
//  * the occasional unsynced frame
//		- Hack Em Hanglyman (pre-release)
//
//  * lots of unsynced frames (during computer "thinking" period)
//		- Andrew Davies' Chess
//
//  * does not set VBLANK for pixels that are clearly not meant to be seen
//  these ROMs rely on the SafeTop and SafeBottom values being correct
//		- Communist Mutants From Space
//		- Tapper
//		- Spike's Peak
//
//	* ROMs that do not set VBLANK *at all*. in these cases the commit()
//	function uses the black top/bottom values rather than vblank top/bottom
//	values.
//		- Hack Em Hanglyman (release and pre-release)
//		- Legacy of the Beast
//
//	* ROMs that *do* set VBLANK but might be caught by the black top/bottom
//	rule if frameHasVBlank was incorrectly set
//		- aTaRSI (demo)
//		- Supercharger "rewind tape" screen
//
type resizer struct {
	// candidate top/bottom values for an actual resize.
	//
	// updated during the examine phase if the tv image goes beyond the current
	// stable top/bottom vaulues
	//
	// vblankTop/vblankBottom records the extent of the negative VBLANK signal
	//
	// blackTop/blackBottom records the extent of black pixels.
	vblankTop    int
	vblankBottom int
	blackTop     int
	blackBottom  int

	// whether the frame has a vblank. if this value is negative the
	// blackTop/blackBottom values are the candidate values used in the
	// commit() function
	frameHasVBlank bool

	// the top/bottom values that will become the new stable top/bottom values
	// once pendingCt has reached zero.
	//
	// update during the commit() function if current top/bottom values differ
	// to the pending values.
	//
	// in a stable image, pending top/bottom will be equal to stable top/bottom
	// meaning that by induction will also equal current top/bottom.
	pendingTop    int
	pendingBottom int

	// number of frames until a resize is committed to the PixelRenderers this
	// gives time for the screen to settle down.
	pendingCt int
}

// set resizer's top/bottom values to equal tv top/bottom values.
func (sr *resizer) initialise(tv *Television) {
	sr.vblankTop = tv.state.frameInfo.VisibleTop
	sr.vblankBottom = tv.state.frameInfo.VisibleBottom
	sr.pendingTop = tv.state.frameInfo.VisibleTop
	sr.pendingBottom = tv.state.frameInfo.VisibleBottom
}

// examine signal for resizing possiblity. this is an expensive operation to do
// for every single signal/pixel. should probably be throttled in some way.
func (sr *resizer) examine(tv *Television, sig signal.SignalAttributes) {
	// do not try to resize during frame that isn't "vsynced".
	//
	// the best example of this is Andrew Davie's chess which simply does
	// not care about frames during the computer's thinking time - we don't
	// want to resize during these frames.
	if !tv.state.frameInfo.VSync {
		// reset any pending changes on an unsynced frame
		sr.pendingCt = 0
		sr.pendingTop = sr.vblankTop
		sr.pendingBottom = sr.vblankBottom
		sr.frameHasVBlank = false
		return
	}

	sr.frameHasVBlank = sr.frameHasVBlank || sig&signal.VBlank == signal.VBlank

	// if VBLANK is off at any point after than HBLANK period then note the
	// change in current top/bottom if appropriate
	if tv.state.clock > specification.ClksHBlank && sig&signal.VBlank != signal.VBlank {
		// update current top/bottom values
		//
		// comparing against current top/bottom scanline, rather than ideal
		// top/bottom scanline of the specification. this means that a screen will
		// never "shrink" until the specification is changed either manually or
		// automatically.
		//
		// limit the top/bottom scanlines to a safe area. the atari safe area
		// is too conservative but equally we don't ever want the entire screen
		// to be visible. the new safetop and safebottom values are defined in
		// our TV specifications.
		if tv.state.scanline < sr.vblankTop && tv.state.scanline >= tv.state.frameInfo.Spec.NewSafeVisibleTop {
			sr.vblankTop = tv.state.scanline
		} else if tv.state.scanline > sr.vblankBottom && tv.state.scanline <= tv.state.frameInfo.Spec.NewSafeVisibleBottom {
			sr.vblankBottom = tv.state.scanline
		}
	}

	// some ROMs never set VBLANK. for these cases we also record the extent of
	// non-black pixels. these values are using in the commit() function in the
	// event that frameHasVBlank is false.
	if tv.state.clock > specification.ClksHBlank && sig&signal.VBlank != signal.VBlank && sig&signal.Color != 0 {
		if tv.state.scanline < sr.blackTop && tv.state.scanline >= tv.state.frameInfo.Spec.NewSafeVisibleTop {
			sr.blackTop = tv.state.scanline
		} else if tv.state.scanline > sr.blackBottom && tv.state.scanline <= tv.state.frameInfo.Spec.NewSafeVisibleBottom {
			sr.blackBottom = tv.state.scanline
		}
	}
}

// some ROMs will want to resize every frame if allowed. this is ugly so we
// slow it down by counting from framesUntilResize down to zero. the resize
// will only be committed (ie. the actual top/bottom values changed to match
// the resize top/bottom value) at that point
//
// the counter will be reset if the screen size changes in the interim.
const framesUntilResize = 2

// commit resizing possibility found through examine() function.
func (sr *resizer) commit(tv *Television) error {
	// only commit on even frames. the only reason we do this is to catch
	// flicker kernels where pixels are different every frame. this is a bit of
	// a pathological situation but it does happen so we should handle it
	//
	// an example of this is the CDFJ QBert demo ROM
	//
	// note that this means the framesUntilResize value is effectively double
	// that value stated
	if tv.state.frameNum%2 == 0 {
		return nil
	}

	// make sure candidate top and bottom value are equal to stable top/bottom
	// at beginning of a frame
	defer func() {
		sr.vblankTop = tv.state.frameInfo.VisibleTop
		sr.vblankBottom = tv.state.frameInfo.VisibleBottom
		sr.blackTop = tv.state.frameInfo.VisibleTop
		sr.blackBottom = tv.state.frameInfo.VisibleBottom
		sr.frameHasVBlank = false
	}()

	// if top/bottom values this frame are not the same as pending top/bottom
	// values then update pending values and reset pending counter.
	//
	// the value frameHasVBlank is used to decide which candidate values to
	// use: vblankTop/vblankBottom of blackTop/blackBottom
	if sr.frameHasVBlank {
		if sr.pendingTop != sr.vblankTop {
			sr.pendingTop = sr.vblankTop
			sr.pendingCt = framesUntilResize
		}
		if sr.pendingBottom != sr.vblankBottom {
			sr.pendingBottom = sr.vblankBottom
			sr.pendingCt = framesUntilResize
		}
	} else {
		if sr.pendingTop != sr.blackTop {
			sr.pendingTop = sr.blackTop
			sr.pendingCt = framesUntilResize
		}
		if sr.pendingBottom != sr.blackBottom {
			sr.pendingBottom = sr.blackBottom
			sr.pendingCt = framesUntilResize
		}
	}

	// do nothing if counter is zero
	if sr.pendingCt == 0 {
		return nil
	}

	// if pending top/bottom find themselves back at the stable top/bottom
	// values then there is no need to do anything.
	if sr.pendingTop == tv.state.frameInfo.VisibleTop && sr.pendingBottom == tv.state.frameInfo.VisibleBottom {
		sr.pendingCt = 0
		return nil
	}

	// reduce pending counter every frame that is active
	sr.pendingCt--

	// do nothing if counter is not yet zero
	if sr.pendingCt > 0 {
		return nil
	}

	// return if there's nothing to do
	if sr.pendingBottom == tv.state.frameInfo.VisibleBottom && sr.pendingBottom == tv.state.frameInfo.VisibleTop {
		return nil
	}

	// sanity check before we do anything drastic
	if tv.state.frameInfo.VisibleTop < tv.state.frameInfo.VisibleBottom {
		// add one to the bottom value before committing. Ladybug and Hack'Em
		// Pacman are good examples of ROMs that are "wrong" if we don't do
		// this
		sr.pendingBottom++

		// add another one. Man Down is an example of ROM which is "wrong"
		// without an additional (two) scanlines after the VBLANK
		sr.pendingBottom++

		// !!TODO: more elegant way of handling the additional scanline problem

		// clamp bottom scanline to safe bottom
		if sr.pendingBottom > tv.state.frameInfo.Spec.NewSafeVisibleBottom {
			sr.pendingBottom = tv.state.frameInfo.Spec.NewSafeVisibleBottom
		}

		// update statble top/bottom values
		tv.state.frameInfo.VisibleTop = sr.pendingTop
		tv.state.frameInfo.VisibleBottom = sr.pendingBottom

		// call Resize() for all attached pixel renderers
		for f := range tv.renderers {
			err := tv.renderers[f].Resize(tv.state.frameInfo)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
