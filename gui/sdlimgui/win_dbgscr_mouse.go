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
	"fmt"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type dbgScrMousePoint struct {
	x int
	y int
}

type dbgScrMouse struct {
	// whether the mouse is inside the screen boundaries
	valid bool

	// imgui coords of mouse
	pos imgui.Vec2

	// offset of mouse from top-left of screen
	offset int

	// scaled mouse coordinates. top-left corner is zero for uncropped screens.
	// cropped screens are adjusted as required
	//
	// use these values to index the reflection array, for example
	scaled dbgScrMousePoint

	// mouse position adjusted so that clock and scanline represent the
	// underlying screen (taking cropped setting into account)
	clock    int
	scanline int
}

func (m dbgScrMouse) String() string {
	return fmt.Sprintf("Scanline: %d, Clock: %d", m.scanline, m.clock)
}

func (win *winDbgScr) mouseCoords() dbgScrMouse {
	mouse := dbgScrMouse{}

	mouse.pos = imgui.MousePos().Minus(win.screenOrigin)
	mouse.offset = win.mouse.scaled.x + win.mouse.scaled.y*specification.ClksScanline

	// outside bounds of window
	mouse.valid = win.mouse.pos.X >= 0.0 && win.mouse.pos.Y >= 0.0 && mouse.offset >= 0 && mouse.offset < len(win.scr.crit.reflection)
	if !mouse.valid {
		return mouse
	}

	// scaled mouse position coordinates
	mouse.scaled.x = int(mouse.pos.X / win.xscaling)
	mouse.scaled.y = int(mouse.pos.Y / win.yscaling)

	// corresponding clock and scanline values for scaled mouse coordinates
	mouse.clock = mouse.scaled.x
	mouse.scanline = mouse.scaled.y

	// adjust depending on whether screen is cropped (or in CRT Preview)
	if win.cropped || win.crtPreview {
		mouse.scaled.x += specification.ClksHBlank
		mouse.scaled.y += win.scr.crit.frameInfo.VisibleTop
		mouse.scanline += win.scr.crit.frameInfo.VisibleTop
	} else {
		mouse.clock -= specification.ClksHBlank
	}

	return mouse
}
