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
//  *  changes size after setup phase
//		- Ladybug
//
//  * as above + it throws in an unsynced frame every now and again
//		- Hack Em Hangly Pacman
//
//  * lots of unsynced frames during computer "thinking" period
//		- Andrew Davies' Chess
//
//	* unsynced frames every other frame
//		- Mega Bitmap Demo (atext.bin)
//
//  * does not set VBLANK for pixels that are clearly not meant to be seen
//  these ROMs rely on the NewSafeTop and NewSafeBottom values
//		- Communist Mutants From Space
//		- Tapper
//		- Spike's Peak
//
//   * does not set VBLANK but we can crop more aggressively by assuming that a scanline
//   consisting only of black pixels should not be seen
//		- Legacy of the Beast
//
//	 * test resizing counter
//		- Supercharger BIOS (resizes excessively due to moving starfield)
type resizer struct {
	// the stable top/bottom values. what the resized frame actually is. these
	// are the values that the PixelRenderers should consider to be the visible
	// range.
	top    int
	bottom int

	// top/bottom values for current frame.
	//
	// updated during the examine phase if the tv image goes beyond the current
	// stable top/bottom vaulues.
	//
	// these values are used to decide whether to start the pending resize
	// counter.
	currTop    int
	currBottom int

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
	sr.top = tv.state.top
	sr.bottom = tv.state.bottom
	sr.currTop = tv.state.top
	sr.currBottom = tv.state.bottom
	sr.pendingTop = tv.state.top
	sr.pendingBottom = tv.state.bottom
}

func (sr *resizer) examine(tv *Television, sig signal.SignalAttributes) {
	// ignore any frame that isn't "synced" is also not allowed to resize the
	// TV. the best example of this is Andrew Davie's chess which simply does
	// not care about frames during the computer's thinking time.
	//
	// the "mega bitmap demo" (atext.bin) is by comparison is a ROM that spits
	// out unsynced frames every other frame
	if !tv.state.syncedFrame {
		return
	}

	// if vblank is off at any point after than HBLANK period then note the
	// change in current top/bottom if appropriate
	if tv.state.clock > specification.ClksHBlank && !sig.VBlank && sig.Pixel > 0 {
		// update current top/bottom values
		//
		// comparing against current top/bottom scanline, rather than ideal
		// top/bottom scanline of the specification. this means that a screen will
		// never "shrink" until the specification is changed either manually or
		// automatically.
		//
		// we also limit to the top/bottom scanlines to a safe area. the atari
		// safe area is too conservative so we've defined our own.
		if tv.state.scanline < sr.currTop && tv.state.scanline >= tv.state.spec.NewSafeTop {
			sr.currTop = tv.state.scanline
		} else if tv.state.scanline > sr.currBottom && tv.state.scanline <= tv.state.spec.NewSafeBottom {
			sr.currBottom = tv.state.scanline
		}
	}
}

// some ROMs will want to resize every frame if allowed. this is ugly so we
// slow it down by counting from framesUntilResize down to zero. the resize
// will only be committed (ie. the actual top/bottom values changed to match
// the resize top/bottom value) when it doe reach zero.
//
// the counter will be reset if the screen size changes in the interim.
const framesUntilResize = 2

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

	// make sure current top and current bottom are always equal to stable
	// top/bottom at beginning of a frame
	defer func() {
		sr.currTop = sr.top
		sr.currBottom = sr.bottom
	}()

	// if top/bottom values this frame are not the same as pending top/bottom
	// values then update pending values and reset pending counter.
	//
	// note that unlike the expansion of current top and bottom value we allow
	// shrinkage of pending top and bottom values
	if sr.currTop != sr.pendingTop {
		sr.pendingTop = sr.currTop
		sr.pendingCt = framesUntilResize
	}
	if sr.currBottom != sr.pendingBottom {
		sr.pendingBottom = sr.currBottom
		sr.pendingCt = framesUntilResize
	}

	// do nothing if counter is zero
	if sr.pendingCt == 0 {
		return nil
	}

	// if pending top/bottom find themselves back at the stable top/bottom
	// values then there is no need to do anything.
	if sr.pendingTop == sr.top && sr.pendingBottom == sr.bottom {
		sr.pendingCt = 0
		return nil
	}

	// reduce pending counter every frame that is active
	sr.pendingCt--

	// do nothing if counter is not yet zero
	if sr.pendingCt > 0 {
		return nil
	}

	// commit pending values
	sr.top = sr.pendingTop
	sr.bottom = sr.pendingBottom

	// return if there's nothing to do
	if sr.bottom == tv.state.bottom && sr.top == tv.state.top {
		return nil
	}

	// sanity check before we do anything drastic
	if tv.state.top < tv.state.bottom {
		// add one to the bottom value before committing. Ladybug and Hack'Em
		// Pacman are good examples of ROMs that are "wrong" if we don't do
		// this
		sr.bottom++

		// add another one. Man Down is an example of ROM which is "wrong"
		// without an additional (two) scanlines after the VBLANK
		sr.bottom++

		// clamp bottom scanline to safe bottom
		if sr.bottom > tv.state.spec.NewSafeBottom {
			sr.bottom = tv.state.spec.NewSafeBottom
		}

		// TODO: more elegant way of handling the additional scanline problem

		// update statble top/bottom values
		tv.state.top = sr.top
		tv.state.bottom = sr.bottom

		// call Resize() for all attached pixel renderers
		for f := range tv.renderers {
			err := tv.renderers[f].Resize(tv.state.spec, tv.state.top, tv.state.bottom)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
