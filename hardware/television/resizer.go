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
	top    int
	bottom int

	// number of frames until a resize can take place
	counter int
}

// some ROMs will want to resize every frame if allowed. this is ugly so we
// slow it down by counting from framesUntilResize down to zero. the resize
// will only be committed (ie. the actual top/bottom values changed to match
// the resize top/bottom value) when it doe reach zero.
//
// the counter will be reset if the screen size changes in the interim.
const framesUntilResize = 5

// set resizer's top/bottom values to equal tv top/bottom values.
func (sr *resizer) initialise(tv *Television) {
	sr.top = tv.state.top
	sr.bottom = tv.state.bottom
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

	// if vblank is off at any point after than HBLANK period then tentatively
	// extend the top/bottom of the screen. we'll commit the resize procedure
	// in the newFrame() function when sr.counter reaches 0.
	if tv.state.horizPos > specification.HorizClksHBlank && !sig.VBlank && sig.Pixel > 0 {
		// comparing against current top/bottom scanline, rather than ideal
		// top/bottom scanline of the specification. this means that a screen will
		// never "shrink" until the specification is changed either manually or
		// automatically.
		//
		// we also limit to the top/bottom scanlines to a safe area. the atari
		// safe area is too conservative so we've defined our own.
		if tv.state.scanline < sr.top && tv.state.scanline >= tv.state.spec.NewSafeTop {
			sr.top = tv.state.scanline
			sr.counter = framesUntilResize
		} else if tv.state.scanline > sr.bottom && tv.state.scanline <= tv.state.spec.NewSafeBottom {
			sr.bottom = tv.state.scanline
			sr.counter = framesUntilResize
		}
	}
}

func (sr *resizer) commit(tv *Television) error {
	sr.counter--
	if sr.counter > 0 {
		return nil
	}

	// do not allow resizing to take place for the first few frames of a ROM.
	// these frames tend to be set up frames and can be wildly unstable.
	if tv.state.syncedFrameNum <= leadingFrames {
		return nil
	}

	// return if there's nothing to do
	if sr.bottom == tv.state.bottom && sr.top == tv.state.top {
		return nil
	}

	// something has changed so call Resize() for all attached pixel renderers
	if tv.state.top < tv.state.bottom {
		// update real top value
		tv.state.top = sr.top

		// add one to the bottom value before committing. we shouldn't need to
		// do this but some screens will end up being one scanline too short
		// without it. for example, Ladybug and Hack'Em Pacman
		sr.bottom++

		// update real bottom value
		tv.state.bottom = sr.bottom

		for f := range tv.renderers {
			err := tv.renderers[f].Resize(tv.state.spec, tv.state.top, tv.state.bottom-tv.state.top)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
