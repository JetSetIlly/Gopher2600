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

package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/imgui-go/v5"
)

type dbgScrMousePos struct {
	x int
	y int
}

type dbgScrMouse struct {
	// whether the mouse is inside the screen boundaries
	valid bool

	// coords of mouse
	pos dbgScrMousePos

	// number of pixels measured from top-left of screen
	offset int

	// scaled mouse coordinates. top-left corner is zero for uncropped screens.
	// cropped screens are adjusted as required
	//
	// use these values to index the reflection array, for example
	scaled dbgScrMousePos

	// mouse position adjusted so that clock and scanline represent the
	// underlying screen (taking cropped setting into account)
	//
	// (note that tv.Scanline is equal to scaled.y but that tv.Clock is
	// different to scaled.x. in the case of television coordinates the value
	// zero indicates the start of the visible screen and not the left most edge
	// of the HBLANK)
	tv coords.TelevisionCoords
}

func (m dbgScrMouse) String() string {
	return m.tv.String()
}

// pivot returns UV position of the mouse
func (m dbgScrMouse) pivot(v winDbgScrView) imgui.Vec2 {
	return imgui.Vec2{
		X: float32(m.pos.x) / v.scaledWidth,
		Y: float32(m.pos.y) / v.scaledHeight,
	}
}

func currentDbgScrMouse(scr *screen, view winDbgScrView) dbgScrMouse {
	pos := imgui.MousePos().Minus(view.screenOrigin)

	mouse := dbgScrMouse{}
	mouse.pos.x = int(pos.X)
	mouse.pos.y = int(pos.Y)

	// convert to UV
	u := pos.X / view.scaledWidth
	v := pos.Y / view.scaledHeight

	// zoom UV (pivot is in UV space already)
	u = view.pivot.X + (u-view.pivot.X)/view.scaledZoom()
	v = view.pivot.Y + (v-view.pivot.Y)/view.scaledZoom()

	// convert from UV back to pixels
	x := u * view.scaledWidth
	y := v * view.scaledHeight

	// scaled mouse position coordinates
	mouse.scaled.x = int(x / view.xscaling)
	mouse.scaled.y = int(y / view.yscaling)

	// corresponding clock and scanline values for scaled mouse coordinates
	mouse.tv.Clock = mouse.scaled.x
	mouse.tv.Scanline = mouse.scaled.y

	// frame field of the coordinates field is undefined in this context
	mouse.tv.Frame = coords.FrameIsUndefined

	// adjust depending on whether screen is cropped
	if view.cropped {
		mouse.scaled.x += specification.ClksHBlank
		mouse.scaled.y += scr.crit.frameInfo.VisibleTop
		mouse.tv.Scanline += scr.crit.frameInfo.VisibleTop
	} else {
		mouse.tv.Clock -= specification.ClksHBlank
	}

	// limit clock/scanline values after cropped adjustment
	mouse.tv.Clock = max(min(specification.ClksVisible-1, mouse.tv.Clock), -specification.ClksHBlank)
	mouse.tv.Scanline = max(min(scr.crit.frameInfo.TotalScanlines-1, mouse.tv.Scanline), 0)

	// offset is number of pixels from top-left of screen counting left-to-right
	// and top-to-bottom
	mouse.offset = mouse.scaled.x + mouse.scaled.y*specification.ClksScanline

	// check validity of mouse position
	mouse.valid = mouse.pos.x >= 0 && mouse.pos.y >= 0 &&
		mouse.offset >= 0 && mouse.offset < len(scr.crit.reflection)

	return mouse
}
