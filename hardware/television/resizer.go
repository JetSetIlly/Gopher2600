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
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// Resizer handles the expansion of the visible area of the TV screen
//
// ROMs used to test resizing:
//
// * good base cases
//   - Pitfall
//   - Hero
//
// * frame that needs to be resized after startup period
//   - Hack Em Hanglyman (pre-release)
//
// * the occasional unsynced frame
//   - Hack Em Hanglyman (pre-release)
//
// * lots of unsynced frames (during computer "thinking" period)
//   - Andrew Davies' Chess
//
// * does not set VBLANK for pixels that are clearly not meant to be seen
// these ROMs rely on the SafeTop and SafeBottom values being correct
//   - Communist Mutants From Space
//   - Tapper
//   - Spike's Peak
//
// * ROMs that do not set VBLANK *at all*. in these cases the commit()
// function uses the black top/bottom values rather than vblank top/bottom
// values.
//   - Hack Em Hanglyman (release and pre-release)
//   - Legacy of the Beast
//
// *ROMs that *do* set VBLANK but might be caught by the black top/bottom
// rule if usingVBLANK was incorrectly set
//   - aTaRSI (demo)
//   - Supercharger "rewind tape" screen
//
// * bottom of screen needs careful consideration
//   - Ladybug
//   - Man Goes Down
//
// finally, the following conditions are worth documenting as being important:
//
// * PAL ROMs without VSYNC cannot be sized or changed to the correct spec automatically
//   - Nightstalker
type Resizer struct {
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
	usingVBLANK bool

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

	// if resizer has been part of a preview emulation the preview fields
	// indicate from which number the resize information is valid
	//
	// if the current frame number is prior to the previewFrameNum then resizing
	// can be skipped
	previewFrameNum int
	previewStable   bool
}

func (rz *Resizer) String() string {
	if rz.pendingCt > 0 {
		return fmt.Sprintf("PENDING top: %d, bottom: %d", rz.pendingTop, rz.pendingBottom)
	} else if rz.usingVBLANK {
		return fmt.Sprintf("VBLANK top: %d, bottom: %d", rz.vblankTop, rz.vblankBottom)
	} else {
		return fmt.Sprintf("BLACK top: %d, bottom: %d", rz.blackTop, rz.blackBottom)
	}
}

func (rz *Resizer) reset(spec specification.Spec) {
	rz.setSpec(spec)
	rz.previewFrameNum = 0
}

// set resizer to nominal specification values
func (rz *Resizer) setSpec(spec specification.Spec) {
	rz.vblankTop = spec.IdealVisibleTop
	rz.vblankBottom = spec.IdealVisibleBottom
	rz.blackTop = spec.IdealVisibleTop
	rz.blackBottom = spec.IdealVisibleBottom
	rz.pendingTop = spec.IdealVisibleTop
	rz.pendingBottom = spec.IdealVisibleBottom
}

// examine signal for resizing possiblity. this is an expensive operation to do
// for every single signal/pixel. should probably be throttled in some way.
func (rz *Resizer) examine(state *State, sig signal.SignalAttributes) {
	// if current frame number is less that the validFrom field then do nothing
	if state.frameInfo.FrameNum < rz.previewFrameNum {
		return
	}

	// do not try to resize during frame that isn't "vsynced".
	//
	// the best example of this is Andrew Davie's chess which simply does
	// not care about frames during the computer's thinking time - we don't
	// want to resize during these frames.
	if !state.frameInfo.IsSynced {
		// reset any pending changes on an unsynced frame
		rz.pendingCt = 0
		rz.pendingTop = rz.vblankTop
		rz.pendingBottom = rz.vblankBottom
		return
	}

	// some ROMs never set VBLANK but we still want to do our best and frame the
	// screen nicely. this flag controls whether we use vblank top/bottom values
	// or "black" top/bottom values
	rz.usingVBLANK = rz.usingVBLANK || sig&signal.VBlank == signal.VBlank

	// if VBLANK is off then update the top/bottom values note
	if sig&signal.VBlank != signal.VBlank {
		if state.frameInfo.Stable {
			if state.scanline < rz.vblankTop &&
				state.scanline >= state.frameInfo.Spec.ExtendedVisibleTop {
				rz.vblankTop = state.scanline
			} else if state.scanline > rz.vblankBottom &&
				state.scanline <= state.frameInfo.Spec.ExtendedVisibleBottom {
				rz.vblankBottom = state.scanline
			}
		} else {
			// if television is not yet stable then the size can shrink as well
			// as grow. this is important for PAL60 ROMs which start off as PAL
			// sized but will shrink to NTSC size
			if state.scanline != rz.vblankTop &&
				state.scanline >= state.frameInfo.Spec.ExtendedVisibleTop &&
				state.scanline <= state.frameInfo.Spec.ExtendedVisibleTop+50 {
				rz.vblankTop = state.scanline
			} else if state.scanline != rz.vblankBottom &&
				state.scanline <= state.frameInfo.Spec.ExtendedVisibleBottom &&
				state.scanline >= state.frameInfo.Spec.ExtendedVisibleBottom-75 {
				rz.vblankBottom = state.scanline
			}
		}
	}

	// early return if frameHasVBLANK is true
	if rz.usingVBLANK {
		return
	}

	// black-pixel resizing requires a stable frame
	if !state.frameInfo.Stable {
		return
	}

	// some ROMs never set VBLANK. for these cases we also record the extent of
	// non-black pixels. these values are using in the commit() function in the
	// event that usingVBLANK is false.
	if state.clock > specification.ClksHBlank && sig&signal.VBlank != signal.VBlank {
		px := signal.ColorSignal((sig & signal.Color) >> signal.ColorShift)
		col := state.frameInfo.Spec.GetColor(px)
		if col.R != 0x00 || col.G != 0x00 || col.B != 0x00 {
			if state.frameInfo.Stable {
				if state.scanline < rz.blackTop &&
					state.scanline >= state.frameInfo.Spec.ExtendedVisibleTop {
					rz.blackTop = state.scanline
				} else if state.scanline > rz.blackBottom &&
					state.scanline <= state.frameInfo.Spec.ExtendedVisibleBottom {
					rz.blackBottom = state.scanline
				}
			}
		}
	}
}

