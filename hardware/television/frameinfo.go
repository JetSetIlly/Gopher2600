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
	"image"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// FrameInfo records the current frame information, as opposed to the optimal
// values of the specification.
type FrameInfo struct {
	// a copy of the television Spec that is considered to be in action. All
	// fields with the exception of ID, Colors and HorizontalScanRate may have
	// been superceded by values in the FrameInfo field. But they are good to
	// have for reference
	Spec specification.Spec

	// FrameNum can be used to distinguish one FrameInfo instance from another
	FrameNum int

	// the top and bottom scanlines that are to be present visually to the
	// player. this is generally related to the state of VBLANK but in the case
	// of screens when the VBLANK is never set, the visible area is determined by
	// the extent of non-black output
	//
	// consumers of FrameInfo should use these values rather than deriving that
	// information from VBLANK
	//
	// see the Crop() function for the preferred way of using these values to
	// create the a rectangle of the visible screen area. in particular, note
	// how the VisibleBottom value is treated in that context
	VisibleTop    int
	VisibleBottom int

	// the number of scanlines considered to be in the frame
	TotalScanlines int

	// the scanline which the screen starts at. ie. the scanline the TV flies
	// back to. is zero in most cases. if it is not zero then the screen is
	// recoving from a roll and is therefore not synced. see IsSynced() function
	TopScanline int

	// the top/bottom bounds of VBLANK. this is *not* the same as VisibleTop and
	// VisbleBottom, which takes into account ideal screen sizing and
	// situations where VBLANK is never set
	VBLANKtop    int
	VBLANKbottom int

	// the refresh rate. this value is derived from the number of scanlines
	// and is really a short-cut for:
	//
	//    Spec.HorizontalScanRate / TotalScanlines
	RefreshRate float32

	// has the TotalScanline field, and the RefreshRate field, changed since the
	// previous frame
	Jitter bool

	// whether the TV frame was begun as a result of a valid VSYNC signal
	FromVSYNC bool

	// VSYNCscanline is the scanline on which the VSYNC signal starts. not valid
	// if FromVSYNC is false
	VSYNCscanline int
	VSYNCcount    int

	// Stable is true once the television frame has been consistent for N
	// frames after reset. This is useful for pixel renderers that don't want
	// to show the loose frames that often occur after VCS hard-reset.
	//
	// once Stable is true then the Specification will not change (except
	// manually). This is important for ROMs that allow the VCS to run without
	// VSYNC - we don't want those ROMs to change the specifciation after he
	// startup period. A good example of such a ROM is Andrew Davie's 3e+ chess
	// demos.
	Stable bool

	// if the profile of the VSYNC signal has changed after the Stable flag has
	// been set then VSYNCunstable will be true
	VSYNCunstable bool

	// if the profile of the VBLANK bounds has changed after the Stable flag has
	// been set then VBLANKunstable will be true
	VBLANKunstable bool
}

// NewFrameInfo returns an initialised FrameInfo for the specification.
func NewFrameInfo(spec specification.Spec) FrameInfo {
	info := FrameInfo{
		Spec: spec,
	}
	info.reset()
	return info
}

func (info FrameInfo) String() string {
	return fmt.Sprintf("top: %d, bottom: %d, total: %d", info.VisibleTop, info.VisibleBottom, info.TotalScanlines)
}

// Crop returns an image.Rectangle for the cropped region of the screen. Using
// this is preferrable than using the VisibleTop/Bottom fields to construct the rectangle
//
// If the VisibleTop/Bottom fields are used in preference to this function for
// whatever reason, bear in mind that the VisibleBottom field should be adjusted
// by +1 in order to include the all visible scanlines in the rectangle
//
// To prove the need for this, consider what would happen if the screen was one
// scanline tall. In that case both the top and bottom values would be the same:
//
//	r := image.Rect(0, 10, 100, 10)
//
// The height of this rectangle will be zero, as shown by the Size() function
//
//	isZero := r.Size().Y == 0
func (info FrameInfo) Crop() image.Rectangle {
	return image.Rect(
		specification.ClksHBlank, info.VisibleTop,
		specification.ClksHBlank+specification.ClksVisible, info.VisibleBottom+1,
	)
}

// IsDifferent returns true if any of the pertinent display information is
// different between the two copies of FrameInfo
func (info FrameInfo) IsDifferent(cmp FrameInfo) bool {
	return info.Spec.ID != cmp.Spec.ID ||
		info.VisibleTop != cmp.VisibleTop ||
		info.VisibleBottom != cmp.VisibleBottom
}

func (info *FrameInfo) reset() {
	info.VisibleTop = info.Spec.IdealVisibleTop
	info.VisibleBottom = info.Spec.IdealVisibleBottom
	info.TotalScanlines = info.Spec.ScanlinesTotal
	info.RefreshRate = info.Spec.RefreshRate
	info.FromVSYNC = false
	info.Stable = false
}

// TotalClocks returns the total number of clocks required to generate the
// frame. The value returned assumes scanlines are complete - which may not be
// the case.
func (info FrameInfo) TotalClocks() int {
	return info.TotalScanlines * specification.ClksScanline
}

// VBLANKClocks returns the number of clocks in the VBLANK portion of the frame.
func (info FrameInfo) VBLANKClocks() int {
	return info.VisibleTop * specification.ClksScanline
}

// ScreenClocks returns the number of clocks in the visible portion of the frame.
func (info FrameInfo) ScreenClocks() int {
	return (info.VisibleBottom - info.VisibleTop) * specification.ClksScanline
}

// OverscanClocks returns the number of clocks in the Overscan portion of the frame.
func (info FrameInfo) OverscanClocks() int {
	return (info.TotalScanlines - info.VisibleBottom) * specification.ClksScanline
}

// AtariSage is true if VBLANK top/bottom are equal to the values suggested by
// Atari for the specification.
func (info FrameInfo) AtariSafe() bool {
	return info.VisibleTop == info.Spec.AtariSafeVisibleTop &&
		info.VisibleBottom == info.Spec.AtariSafeVisibleBottom
}

// IsSynced returns true if the frame is synchronised properly. The FromVSYNC
// field tells us that the frame was generated from a corrected VSYNC signal but
// the screen might still not be settled in a synchronised state.
func (info FrameInfo) IsSynced() bool {
	return info.TopScanline == 0 && info.FromVSYNC
}
