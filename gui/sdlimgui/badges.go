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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v2"
)

func badgeBreakpointAnyBank(cols *Colors) {
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	radius := imgui.FrameHeight() / 6
	p.Y += imgui.TextLineHeight() * 0.75

	dl.AddCircle(p, radius, colorConvertFloat4ToU32(cols.BreakpointPC))
}

func badgeBreakpointThisBank(cols *Colors) {
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	radius := imgui.FrameHeight() / 6
	p.Y += imgui.TextLineHeight() * 0.75

	dl.AddCircleFilled(p, radius, colorConvertFloat4ToU32(cols.BreakpointPC))
}