// some ROMs will want to resize every frame if allowed. this is ugly so we
// slow it down by counting from framesUntilResize down to zero. the resize
// will only be committed (ie. the actual top/bottom values changed to match
// the resize top/bottom value) at that point
//
// the counter will be reset if the screen size changes in the interim.
//
// (27/04/24) increased to 3 to accomodate MovieCart when being rewound using
// the emulator's rewind system - the Moviecart rewind is unaffected by this
// value. it's unclear if there are any adverse affects to this increase but I
// don't believe there will be
const framesUntilResize = 3

// commit resizing possibility found through examine() function.
func (rz *Resizer) commit(state *State) error {
	// if current frame number is less that the validFrom field then do nothing
	if state.frameInfo.FrameNum < rz.previewFrameNum {
		return nil
	}

	// make sure candidate top and bottom value are equal to stable top/bottom
	// at beginning of a frame
	defer func() {
		rz.vblankTop = state.frameInfo.VisibleTop
		rz.vblankBottom = state.frameInfo.VisibleBottom
		rz.blackTop = state.frameInfo.VisibleTop
		rz.blackBottom = state.frameInfo.VisibleBottom
		rz.usingVBLANK = false
	}()

	// do not resize unless screen is synchronised
	if !state.vsync.isSynced() {
		return nil
	}

	// if top/bottom values this frame are not the same as pending top/bottom
	// values then update pending values and reset pending counter.
	//
	// the value usingVBLANK is used to decide which candidate values to
	// use: vblankTop/vblankBottom of blackTop/blackBottom
	if rz.usingVBLANK {
		if rz.pendingTop != rz.vblankTop {
			rz.pendingTop = rz.vblankTop
			rz.pendingCt = framesUntilResize
		}
		if rz.pendingBottom != rz.vblankBottom {
			rz.pendingBottom = rz.vblankBottom
			rz.pendingCt = framesUntilResize
		}
	} else {
		if rz.pendingTop != rz.blackTop {
			rz.pendingTop = rz.blackTop
			rz.pendingCt = framesUntilResize
		}
		if rz.pendingBottom != rz.blackBottom {
			rz.pendingBottom = rz.blackBottom
			rz.pendingCt = framesUntilResize
		}
	}

	// do nothing if counter is zero
	if rz.pendingCt == 0 {
		return nil
	}

	// if pending top/bottom find themselves back at the stable top/bottom
	// values then there is no need to do anything.
	if rz.pendingTop == state.frameInfo.VisibleTop && rz.pendingBottom == state.frameInfo.VisibleBottom {
		rz.pendingCt = 0
		return nil
	}

	// reduce pending counter every frame that is active
	rz.pendingCt--

	// do nothing if counter is not yet zero
	if rz.pendingCt > 0 {
		return nil
	}

	// return if there's nothing to do
	if rz.pendingBottom == state.frameInfo.VisibleBottom && rz.pendingBottom == state.frameInfo.VisibleTop {
		return nil
	}

	// sanity check before we do anything drastic
	if state.frameInfo.VisibleTop < state.frameInfo.VisibleBottom {

		// clamp top value if it is being changed. clamping makes sure that the
		// VisibleTop value is between the atari and new safe values
		if state.frameInfo.VisibleTop != rz.pendingTop {
			if rz.pendingTop < state.frameInfo.Spec.ExtendedVisibleTop {
				rz.pendingTop = state.frameInfo.Spec.ExtendedVisibleTop
			} else if rz.pendingTop > state.frameInfo.Spec.IdealVisibleTop {
				rz.pendingTop = state.frameInfo.Spec.IdealVisibleTop
			}
			state.frameInfo.VisibleTop = rz.pendingTop
		}

		// clamp bottom value if it is being changed. clamping makes sure that
		// the VisibleBottom value is between the atari and new safe values
		if state.frameInfo.VisibleBottom != rz.pendingBottom {
			if rz.pendingBottom > state.frameInfo.Spec.ExtendedVisibleBottom {
				rz.pendingBottom = state.frameInfo.Spec.ExtendedVisibleBottom
			} else if rz.pendingBottom < state.frameInfo.Spec.IdealVisibleBottom {
				rz.pendingBottom = state.frameInfo.Spec.IdealVisibleBottom
			}
			state.frameInfo.VisibleBottom = rz.pendingBottom
		}
	}

	return nil
}
