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

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// FrameInfo records the current frame information, as opposed to the optimal
// values of the specification, a copy of which is provided as reference.
type FrameInfo struct {
	Spec specification.Spec

	FrameNum int

	// the top and bottom scanlines that are to be present visually to the
	// player. this is generally related to the state of VBLANK but the
	// relationship is not as simple as it might seem
	//
	// consumers of FrameInfo should use these values rather than deriving that
	// information from VBLANK
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

	// a VSync frame is one which was generated from a valid VSYNC/VBLANK
	// sequence and which hasn't cause the update frequency of the television
	// to change.
	VSync bool

	// the number of scanlines in the VSync. value is not meaningful if VSync
	// is false
	VSyncScanlines int

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

func (info *FrameInfo) reset() {
	info.VisibleTop = info.Spec.AtariSafeVisibleTop
	info.VisibleBottom = info.Spec.AtariSafeVisibleBottom
	info.TotalScanlines = info.Spec.ScanlinesTotal
	info.RefreshRate = info.Spec.RefreshRate
	info.VSync = false
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
