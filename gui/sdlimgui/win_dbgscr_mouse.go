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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
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

func (win *winDbgScr) mouseFromVec2(pos imgui.Vec2) dbgScrMouse {
	mouse := dbgScrMouse{}
	mouse.pos.x = int(pos.X)
	mouse.pos.y = int(pos.Y)

	// scaled mouse position coordinates
	mouse.scaled.x = int(pos.X / win.xscaling)
	mouse.scaled.y = int(pos.Y / win.yscaling)

	// offset is number of pixels from top-left of screen counting left-to-right
	// and top-to-bottom
	mouse.offset = win.mouse.scaled.x + win.mouse.scaled.y*specification.ClksScanline

	// check validity of mouse position
	mouse.valid = win.mouse.pos.x >= 0.0 && win.mouse.pos.y >= 0.0 &&
		mouse.offset >= 0 && mouse.offset < len(win.scr.crit.reflection)

	// mouse position is not in debug screen area
	if !mouse.valid {
		return mouse
	}

	// corresponding clock and scanline values for scaled mouse coordinates
	mouse.tv.Clock = mouse.scaled.x
	mouse.tv.Scanline = mouse.scaled.y

	// frame field of the coordinates field is undefined in this context
	mouse.tv.Frame = coords.FrameIsUndefined

	// adjust depending on whether screen is cropped (or in CRT Preview)
	if win.cropped || win.crtPreview {
		mouse.scaled.x += specification.ClksHBlank
		mouse.scaled.y += win.scr.crit.frameInfo.VisibleTop
		mouse.tv.Scanline += win.scr.crit.frameInfo.VisibleTop
	} else {
		mouse.tv.Clock -= specification.ClksHBlank
	}

	return mouse
}

func (win *winDbgScr) currentMouse() dbgScrMouse {
	return win.mouseFromVec2(imgui.MousePos().Minus(win.screenOrigin))
}
