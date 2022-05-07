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

	VisibleTop    int
	VisibleBottom int

	TotalScanlines int

	// the refresh rate. calculated from the TotalScanlines value
	RefreshRate float32

	// a VSynced frame is one which was generated from a valid VSYNC/VBLANK
	// sequence and which hasn't cause the update frequency of the television
	// to change.
	VSynced bool

	// Stable is true once the television frame has been consistent for N frames
	// after reset. This is useful for pixel renderers so that they don't show
	// the loose frames that often occur after VCS reset.
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
	info.VSynced = false
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
