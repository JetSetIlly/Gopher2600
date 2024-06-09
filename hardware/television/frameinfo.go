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
// values of the specification, a copy of which is provided as reference.
type FrameInfo struct {
	Spec specification.Spec

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

	// the number of scanlines considered to be in the frame. the number of
	// scanlines that are actually in the frame may actually be less or more.
	// this can happen when a the refresh rate is changing, for example.
	//
	// note therefore, that the refresh rate can change but the reported number
	// of total scanlines not changing at the same time. the practical
	// consequence of this is that it is possible for there to be more
	// scanlines in the signals slice sent to the PixelRenderer via the
	// SetPixels() function
	TotalScanlines int

	// the refresh rate. this value is derived from the number of scanlines in
	// the frame but note that that may not be equal to the TotalScanlines
	// field
	RefreshRate float32

	// has the refresh rate changed since the previous frame
	Jitter bool

	// whether the TV is synchronised with the incoming TV signal
	IsSynced bool

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
	info.IsSynced = false
	info.Stable = false
}

// IsAtariSafe returns true if the current frame matches the AtariSafe values
// of the current specification.
func (info FrameInfo) IsAtariSafe() bool {
	return info.VisibleTop == info.Spec.AtariSafeVisibleTop && info.VisibleBottom == info.Spec.AtariSafeVisibleBottom
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
